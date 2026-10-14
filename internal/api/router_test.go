package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/db"
)

// webDir is the internal/web directory relative to this test's working dir
// (internal/api/), which is where NewRouter points staticAssetServer.
const webDir = "../web"

// TestStaticAssetServer_ServesCSS pins L3: real CSS assets must still be
// served under /static.
func TestStaticAssetServer_ServesCSS(t *testing.T) {
	h := staticAssetServer(webDir)
	req := httptest.NewRequest("GET", "/styles.css", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /styles.css, got %d (body=%q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("expected a CSS content type, got %q", ct)
	}
}

// TestStaticAssetServer_DeniesHTMLTemplateSource pins L3: the .html template
// sources in internal/web must never be served under /static.
func TestStaticAssetServer_DeniesHTMLTemplateSource(t *testing.T) {
	h := staticAssetServer(webDir)
	for _, path := range []string{"/base.html", "/book-detail.html", "/partials/book-card.html"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %s (template source), got %d", path, rec.Code)
		}
	}
}

// TestStaticAssetServer_DeniesDirectoryListing pins L3: a directory request
// must 404, not auto-list the internal/web contents.
func TestStaticAssetServer_DeniesDirectoryListing(t *testing.T) {
	h := staticAssetServer(webDir)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for the directory root, got %d (body=%q)", rec.Code, rec.Body.String())
	}
}

// newTestRouter builds the real router over a fresh migrated in-memory DB so
// route-level auth wiring can be pinned without a live server. Templates are
// nil: the anonymous tests below only assert the redirect, which happens in
// RequireAuthHTML before any template is touched.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := goose.SetDialect("sqlite"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(database.DB, "../../migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	authSvc := auth.New(database.DB, []byte("test-session-secret-that-is-32-bytes!!"))
	authSvc.Store().Options.Secure = false
	return NewRouter(database, authSvc, &RouterConfig{})
}

// TestRouter_WishlistRequiresSessionAuth pins the /wishlist auth gap fix:
// the page route must be wrapped in RequireAuthHTML (Guest-or-Admin) exactly
// like /books and /reading-log, so an anonymous GET 302s to /login instead
// of rendering. Before the fix /wishlist rendered publicly with 200.
func TestRouter_WishlistRequiresSessionAuth(t *testing.T) {
	router := newTestRouter(t)

	for _, path := range []string{"/wishlist", "/books", "/reading-log"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("GET %s anonymous: expected 302, got %d — route is not session-auth guarded", path, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/login" {
			t.Fatalf("GET %s anonymous: expected redirect to /login, got %q", path, loc)
		}
	}
}

// TestRouter_BookSearchRequiresSessionAuth pins the /books/search auth gap
// fix: the FTS JSON search route must be wrapped in RequireAuthHTML
// (Guest-or-Admin) exactly like /books and /wishlist, so an anonymous GET
// 302s to /login instead of leaking titles/authors over the whole
// collection. Before the fix /books/search returned search JSON publicly.
func TestRouter_BookSearchRequiresSessionAuth(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest("GET", "/books/search?q=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("GET /books/search anonymous: expected 302, got %d — route is not session-auth guarded", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("GET /books/search anonymous: expected redirect to /login, got %q", loc)
	}
}
