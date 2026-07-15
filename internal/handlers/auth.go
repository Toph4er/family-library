package handlers

import (
	"database/sql"
	"crypto/subtle"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"github.com/Toph4er/family-library/internal/auth"
	pages "github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/middleware"
	"github.com/Toph4er/family-library/internal/theme"
)

// pageData holds template context for login-related pages.

// validateLoginCSRF checks that the CSRF token from the request matches the
// token stored in the session.  The token is read from the X-CSRF-Token
// header (set by HTMX via hx-headers) or from the csrf_token form field.
// On mismatch it writes a 403 JSON error response.
// Returns true when the token is valid (or when no session cookie is present,
// meaning the login page was never rendered — in which case there is nothing
// to protect against).
func validateLoginCSRF(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, sessionName string) bool {
	// HTMX can send the token as a header or as a form field.
	headerToken := r.Header.Get("X-CSRF-Token")
	formToken := r.FormValue("csrf_token")
	token := headerToken
	if token == "" {
		token = formToken
	}

	// If no token was provided at all, check whether a session cookie exists.
	// No cookie means the login page was never loaded → no CSRF protection needed.
	// A cookie without a token means the page was loaded but something went wrong.
	if token == "" {
		session, err := store.Get(r, sessionName)
		if err != nil {
			// No session cookie at all — safe to proceed.
			return true
		}
		if session.IsNew {
			// No existing session cookie — the login page was never rendered.
			return true
		}
		// Session exists but no token — the page was rendered but the token
		// wasn't included in the form.  This shouldn't happen in normal flow.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"CSRF token missing"}`))
		return false
	}

	session, err := store.Get(r, sessionName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"CSRF token missing"}`))
		return false
	}

	sessionToken, ok := session.Values[middleware.CSRFTokenKey].(string)
	if !ok || sessionToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"CSRF token missing"}`))
		return false
	}

	if !constantTimeEqual(sessionToken, token) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"CSRF token invalid"}`))
		return false
	}

	return true
}

// constantTimeEqual performs a constant-time string comparison to prevent
// timing side-channel attacks on CSRF token validation.
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// pageData holds template context for login-related pages.
type pageData struct {
	Year            int
	CSRFToken       string
	Nonce           string
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
// Already-authenticated users are redirected to /dashboard.
func RenderLoginPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to dashboard
		if user := auth.GetUserFromContext(r); user != nil {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}

		// Also check session directly (context may not be populated for unauthenticated requests)
		session, err := store.Get(r, sessionName)
		if err == nil {
			if _, ok := session.Values[auth.UserIDKey]; ok {
				http.Redirect(w, r, "/dashboard", http.StatusFound)
				return
			}
		}

		token := getCSRFToken(w, store, sessionName, r)
		data := pageData{
			Year:        time.Now().Year(),
			CSRFToken:   token,
			Nonce:       middleware.GetCSPNonce(r),
			ActiveTheme: pages.LoadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if pages.IsHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
			}
		}
	}
}

// RenderGuestLoginPage renders the guest login page template.
// Already-authenticated users are redirected to /dashboard.
func RenderGuestLoginPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to dashboard
		if user := auth.GetUserFromContext(r); user != nil {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}

		// Also check session directly (context may not be populated for unauthenticated requests)
		session, err := store.Get(r, sessionName)
		if err == nil {
			if _, ok := session.Values[auth.UserIDKey]; ok {
				http.Redirect(w, r, "/dashboard", http.StatusFound)
				return
			}
		}

		token := getCSRFToken(w, store, sessionName, r)
		data := pageData{
			Year:        time.Now().Year(),
			CSRFToken:   token,
			Nonce:       middleware.GetCSPNonce(r),
			ActiveTheme: pages.LoadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if pages.IsHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "guest-login.html", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
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
			Nonce:       middleware.GetCSPNonce(r),
			ActiveTheme: pages.LoadActiveTheme(db),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if pages.IsHTMXRequest(r) {
			if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
			}
		} else {
			if err := tmpl.ExecuteTemplate(w, "logout.html", data); err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
			}
		}
	}
}

// HTMLLoginHandler handles admin login via HTMX (form-encoded, returns HTML).
// On success: sets HX-Redirect header to /dashboard.
// On failure: returns an HTML error fragment for HTMX to swap into the DOM.
func HTMLLoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		// Validate CSRF token from session (login pages generate one via
		// getCSRFToken).  This protects against CSRF-based account takeover.
		if !validateLoginCSRF(w, r, authSvc.Store(), auth.SessionID) {
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
			HTMXErrorResponse(w, r, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Success — tell HTMX to redirect to /dashboard
		_ = user
		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLGuestLoginHandler handles guest login via HTMX (form-encoded, returns HTML).
// On success: sets HX-Redirect header to /dashboard.
// On failure: returns an HTML error fragment for HTMX to swap into the DOM.
func HTMLGuestLoginHandler(authSvc *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		// Validate CSRF token from session (guest-login page generates one
		// via getCSRFToken).  This protects against CSRF-based account takeover.
		if !validateLoginCSRF(w, r, authSvc.Store(), auth.SessionID) {
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
			HTMXErrorResponse(w, r, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Success — tell HTMX to redirect to /dashboard
		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
	}
}
