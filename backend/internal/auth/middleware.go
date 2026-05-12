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

// GetUserFromContext retrieves the session user from request context
//
func GetUserFromContext(r *http.Request) *SessionUser {
	if user, ok := r.Context().Value(userContextKey).(*SessionUser); ok {
		return user
	}
	return nil
}
