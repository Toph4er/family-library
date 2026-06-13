package handlers

import (
	"net/http"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/middleware"
)

// CSRFTokenHandler handles GET /api/v1/csrf by returning the current CSRF
// token from the session, or generating a new one if none exists.
//
// Deprecated: No longer consumed by frontend (tokens come from {{.CSRFToken}}).
// Kept for test compatibility.
func CSRFTokenHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store := authSvc.Store()
		session, err := store.Get(r, auth.SessionID)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "session error")
			return
		}

		// Reuse existing token if present (don't rotate on GET).
		if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
			JSONResponse(w, http.StatusOK, map[string]interface{}{
				"csrf_token": token,
			})
			return
		}

		// Generate a new token.
		token, err := middleware.GenerateCSRFToken()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to generate CSRF token")
			return
		}
		session.Values[middleware.CSRFTokenKey] = token

		if err := session.Save(r, w); err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to set CSRF token")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"csrf_token": token,
		})
	}
}
