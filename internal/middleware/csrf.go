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
	"reflect"
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
// For unsafe methods the client token — taken from the X-CSRF-Token header
// (HTMX/fetch) or the csrf_token form field (plain form posts) — is compared
// (constant-time) against the token stored in the session.
//
// The token is NOT rotated on use: the value is baked into the page at render
// time and is stable for the session's lifetime.  Rotating per request would
// break multi-mutation pages (e.g. settings.html issuing several updates from
// a single load) because the client still holds the pre-rotation token.
//
// The validated session is placed into the request context so that downstream
// handlers can reuse the same session object.  After the handler completes the
// middleware saves the session ONLY if the handler mutated it — an
// unconditional save would re-emit the cookie with the store's default MaxAge
// (30 min), clobbering the longer lifetime set at login (L8).
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
				writeCSRFError(w)
				return
			}

			// — read stored token —
			sessionToken, ok := session.Values[CSRFTokenKey].(string)
			if !ok || sessionToken == "" {
				writeCSRFError(w)
				return
			}

			// — read client token (header for HTMX/fetch, form field for plain posts) —
			clientToken := r.Header.Get("X-CSRF-Token")
			if clientToken == "" {
				clientToken = r.FormValue("csrf_token")
			}
			if clientToken == "" {
				writeCSRFError(w)
				return
			}

			// — constant-time comparison —
			if subtle.ConstantTimeCompare([]byte(sessionToken), []byte(clientToken)) != 1 {
				writeCSRFError(w)
				return
			}

			// — share session with downstream handlers via context —
			r = SetSessionInContext(r, session)

			// Snapshot the session values so we can detect handler-side
			// mutations (e.g. a login handler storing auth fields) after the
			// handler runs.
			snapshot := make(map[interface{}]interface{}, len(session.Values))
			for k, v := range session.Values {
				snapshot[k] = v
			}

			// — process request —
			next.ServeHTTP(w, r)

			// — persist handler-side session mutations — only when the
			// session actually changed. An unconditional save would re-emit the
			// session cookie with the store's default MaxAge (30 min),
			// clobbering the longer lifetime set at login (L8: 30 days for
			// admin, 4 hours for guest).
			if !reflect.DeepEqual(snapshot, session.Values) {
				if err := session.Save(r, w); err != nil {
					slog.Error("csrf: failed to save session", "error", err)
				}
			}
		})
	}
}

// writeCSRFError writes a single, uniform 403 response for every CSRF failure
// (no session, no stored token, no client token, or mismatched token).
//
// L9: the body and status are identical for all failure modes so a client
// cannot distinguish "token absent" from "token invalid" — removing the probe
// oracle.
func writeCSRFError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "CSRF validation failed"})
}
