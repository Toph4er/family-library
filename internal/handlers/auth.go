package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/middleware"
	"git.rcsmaine.com/chris/library/internal/theme"
)

// pageData holds template context for login-related pages.
type pageData struct {
	Year            int
	CSRFToken       string
	IsAdmin         bool
	IsAuthenticated bool
	ActiveTheme     theme.Theme
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
func RenderLoginPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
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
			Year:        time.Now().Year(),
			CSRFToken:   token,
			ActiveTheme: loadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if isHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		}
	}
}

// RenderGuestLoginPage renders the guest login page template.
// Already-authenticated users are redirected to /books.
func RenderGuestLoginPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
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
			Year:        time.Now().Year(),
			CSRFToken:   token,
			ActiveTheme: loadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if isHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "guest-login.html", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		}
	}
}

// RenderLogoutSuccess renders a simple "logged out" confirmation page.
func RenderLogoutSuccess(tmpl *template.Template, db *sql.DB, authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Perform the actual logout first
		_ = authSvc.Logout(w, r)
		data := pageData{
			Year:        time.Now().Year(),
			ActiveTheme: loadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if isHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "logout.html", data); err != nil {
				HTMXError(w, http.StatusInternalServerError)
			}
		}
	}
}

// htmlErrorFragment returns a styled HTML error div for HTMX swapping.
func htmlErrorFragment(message string) string {
	return "<div class=\"p-3 rounded-lg bg-error/10 border border-error/20 text-error text-sm\" role=\"alert\">" + template.HTMLEscapeString(message) + "</div>"
}

// HTMLLoginHandler handles admin login via HTMX (form-encoded, returns HTML).
// On success: sets HX-Redirect header to /books.
// On failure: returns an HTML error fragment for HTMX to swap into the DOM.
func HTMLLoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Username and password are required")))
			return
		}

		user, err := authSvc.Login(w, r, username, password)
		if err != nil {
			if apiErr, ok := err.(*auth.APIError); ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(apiErr.Code)
				_, _ = w.Write([]byte(htmlErrorFragment(apiErr.Message))) // #nosec G705 -- apiErr.Message is from internal auth service, not user input; htmlErrorFragment escapes output
				return
			}
			HTMXErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Success — tell HTMX to redirect to /books
		_ = user
		w.Header().Set("HX-Redirect", "/books")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLGuestLoginHandler handles guest login via HTMX (form-encoded, returns HTML).
// On success: sets HX-Redirect header to /books.
// On failure: returns an HTML error fragment for HTMX to swap into the DOM.
func HTMLGuestLoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		password := r.FormValue("password")

		if password == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Password is required")))
			return
		}

		if err := authSvc.GuestLogin(w, r, password); err != nil {
			if apiErr, ok := err.(*auth.APIError); ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(apiErr.Code)
				_, _ = w.Write([]byte(htmlErrorFragment(apiErr.Message))) // #nosec G705 -- apiErr.Message is from internal auth service, not user input; htmlErrorFragment escapes output
				return
			}
			HTMXErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Success — tell HTMX to redirect to /books
		w.Header().Set("HX-Redirect", "/books")
		w.WriteHeader(http.StatusOK)
	}
}
