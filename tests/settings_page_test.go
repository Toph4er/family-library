// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/handlers"
)

// settingsPageFuncs returns the template functions used by settings.html and
// base.html. They mirror the funcMap registered in cmd/library/server.go —
// only the three custom functions the settings page invokes (formatTime,
// humanizeKey, guestVisibilityFields); eq/ne/index are Go builtins.
func settingsPageFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(s string, tzName string) string {
			if tzName == "" {
				tzName = "America/New_York"
			}
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return s
			}
			// Try space-separated format first
			t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
			if err != nil {
				// Try ISO 8601 with Z suffix (e.g. 2026-07-14T20:37:00Z)
				t, err = time.Parse("2006-01-02T15:04:05Z", s)
				if err != nil {
					// Try ISO 8601 without Z
					t, err = time.Parse("2006-01-02T15:04:05", s)
					if err != nil {
						return s
					}
				}
			}
			return t.In(loc).Format("Jan 2, 2006 at 3:04 PM")
		},
		// Humanize setting keys for display
		"humanizeKey": func(key string) string {
			parts := strings.Split(key, "_")
			for i, p := range parts {
				if len(p) > 0 {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return strings.Join(parts, " ")
		},
		// Guest visibility fields in a consistent order
		"guestVisibilityFields": func() []string {
			return []string{
				"title", "subtitle", "authors", "illustrators",
				"publisher", "publication_year", "page_count", "quantity", "book_type",
				"reading_levels", "genres", "themes", "awards",
				"gift_from", "gift_relationship", "child_rating", "read_count",
				"cover_image_url", "cover_source",
				"isbn", "condition", "location", "notes",
				"date_received", "last_read_date",
			}
		},
	}
}

// loadSettingsPageTemplate parses base.html + settings.html + the user-row
// partial the same way the app's loadTemplates does, so the settings page
// handler can be exercised end-to-end.
func loadSettingsPageTemplate(t *testing.T) *template.Template {
	t.Helper()
	rel := func(name string) string {
		return filepath.Join("..", "internal", "web", name)
	}
	tmpl, err := template.New("").Funcs(settingsPageFuncs()).ParseFiles(
		rel("base.html"),
		rel("settings.html"),
		rel("partials/user-row.html"),
	)
	if err != nil {
		t.Fatalf("failed to parse settings page templates: %v", err)
	}
	return tmpl
}

// TestSeedAdminUser_SetsDisplayName is a regression test for the fresh-install
// seeding: the seeded admin row must carry a sane display_name (previously it
// was NULL, which the settings page then crashed on).
func TestSeedAdminUser_SetsDisplayName(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	runMigrations(t, database)

	authSvc := auth.New(database.DB, []byte("test-session-secret-that-is-32-bytes!!"))
	if err := authSvc.SeedAdminUser(testUsername, testPassword); err != nil {
		t.Fatalf("SeedAdminUser failed: %v", err)
	}

	var displayName sql.NullString
	err = database.QueryRow("SELECT display_name FROM users WHERE username = ?", testUsername).Scan(&displayName)
	if err != nil {
		t.Fatalf("failed to query seeded admin: %v", err)
	}
	if !displayName.Valid || displayName.String != testUsername {
		t.Fatalf("expected seeded admin display_name=%q, got valid=%v value=%q", testUsername, displayName.Valid, displayName.String)
	}

	// Seeding must remain idempotent — a second call is a no-op.
	if err := authSvc.SeedAdminUser(testUsername, testPassword); err != nil {
		t.Fatalf("second SeedAdminUser call failed: %v", err)
	}
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("expected exactly 1 user after re-seed, got %d (err=%v)", count, err)
	}
}

// TestHTMLSettingsPage_NullDisplayName is a regression test for the
// fresh-install 500: SeedAdminUser used to insert the admin without a
// display_name, and the settings page scanned that NULL into a plain string,
// failing with "converting NULL to string is unsupported". The handler must
// render 200 when a users row has a NULL display_name.
func TestHTMLSettingsPage_NullDisplayName(t *testing.T) {
	env := setupTestEnv(t)

	// Precondition: the seeded admin row must have a NULL display_name —
	// the exact fresh-install state that triggered the 500.
	var displayName sql.NullString
	err := env.db.QueryRow("SELECT display_name FROM users WHERE username = ?", testUsername).Scan(&displayName)
	if err != nil {
		t.Fatalf("failed to query seeded admin: %v", err)
	}
	if displayName.Valid {
		t.Fatalf("precondition: seeded admin should have NULL display_name, got %q", displayName.String)
	}

	tmpl := loadSettingsPageTemplate(t)

	r := chi.NewRouter()
	r.Handle("/settings", env.auth.RequireAdminHTML(
		handlers.RenderSettingsPage(tmpl, env.db.DB, env.auth.Store(), auth.SessionID),
	))

	cookie := loginAndGetCookie(t, env)
	req := httptest.NewRequest("GET", "/settings", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 with NULL display_name (fresh install), got %d: %s", rec.Code, rec.Body.String())
	}

	// The page must list the seeded user.
	if !strings.Contains(rec.Body.String(), testUsername) {
		t.Fatalf("expected settings page to list user %q", testUsername)
	}

	// Sanity: the seeded user's row data round-trips (username, role).
	var username, role string
	err = env.db.QueryRow("SELECT username, role FROM users WHERE username = ?", testUsername).Scan(&username, &role)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if fmt.Sprint(username) != testUsername || role != "admin" {
		t.Fatalf("unexpected user row: %q / %q", username, role)
	}
}
