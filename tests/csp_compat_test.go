package tests

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Toph4er/family-library/internal/handlers"
)

// CSP-compatibility regression tests for the v1.0.4 (M6) regression that
// broke the mobile nav and the reading-log book-selector modal.
//
// The M6 CSP removed 'unsafe-eval' and switched Alpine to the @alpinejs/csp
// build, which silently kills two template idioms:
//
//  1. The @alpinejs/csp build cannot parse x-* expressions containing
//     operators, literals, or calls (e.g. x-data="{ open: false }",
//     x-show="!open", @click="open = !open"). Only bare dotted identifier
//     paths are supported. Unparseable expressions fail-closed (warn + no-op),
//     so the mobile menu stayed hidden and its hamburger never rendered.
//
//  2. htmx compiles every hx-on:* attribute value with new Function(), which
//     CSP without 'unsafe-eval' blocks; htmx catches the EvalError and the
//     handler silently no-ops. This killed modal close buttons, book-result
//     taps, and the modal form field toggles.
//
// The fix moves ALL such behavior to nonce'd / delegated plain-JS handlers
// keyed on data-attributes, stable ids, and classes. These tests pin the
// templates and the server-rendered HTMX fragments against any reintroduction
// of inline script-eval handlers.

var (
	reHxOnAttr   = regexp.MustCompile(`hx-on[:A-Za-z0-9_-]*=`)
	reInlineOn   = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	reAlpineExpr = regexp.MustCompile(`(?:x-data|x-show|x-if|x-model|x-transition:[a-z-]+|x-on:[a-z-]+|@click|@change)="[^"]+"`)
)

// baseRenderData is a minimal template context for rendering base.html
// standalone in tests (mirrors the fields pageContext provides).
func baseRenderData() map[string]interface{} {
	return map[string]interface{}{
		"Nonce":           "testnonce",
		"CSRFToken":       "testcsrf",
		"Year":            "2026",
		"IsAuthenticated": true,
		"IsAdmin":         true,
		"IsGuest":         false,
		"UserTimezone":    "UTC",
		"ActiveTheme": map[string]interface{}{
			"CSSOverrideBlock": template.CSS(""),
			"FooterTagline":    "test tagline",
		},
	}
}

func TestBaseTemplate_CSPClean(t *testing.T) {
	tmpl, err := template.ParseFiles(filepath.Join("..", "internal", "web", "base.html"))
	if err != nil {
		t.Fatalf("parse base.html: %v", err)
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "base", baseRenderData()); err != nil {
		t.Fatalf("execute base: %v", err)
	}
	out := buf.String()

	if loc := reHxOnAttr.FindString(out); loc != "" {
		t.Errorf(`base.html renders %q; hx-on:* needs 'unsafe-eval' and silently no-ops without it`, loc)
	}
	if loc := reInlineOn.FindString(out); loc != "" {
		t.Errorf("base.html renders inline event attribute %q; blocked by CSP script-src", loc)
	}
	if loc := reAlpineExpr.FindString(out); loc != "" {
		t.Errorf("base.html renders Alpine expression %q; the @alpinejs/csp build only parses bare dotted paths — use delegated JS instead", loc)
	}

	// The plain-JS mobile menu contract: the hamburger and the mobile-menu
	// links must carry the data-attributes the nonce'd toggle script binds.
	if !strings.Contains(out, "data-nav-toggle") {
		t.Error(`base.html mobile menu hamburger must carry data-nav-toggle (bound by the CSP-safe toggle script)`)
	}
	if !strings.Contains(out, "data-nav-close") {
		t.Error(`base.html mobile-menu links must carry data-nav-close so the menu closes on navigation`)
	}
}

func TestWebTemplates_NoInlineScriptHandlers(t *testing.T) {
	root := filepath.Join("..", "internal", "web")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		if loc := reHxOnAttr.FindString(src); loc != "" {
			t.Errorf("%s contains %q (hx-on:* requires 'unsafe-eval')", path, loc)
		}
		if loc := reInlineOn.FindString(src); loc != "" {
			t.Errorf("%s contains inline event attribute %q (blocked by CSP script-src)", path, loc)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

// renderFragment serves req through handler mounted on a chi router so
// chi.URLParam works, and returns the response body.
func renderFragment(t *testing.T, method, pattern, path string, handler http.HandlerFunc) string {
	t.Helper()
	r := chi.NewRouter()
	r.Method(method, pattern, handler)
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s -> %d: %s", method, path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestServerRenderedHTMXFragments_NoInlineScriptHandlers(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "Goodnight Moon", "M. Wise Brown")

	// A family member + reading log so the edit-form handlers render.
	if _, err := env.db.Exec(`INSERT INTO family_members (name, relation) VALUES ('Emma', 'daughter')`); err != nil {
		t.Fatalf("seed family member: %v", err)
	}
	if _, err := env.db.Exec(`INSERT INTO reading_logs (book_id, start_page, end_page, total_pages, entire_book, reader_name) VALUES (1, 1, 10, 20, 0, 'Emma')`); err != nil {
		t.Fatalf("seed reading log: %v", err)
	}

	fragments := []struct {
		name    string
		method  string
		pattern string
		path    string
		handler http.HandlerFunc
	}{
		{"book-selector initial", "GET", "/reading-logs/book-selector", "/reading-logs/book-selector", handlers.HTMLBookSelectorHandler(env.db.DB)},
		{"book-selector search", "GET", "/reading-logs/book-selector", "/reading-logs/book-selector?q=goodnight", handlers.HTMLBookSelectorHandler(env.db.DB)},
		{"reading-log new-form", "GET", "/reading-logs/{book_id}/new-form", "/reading-logs/1/new-form", handlers.HTMLReadingLogFormHandler(env.db.DB)},
		{"reading-log edit-form", "GET", "/reading-logs/{id}/edit-form", "/reading-logs/1/edit-form", handlers.HTMLReadingLogEditFormHandler(env.db.DB)},
		{"user form new", "GET", "/settings/users/new-form", "/settings/users/new-form", handlers.HTMLUserFormHandler(env.db.DB)},
		{"user form edit", "GET", "/settings/users/{id}/edit", "/settings/users/1/edit", handlers.HTMLUserFormHandler(env.db.DB)},
		{"family form new", "GET", "/settings/family-members/new-form", "/settings/family-members/new-form", handlers.HTMLFamilyMemberFormHandler(env.db)},
	}

	for _, f := range fragments {
		t.Run(f.name, func(t *testing.T) {
			body := renderFragment(t, f.method, f.pattern, f.path, f.handler)
			if loc := reHxOnAttr.FindString(body); loc != "" {
				t.Errorf("%s contains %q (hx-on:* silently no-ops under the M6 CSP)", f.name, loc)
			}
			if loc := reInlineOn.FindString(body); loc != "" {
				t.Errorf("%s contains inline event attribute %q", f.name, loc)
			}
		})
	}
}

// TestBookSelector_ItemsExposeDataBookID pins the data-attribute contract the
// CSP-safe delegated click handler binds (see openLogForm in
// web/reading-log.html "scripts"): every result link — initial list and
// search results alike — must carry its book id.
func TestBookSelector_ItemsExposeDataBookID(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "Goodnight Moon", "M. Wise Brown")

	initial := renderFragment(t, "GET", "/reading-logs/book-selector", "/reading-logs/book-selector", handlers.HTMLBookSelectorHandler(env.db.DB))
	if !strings.Contains(initial, `data-book-id="1"`) {
		t.Errorf("initial book-selector result must carry data-book-id=\"1\":\n%s", initial)
	}

	search := renderFragment(t, "GET", "/reading-logs/book-selector", "/reading-logs/book-selector?q=goodnight", handlers.HTMLBookSelectorHandler(env.db.DB))
	if !strings.Contains(search, `data-book-id="1"`) {
		t.Errorf("search book-selector result must carry data-book-id=\"1\":\n%s", search)
	}
}
