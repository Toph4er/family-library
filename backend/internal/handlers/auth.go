package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
	"git.rcsmaine.com/chris/library/backend/internal/models"
)

// pageData holds template context for login-related pages.
type pageData struct {
	Year      int
	CSRFToken string
}

// getCSRFToken retrieves or generates a CSRF token for the given request.
// It first checks the context (set by CSRF middleware), then falls back to
// loading the session from the cookie store. If no token exists, it generates
// a new one and saves the session to set the cookie.
func getCSRFToken(w http.ResponseWriter, store *sessions.CookieStore, sessionName string, r *http.Request) string {
	// Check context first (set by CSRF middleware)
	if s := middleware.GetSessionFromContext(r); s != nil {
		if token, ok := s.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
			return token
		}
	}

	// Load session from cookie store
	session, err := store.Get(r, sessionName)
	if err != nil {
		return ""
	}

	// Reuse existing token if present
	if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
		return token
	}

	// Generate a new token
	token, err := middleware.GenerateCSRFToken()
	if err != nil {
		return ""
	}
	session.Values[middleware.CSRFTokenKey] = token
	if err := session.Save(r, w); err != nil {
		return ""
	}
	return token
}

// RenderLoginPage renders the admin login page template.
func RenderLoginPage(tmpl *template.Template, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := getCSRFToken(w, store, sessionName, r)
		data := pageData{
			Year:      time.Now().Year(),
			CSRFToken: token,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderGuestLoginPage renders the guest login page template.
func RenderGuestLoginPage(tmpl *template.Template, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := getCSRFToken(w, store, sessionName, r)
		data := pageData{
			Year:      time.Now().Year(),
			CSRFToken: token,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "guest-login.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderLogoutSuccess renders a simple "logged out" confirmation page.
func RenderLogoutSuccess(tmpl *template.Template, authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Perform the actual logout first
		_ = authSvc.Logout(w, r)
		data := pageData{
			Year: time.Now().Year(),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "logout.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

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
