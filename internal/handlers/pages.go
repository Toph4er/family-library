// Package handlers provides HTTP request handlers for the family library application.
//
// Page rendering has been moved to the pages sub-package. This file acts as a thin
// shim layer so existing route registrations (e.g., in api/router.go) continue
// to reference handlers.RenderXxxPage without needing changes.
package handlers

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"

	pages "github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/services"
)

// --- Page renderers (delegated to pages package) ---

func RenderLandingPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return pages.RenderLandingPage(tmpl, db, store, sessionName)
}

func RenderDashboardPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	dashSvc := services.NewDashboardService(db)
	return pages.RenderDashboardPage(tmpl, dashSvc, store, sessionName)
}

func RenderBooksPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return pages.RenderBooksPage(tmpl, db, store, sessionName)
}

func RenderBookDetailPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return pages.RenderBookDetailPage(tmpl, db, store, sessionName)
}

func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return pages.RenderWishlistPage(tmpl, db, store, sessionName)
}

func RenderWishlistFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, itemID int64) http.HandlerFunc {
	return pages.RenderWishlistFormPage(tmpl, db, store, sessionName, isEdit, itemID)
}

func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return pages.RenderSettingsPage(tmpl, db, store, sessionName)
}

func RenderBookFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, bookID int64) http.HandlerFunc {
	return pages.RenderBookFormPage(tmpl, db, store, sessionName, isEdit, bookID)
}

// --- Test helpers (re-exported from pages package) ---

// PageContextForTest is the exported BaseContext struct for test access.
type PageContextForTest = pages.BaseContext

// BuildPageContextForTest calls buildBaseContext and returns the context
// needed for CSRF token verification and theme loading in tests.
func BuildPageContextForTest(r *http.Request, store *sessions.CookieStore, sessionName string) PageContextForTest {
	return pages.BuildPageContextForTest(r, store, sessionName)
}
