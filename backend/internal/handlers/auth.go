package handlers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
)

// pageData holds template context for login-related pages.
type pageData struct {
	Year            int
	CSRFToken       string
	IsAdmin         bool
	IsAuthenticated bool
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
// Already-authenticated users are redirected to /books.
func RenderLoginPage(tmpl *template.Template, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to books
		if user := auth.GetUserFromContext(r); user != nil {
			http.Redirect(w, r, "/books", http.StatusFound)
			return
		}

		// Also check session directly (context may not be populated for unauthenticated requests)
		session, err := store.Get(r, sessionName)
		if err == nil {
			if _, ok := session.Values[auth.UserIDKey]; ok {
				http.Redirect(w, r, "/books", http.StatusFound)
				return
			}
		}

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
// Already-authenticated users are redirected to /books.
func RenderGuestLoginPage(tmpl *template.Template, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to books
		if user := auth.GetUserFromContext(r); user != nil {
			http.Redirect(w, r, "/books", http.StatusFound)
			return
		}

		// Also check session directly (context may not be populated for unauthenticated requests)
		session, err := store.Get(r, sessionName)
		if err == nil {
			if _, ok := session.Values[auth.UserIDKey]; ok {
				http.Redirect(w, r, "/books", http.StatusFound)
				return
			}
		}

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
		if err := r.ParseForm(); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			JSONError(w, http.StatusBadRequest, "Username and password are required")
			return
		}

		user, err := authSvc.Login(w, r, username, password)
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
		if err := r.ParseForm(); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		password := r.FormValue("password")
		if password == "" {
			JSONError(w, http.StatusBadRequest, "Password is required")
			return
		}

		if err := authSvc.GuestLogin(w, r, password); err != nil {
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
