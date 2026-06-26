package pages

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
)

// RenderLandingPage renders the public landing page.
// Authenticated users are redirected to /books.
func RenderLandingPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		// Authenticated users should not see the landing page — send them to dashboard.
		if ctx.IsAuthenticated {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}

		siteName := "Our Library"
		siteTagline := "A woodland fairy tale collection"

		var nameVal, taglineVal sql.NullString
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "site_name").Scan(&nameVal); err == nil {
			siteName = nameVal.String
		}
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "site_tagline").Scan(&taglineVal); err == nil {
			siteTagline = taglineVal.String
		}

		ctx.SiteName = siteName
		ctx.SiteTagline = siteTagline

		renderPage(w, r, tmpl, "landing.html", ctx)
	}
}
