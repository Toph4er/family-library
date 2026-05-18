// Package middleware provides HTTP middleware for security, logging,
// rate limiting, and error recovery.
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
)

// CSRFTokenKey is the session key used to store the CSRF token.
const CSRFTokenKey = "csrf_token"

// sessionCtxKey is the unexported context key for the shared session object.
type sessionCtxKey struct{}

// SetSessionInContext stores a session in the request context so downstream
// handlers can reuse the same session object instead of re-reading from cookies.
func SetSessionInContext(r *http.Request, session *sessions.Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, session))
}

// GetSessionFromContext retrieves a session from the request context.
// Returns nil when the CSRF middleware is not in the chain.
func GetSessionFromContext(r *http.Request) *sessions.Session {
	if s, ok := r.Context().Value(sessionCtxKey{}).(*sessions.Session); ok {
		return s
	}
	return nil
}

// GenerateCSRFToken returns a cryptographically secure token: 32 random bytes,
// hex-encoded to a 64-character string.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CSRFProtection returns middleware that validates CSRF tokens for unsafe HTTP
// methods (POST, PUT, PATCH, DELETE).
//
// Safe methods (GET, HEAD, OPTIONS) pass through without validation.
//
// For unsafe methods the X-CSRF-Token header is compared (constant-time) against
// the token stored in the session.  On success the token is rotated and the
// new value is written to the X-CSRF-Token response header.
//
// The validated session is placed into the request context so that downstream
// handlers (e.g. auth login) can reuse the same session object.  After the
// handler completes the middleware saves the session — ensuring both the
// handler's changes and the rotated token are persisted together in a single
// Set-Cookie header.
func CSRFProtection(store *sessions.CookieStore, sessionName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// — skip safe methods —
			switch strings.ToUpper(r.Method) {
			case "GET", "HEAD", "OPTIONS":
				next.ServeHTTP(w, r)
				return
			}

			// — load session —
			session, err := store.Get(r, sessionName)
			if err != nil {
				writeJSONError(w, http.StatusForbidden, "CSRF token missing")
				return
			}

			// — read stored token —
			sessionToken, ok := session.Values[CSRFTokenKey].(string)
			if !ok || sessionToken == "" {
				writeJSONError(w, http.StatusForbidden, "CSRF token missing")
				return
			}

			// — read header token —
			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" {
				writeJSONError(w, http.StatusForbidden, "CSRF token missing")
				return
			}

			// — constant-time comparison —
			if subtle.ConstantTimeCompare([]byte(sessionToken), []byte(headerToken)) != 1 {
				writeJSONError(w, http.StatusForbidden, "CSRF token invalid")
				return
			}

			// — rotate token —
			newToken, err := GenerateCSRFToken()
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			session.Values[CSRFTokenKey] = newToken

			// — share session with downstream handlers via context —
			r = SetSessionInContext(r, session)

			// — expose new token to the client —
			w.Header().Set("X-CSRF-Token", newToken)

			// — process request —
			next.ServeHTTP(w, r)

			// — persist session (rotated token + any handler mutations) —
			if err := session.Save(r, w); err != nil {
				slog.Error("csrf: failed to save session", "error", err)
			}
		})
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
