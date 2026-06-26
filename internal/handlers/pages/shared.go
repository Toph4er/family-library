// Package pages provides template rendering for all page handlers.
// Each page gets its own file, and shared infrastructure lives here.
package pages

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/middleware"
	"github.com/Toph4er/family-library/internal/theme"
)

// sensitiveKeys lists setting keys that must never be exposed in list responses.
var SensitiveKeys = map[string]struct{}{
	"guest_password_hash": {},
}

// BuildPageContextForTest calls buildBaseContext and returns the context
// needed for CSRF token verification and theme loading in tests.
func BuildPageContextForTest(r *http.Request, store *sessions.CookieStore, sessionName string) BaseContext {
	return buildBaseContext(r, store, sessionName, nil)
}

// renderHTMXError writes an HTTP error status code for HTMX requests.
// HTMX only cares about the status code — no body is sent.
func renderHTMXError(w http.ResponseWriter, r *http.Request, status int) {
	if status >= 500 {
		slog.Error("server error", "method", r.Method, "path", r.URL.Path, "status", status)
	}
	w.WriteHeader(status)
}

// BaseContext holds common template data shared across all page handlers.
type BaseContext struct {
	Year            int
	CSRFToken       string
	Nonce           string
	IsAdmin         bool
	IsAuthenticated bool
	IsGuest         bool
	Username        string
	SiteName        string
	SiteTagline     string
	ActiveTheme     theme.Theme
	AvailableThemes []theme.Theme
	ThemeColorsJSON template.HTML
}

// buildBaseContext creates a BaseContext for the given request.
// It first checks the request context (set by auth middleware), then falls
// back to reading the session directly for routes without that middleware.
func buildBaseContext(r *http.Request, store *sessions.CookieStore, sessionName string, db *sql.DB) BaseContext {
	ctx := BaseContext{Year: time.Now().Year()}
	ctx.Nonce = middleware.GetCSPNonce(r)

	// Check context first (set by auth middleware on protected routes).
	if user := auth.GetUserFromContext(r); user != nil {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = !user.IsGuest
		ctx.IsGuest = user.IsGuest
		ctx.Username = user.Username
		if s := middleware.GetSessionFromContext(r); s != nil {
			if token, ok := s.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
				ctx.CSRFToken = token
			}
		} else {
			// Session not in context (e.g. GET request through RequireAuthHTML).
			// Read from store to extract the CSRF token.
			if session, err := store.Get(r, sessionName); err == nil {
				if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
					ctx.CSRFToken = token
				}
			}
		}
		ctx.ActiveTheme = loadActiveTheme(db)
		return ctx
	}

	// Fall back to reading the session directly (public routes without auth middleware).
	session, err := store.Get(r, sessionName)
	if err != nil {
		return ctx
	}
	if _, ok := session.Values[auth.UserIDKey].(int64); ok {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = true
		if username, ok := session.Values[auth.UsernameKey].(string); ok {
			ctx.Username = username
		}
	} else if isGuest, ok := session.Values[auth.IsGuestKey].(bool); ok && isGuest {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = false
		ctx.IsGuest = true
	}

	if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
		ctx.CSRFToken = token
	}

	ctx.ActiveTheme = loadActiveTheme(db)

	return ctx
}

// LoadActiveTheme reads the active_theme setting and returns the resolved Theme.
// Falls back to WoodlandFairytale if the setting is missing, unknown, or db is nil.
func LoadActiveTheme(db *sql.DB) theme.Theme {
	return loadActiveTheme(db)
}

// IsHTMXRequest checks if the request originated from HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return isHTMXRequest(r)
}

// BuildBaseContext creates a BaseContext for the given request.
// It first checks the request context (set by auth middleware), then falls
// back to reading the session directly for routes without that middleware.
func BuildBaseContext(r *http.Request, store *sessions.CookieStore, sessionName string, db *sql.DB) BaseContext {
	return buildBaseContext(r, store, sessionName, db)
}

// RenderPage renders a template, returning only the content fragment for HTMX
// requests (HX-Request header present) or the full page layout otherwise.
func RenderPage(w http.ResponseWriter, r *http.Request, tmpl *template.Template, pageName string, data interface{}) {
	renderPage(w, r, tmpl, pageName, data)
}

// loadActiveTheme reads the active_theme setting and returns the resolved Theme.
// Falls back to WoodlandFairytale if the setting is missing, unknown, or db is nil.
func loadActiveTheme(db *sql.DB) theme.Theme {
	if db == nil {
		return theme.WoodlandFairytale()
	}
	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "active_theme").Scan(&val)
	if err != nil || val == "" {
		return theme.WoodlandFairytale()
	}
	return theme.GetThemeByID(val)
}

// isHTMXRequest checks if the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// renderPage renders a template, returning only the content fragment for HTMX
// requests (HX-Request header present) or the full page layout otherwise.
func renderPage(w http.ResponseWriter, r *http.Request, tmpl *template.Template, pageName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHTMXRequest(r) {
		// HTMX request — render only the content block, not the base layout.
		if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
			slog.Error("template error", "page", pageName, "error", err)
			renderHTMXError(w, r, http.StatusInternalServerError)
		}
	} else {
		// Full page request — render the page template (which includes base).
		if err := tmpl.ExecuteTemplate(w, pageName, data); err != nil {
			slog.Error("template error", "page", pageName, "error", err)
			renderHTMXError(w, r, http.StatusInternalServerError)
		}
	}
}

// derefString returns the value of a string pointer, or "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefInt returns the value of an int pointer, or 0 if nil.
func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// buildThemeColorsJSON builds a JSON map of theme ID → {bg, text}
// from the available themes, for server-side rendering into the
// switchTheme() JS in settings.html.
func buildThemeColorsJSON(themes []theme.Theme) template.HTML {
	result := "{"
	for i, t := range themes {
		if i > 0 {
			result += ","
		}
		// #nosec G203 -- values come from application-controlled theme definitions, not user input
		result += fmt.Sprintf(`"%s":{"bg":"%s","text":"%s"}`, t.ID, t.Background, t.Text)
	}
	result += "}"
	// #nosec G203 -- values come from application-controlled theme definitions
	return template.HTML(result)
}

// svgIcon returns a 24x24 inline SVG as template.HTML.
func svgIcon(name string) template.HTML {
	const (
		iconBook          = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>`
		iconOpenBook      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>`
		iconStar          = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1" aria-hidden="true"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>`
		iconClipboardList = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M9 14l2 2 4-4"/></svg>`
		iconCalendar      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>`
		iconUsers         = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
		iconTag           = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>`
		iconBarChart      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="12" y1="20" x2="12" y2="10"/><line x1="18" y1="20" x2="18" y2="4"/><line x1="6" y1="20" x2="6" y2="16"/></svg>`
		iconHeart         = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>`
		iconSparkles      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z"/></svg>`
	)
	switch name {
	case "book":
		return template.HTML(iconBook) // #nosec G203 -- hardcoded SVG, not user input
	case "open-book":
		return template.HTML(iconOpenBook) // #nosec G203 -- hardcoded SVG, not user input
	case "star":
		return template.HTML(iconStar) // #nosec G203 -- hardcoded SVG, not user input
	case "clipboard-list":
		return template.HTML(iconClipboardList) // #nosec G203 -- hardcoded SVG, not user input
	case "calendar":
		return template.HTML(iconCalendar) // #nosec G203 -- hardcoded SVG, not user input
	case "users":
		return template.HTML(iconUsers) // #nosec G203 -- hardcoded SVG, not user input
	case "tag":
		return template.HTML(iconTag) // #nosec G203 -- hardcoded SVG, not user input
	case "bar-chart":
		return template.HTML(iconBarChart) // #nosec G203 -- hardcoded SVG, not user input
	case "heart":
		return template.HTML(iconHeart) // #nosec G203 -- hardcoded SVG, not user input
	case "sparkles":
		return template.HTML(iconSparkles) // #nosec G203 -- hardcoded SVG, not user input
	default:
		return ""
	}
}
