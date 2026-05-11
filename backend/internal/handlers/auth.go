package handlers

import (
	"encoding/json"
	"net/http"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/models"
)

// LoginHandler handles admin login
func LoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		user, err := authSvc.Login(w, r, req.Username, req.Password)
		if err != nil {
			if apiErr, ok := err.(*auth.APIError); ok {
				JSONError(w, apiErr.Code, apiErr.Message)
				return
			}
			JSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id":         user.ID,
					"username":   user.Username,
					"role":       user.Role,
					"is_guest":   false,
				},
			},
		})
	}
}

// GuestLoginHandler handles guest login
func GuestLoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.GuestLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := authSvc.GuestLogin(w, r, req.Password); err != nil {
			if apiErr, ok := err.(*auth.APIError); ok {
				JSONError(w, apiErr.Code, apiErr.Message)
				return
			}
			JSONError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"is_guest": true,
					"role":     "guest",
				},
			},
		})
	}
}

// LogoutHandler handles logout
func LogoutHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authSvc.Logout(w, r); err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to logout")
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true})
	}
}

// MeHandler returns the current user info
func MeHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r)
		if user == nil {
			JSONError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"user": user,
			},
		})
	}
}
