// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/Toph4er/family-library/internal/api"
	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/middleware"
)

// newTestSessionStore returns a fresh cookie store (Secure off) for
// middleware-level tests that don't need the full auth service.
func newTestSessionStore() *sessions.CookieStore {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	store := sessions.NewCookieStore(key)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   30 * 60,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	return store
}

// makeSessionCookie creates a session (with an optional CSRF token) and
// returns its Set-Cookie value.
func makeSessionCookie(t *testing.T, store *sessions.CookieStore, name, token string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	sess, err := store.New(req, name)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if token != "" {
		sess.Values[middleware.CSRFTokenKey] = token
	}
	rec := httptest.NewRecorder()
	if err := sess.Save(req, rec); err != nil {
		t.Fatalf("session.Save: %v", err)
	}
	cookie := rec.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("no Set-Cookie from session save")
	}
	return cookie
}

// TestCSRFProtection_UniformFailureResponses pins L9(b): every CSRF failure
// mode (no session, no stored token, no client token, wrong token) must
// return an IDENTICAL status + body, so an attacker cannot probe whether a
// token is "absent" vs "invalid".
func TestCSRFProtection_UniformFailureResponses(t *testing.T) {
	store := newTestSessionStore()
	token, err := middleware.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	cookie := makeSessionCookie(t, store, "csrf_test_session", token)

	r := chi.NewRouter()
	r.Use(middleware.CSRFProtection(store, "csrf_test_session"))
	r.Handle("/mutate", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	post := func(mutate func(*http.Request)) (int, string) {
		req := httptest.NewRequest("POST", "/mutate", nil)
		if mutate != nil {
			mutate(req)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	// 1. No session cookie at all.
	code1, body1 := post(nil)
	// 2. Session exists (with token), client sends no token.
	code2, body2 := post(func(req *http.Request) { req.Header.Set("Cookie", cookie) })
	// 3. Session exists, client sends a WRONG token.
	code3, body3 := post(func(req *http.Request) {
		req.Header.Set("Cookie", cookie)
		req.Header.Set("X-CSRF-Token", "0000000000000000000000000000000000000000000000000000000000000000")
	})

	if code1 != http.StatusForbidden || code2 != http.StatusForbidden || code3 != http.StatusForbidden {
		t.Fatalf("expected 403 for all failure modes, got %d / %d / %d", code1, code2, code3)
	}
	if body1 != body2 || body2 != body3 {
		t.Fatalf("CSRF failure bodies differ — probe oracle:\n  no-session:  %q\n  no-token:    %q\n  bad-token:   %q", body1, body2, body3)
	}

	// Valid token (header) must still pass.
	code4, _ := post(func(req *http.Request) {
		req.Header.Set("Cookie", cookie)
		req.Header.Set("X-CSRF-Token", token)
	})
	if code4 != http.StatusOK {
		t.Fatalf("valid header token: expected 200, got %d", code4)
	}

	// Valid token (form field, plain HTML form post) must also pass.
	formBody := strings.NewReader("csrf_token=" + token + "&x=1")
	req := httptest.NewRequest("POST", "/mutate", formBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid form-field token: expected 200, got %d", rec.Code)
	}

	// Safe methods pass without any token.
	for _, method := range []string{"GET", "OPTIONS"} {
		req := httptest.NewRequest(method, "/mutate", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s without token: expected 200, got %d", method, rec.Code)
		}
	}
}

// TestCSRFProtection_NoUnnecessarySessionSave pins the L8/M2 interaction:
// the middleware must NOT re-save the session when the handler didn't mutate
// it. An unconditional save re-encodes the cookie with the store's default
// MaxAge (30 min) and would silently truncate admin (30-day, L8) and guest
// (4-hour) sessions on every CSRF-validated request. It must still save when
// the handler DID mutate the session.
func TestCSRFProtection_NoUnnecessarySessionSave(t *testing.T) {
	store := newTestSessionStore()
	token, err := middleware.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	cookie := makeSessionCookie(t, store, "csrf_test_session", token)

	r := chi.NewRouter()
	r.Use(middleware.CSRFProtection(store, "csrf_test_session"))
	r.Post("/stable", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK) // does not touch the session
	})
	r.Post("/mutate", func(w http.ResponseWriter, req *http.Request) {
		if s := middleware.GetSessionFromContext(req); s != nil {
			s.Values["test_key"] = "test_value"
		}
		w.WriteHeader(http.StatusOK)
	})

	doPost := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, nil)
		req.Header.Set("Cookie", cookie)
		req.Header.Set("X-CSRF-Token", token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	rec := doPost("/stable")
	if rec.Code != http.StatusOK {
		t.Fatalf("/stable: expected 200, got %d", rec.Code)
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if setCookie != "" {
		t.Fatalf("/stable: unexpected Set-Cookie %q — unconditional session save would reset the cookie's MaxAge to the store default (30 min) and truncate L8's 30-day admin sessions", setCookie)
	}

	rec = doPost("/mutate")
	if rec.Code != http.StatusOK {
		t.Fatalf("/mutate: expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("/mutate: handler mutated the session but no Set-Cookie was written — mutation would be lost")
	}
}

// TestRouter_HTMLMutationRoutes_RequireCSRF pins M2: the state-changing HTML
// routes in the real router must be wrapped with the CSRF middleware (403
// without a token) while a valid token passes through to the handler.
func TestRouter_HTMLMutationRoutes_RequireCSRF(t *testing.T) {
	env := setupTestEnv(t)
	router := api.NewRouter(env.db, env.auth, &api.RouterConfig{})

	token, err := middleware.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// Admin session cookie carrying a CSRF token (same shape the login flow
	// produces: token from the login page render + auth fields from Login).
	req0 := httptest.NewRequest("GET", "/", nil)
	sess, err := env.auth.Store().New(req0, auth.SessionID)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	sess.Values[middleware.CSRFTokenKey] = token
	sess.Values[auth.UserIDKey] = int64(1)
	sess.Values[auth.RoleKey] = "admin"
	sess.Values[auth.UsernameKey] = testUsername
	sess.Values[auth.IsGuestKey] = false
	rec0 := httptest.NewRecorder()
	if err := sess.Save(req0, rec0); err != nil {
		t.Fatalf("session.Save: %v", err)
	}
	cookie := rec0.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("no session cookie from save")
	}

	doPost := func(method, path string, withToken bool) (int, string) {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Cookie", cookie)
		if withToken {
			req.Header.Set("X-CSRF-Token", token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	// No token → 403 "CSRF validation failed" (uniform body from L9b).
	for _, tc := range []struct{ method, path string }{
		{"POST", "/books/create"},
		{"POST", "/settings/users"},
	} {
		code, body := doPost(tc.method, tc.path, false)
		if code != http.StatusForbidden {
			t.Fatalf("%s %s without token: expected 403, got %d (route is not CSRF-protected)", tc.method, tc.path, code)
		}
		if !strings.Contains(body, "CSRF validation failed") {
			t.Fatalf("%s %s without token: unexpected body %q", tc.method, tc.path, body)
		}
	}

	// Valid token → passes CSRF; the handler then rejects the empty form
	// (400) — anything other than the CSRF 403 proves the token was accepted.
	code, body := doPost("POST", "/books/create", true)
	if code == http.StatusForbidden && strings.Contains(body, "CSRF validation failed") {
		t.Fatalf("POST /books/create with valid token was rejected by CSRF middleware: %d %q", code, body)
	}
	if code != http.StatusBadRequest {
		t.Logf("POST /books/create with valid token: status %d (expected 400 from handler validation)", code)
	}
}
