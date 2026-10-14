package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Toph4er/family-library/internal/api"
)

// --- /books/search page-route auth pinning -------------------------------
//
// GET /books/search is the FTS5 JSON search backing the books page. It used
// to be anonymous, so anyone could enumerate titles/authors over the whole
// collection. It must now behave exactly like /books and /wishlist:
// anonymous → 302 /login, Guest-or-Admin → pass the wrapper.

// TestBookSearchRoute_AnonymousRedirectsToLogin pins the fix through the
// REAL router: GET /books/search without a session must 302 to /login.
func TestBookSearchRoute_AnonymousRedirectsToLogin(t *testing.T) {
	env := setupTestEnv(t)
	router := api.NewRouter(env.db, env.auth, &api.RouterConfig{})

	req := httptest.NewRequest("GET", "/books/search?q=trip", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("GET /books/search anonymous: expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("GET /books/search anonymous: expected redirect to /login, got %q", loc)
	}
}

// TestBookSearchRoute_GuestGets200 pins that a Guest session passes the
// RequireAuthHTML wrapper and the JSON search still works end-to-end
// through the real router.
func TestBookSearchRoute_GuestGets200(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")
	router := api.NewRouter(env.db, env.auth, &api.RouterConfig{})

	cookie := guestLoginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/books/search?q=trip", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("guest GET /books/search: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
