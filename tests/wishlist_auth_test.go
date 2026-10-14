// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Toph4er/family-library/internal/api"
	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/handlers"
)

// --- Wishlist page auth pinning -----------------------------------------
//
// /wishlist used to be rendered with no session-auth wrapper, so anonymous
// visitors could read it. It must behave exactly like /books and
// /reading-log: anonymous → 302 /login, Guest-or-Admin → 200.

// TestWishlistPage_AnonymousRedirectsToLogin pins the fix through the REAL
// router: GET /wishlist without a session must 302 to /login, identical in
// shape to the anonymous behavior of /books and /reading-log. The empty
// RouterConfig is safe here — the redirect happens before any template is
// touched.
func TestWishlistPage_AnonymousRedirectsToLogin(t *testing.T) {
	env := setupTestEnv(t)
	router := api.NewRouter(env.db, env.auth, &api.RouterConfig{})

	// /wishlist itself, plus its two page-form siblings, for the record.
	for _, path := range []string{"/wishlist", "/books", "/reading-log"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("GET %s anonymous: expected 302, got %d", path, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/login" {
			t.Fatalf("GET %s anonymous: expected redirect to /login, got %q", path, loc)
		}
	}
}

// wishlistPageFuncs mirrors the subset of the app's funcMap (see
// cmd/library/server.go) that base.html + wishlist.html + its partials
// invoke. Same approach as settingsPageFuncs in settings_page_test.go.
func wishlistPageFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(s string, tzName string) string {
			if tzName == "" {
				tzName = "America/New_York"
			}
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return s
			}
			t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
			if err != nil {
				t, err = time.Parse("2006-01-02T15:04:05Z", s)
				if err != nil {
					t, err = time.Parse("2006-01-02T15:04:05", s)
					if err != nil {
						return s
					}
				}
			}
			return t.In(loc).Format("Jan 2, 2006 at 3:04 PM")
		},
		"humanizeKey": func(key string) string {
			parts := strings.Split(key, "_")
			for i, p := range parts {
				if len(p) > 0 {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return strings.Join(parts, " ")
		},
		"guestVisibilityFields": func() []string {
			return []string{"title", "isbn"}
		},
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"join": func(sep string, parts []string) string {
			return strings.Join(parts, sep)
		},
		"formatISBN": func(isbn interface{}) string {
			switch v := isbn.(type) {
			case string:
				return v
			case *string:
				if v == nil {
					return ""
				}
				return *v
			}
			return ""
		},
		"split": func(s interface{}) []string {
			if str, ok := s.(string); ok {
				return strings.Split(str, ",")
			}
			return []string{}
		},
		"dict": func(args ...interface{}) map[string]interface{} {
			result := make(map[string]interface{}, len(args)/2)
			for i := 0; i+1 < len(args); i += 2 {
				key, _ := args[i].(string)
				result[key] = args[i+1]
			}
			return result
		},
	}
}

// loadWishlistPageTemplate parses base.html + wishlist.html + every partial,
// the way the app's loadTemplates does (mirrors loadSettingsPageTemplate).
func loadWishlistPageTemplate(t *testing.T) *template.Template {
	t.Helper()
	dir := filepath.Join("..", "internal", "web")
	files := []string{
		filepath.Join(dir, "base.html"),
		filepath.Join(dir, "wishlist.html"),
	}
	partials, err := filepath.Glob(filepath.Join(dir, "partials", "*.html"))
	if err != nil {
		t.Fatalf("glob partials: %v", err)
	}
	files = append(files, partials...)
	tmpl, err := template.New("").Funcs(wishlistPageFuncs()).ParseFiles(files...)
	if err != nil {
		t.Fatalf("failed to parse wishlist page templates: %v", err)
	}
	return tmpl
}

// guestLoginAndGetCookie logs in as the guest (same direct-handler shape as
// TestHTMLGuestLoginHandler_Success: no prior session cookie, so the login
// CSRF check is correctly skipped) and returns the session cookie.
func guestLoginAndGetCookie(t *testing.T, env *testEnv) string {
	t.Helper()

	form := strings.NewReader("password=" + testGuestPassword)
	req := httptest.NewRequest("POST", "/auth/guest-login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("guest login failed: status %d, body: %s", rec.Code, rec.Body.String())
	}
	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("no session cookie in guest login response")
	}
	return cookie
}

// TestWishlistPage_AuthenticatedUsersGet200 pins that both a Guest session
// and an Admin session pass the RequireAuthHTML wrapper and the wishlist
// page renders (full template chain, no template errors).
func TestWishlistPage_AuthenticatedUsersGet200(t *testing.T) {
	env := setupTestEnv(t)
	tmpl := loadWishlistPageTemplate(t)

	// Seed one wishlist item so the page has content to render.
	if _, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "Pinned Book", 3); err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}

	h := env.auth.RequireAuthHTML(
		handlers.RenderWishlistPage(tmpl, env.db.DB, env.auth.Store(), auth.SessionID),
	)

	for name, cookie := range map[string]string{
		"guest": guestLoginAndGetCookie(t, env),
		"admin": loginAndGetCookie(t, env),
	} {
		req := httptest.NewRequest("GET", "/wishlist", nil)
		req.Header.Set("Cookie", cookie)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s GET /wishlist: expected 200, got %d: %s", name, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Pinned Book") {
			t.Fatalf("%s GET /wishlist: expected rendered page to contain wishlist item, body missing it (len=%d)", name, rec.Body.Len())
		}
	}
}
