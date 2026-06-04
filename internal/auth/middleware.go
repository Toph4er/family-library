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
//
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
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
//
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok || user.IsGuest {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
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

// RequireAdminHTML is an HTML-aware variant of RequireAdmin that redirects
// non-admin users to / instead of returning a JSON error.
// Guests and unauthenticated users are both redirected.
// Use this for admin-only page routes.
func (a *Auth) RequireAdminHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.GetUserFromSession(r)
		if !ok || user.IsGuest {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Attach user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the session user from request context
//
func GetUserFromContext(r *http.Request) *SessionUser {
	if user, ok := r.Context().Value(userContextKey).(*SessionUser); ok {
		return user
	}
	return nil
}
