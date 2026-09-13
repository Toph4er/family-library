package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

// contextKey is a custom type for request context keys
type contextKey string

const userContextKey contextKey = "user"

// RequireAuth middleware ensures the user is authenticated
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Authentication required",
			})
			return
		}

		// Attach user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin middleware ensures the user is an admin
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok || user.IsGuest {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Admin access required",
			})
			return
		}

		// Attach user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAuthHTML is an HTML-aware variant of RequireAuth that redirects
// unauthenticated users to /login instead of returning a JSON error.
// Use this for page routes (template-rendered endpoints).
func (a *Auth) RequireAuthHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Attach user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAdminHTML is an HTML-aware variant of RequireAdmin that returns a
// 403 Forbidden response for non-admin users (L10: HTML authz failures answer
// 403, matching the API layer's RequireAdmin, instead of a 302 redirect to /).
// Guests and unauthenticated users are both rejected.
// Use this for admin-only page routes.
func (a *Auth) RequireAdminHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok || user.IsGuest {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("<!doctype html><title>403 Forbidden</title><h1>403 Forbidden</h1>"))
			return
		}

		// Attach user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the session user from request context
func GetUserFromContext(r *http.Request) *SessionUser {
	if user, ok := r.Context().Value(userContextKey).(*SessionUser); ok {
		return user
	}
	return nil
}
