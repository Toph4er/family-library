package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/handlers"
)

func TestLoginHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"username":"admin","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if !resp["success"].(bool) {
		t.Fatalf("expected success=true, got false")
	}

	data := resp["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	if user["username"] != "admin" {
		t.Fatalf("expected username=admin, got %v", user["username"])
	}
	if user["role"] != "admin" {
		t.Fatalf("expected role=admin, got %v", user["role"])
	}
	if user["is_guest"] != false {
		t.Fatalf("expected is_guest=false, got %v", user["is_guest"])
	}

	// Verify session cookie was set
	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("expected session cookie to be set")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"username":"admin","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Fatal("expected success=false")
	}
}

func TestLoginHandler_NonExistentUser(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"username":"nobody","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandler_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogoutHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// First login to get a session cookie
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	handlers.LogoutHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Fatal("expected success=true")
	}
}

func TestLogoutHandler_NoSession(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	handlers.LogoutHandler(env.auth).ServeHTTP(rec, req)

	// Logout should still succeed even without an existing session
	// (it creates an empty session and sets MaxAge=-1)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMeHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Build a mini router with RequireAuth middleware
	r := chi.NewRouter()
	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		env.auth.RequireAuth(http.HandlerFunc(handlers.MeHandler(env.auth))).ServeHTTP(w, r)
	})

	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/me", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	user := data["user"].(*auth.SessionUser)
	if user.Username != "admin" {
		t.Fatalf("expected username=admin, got %v", user.Username)
	}
}

func TestMeHandler_NotAuthenticated(t *testing.T) {
	env := setupTestEnv(t)

	r := chi.NewRouter()
	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		env.auth.RequireAuth(http.HandlerFunc(handlers.MeHandler(env.auth))).ServeHTTP(w, r)
	})

	req := httptest.NewRequest("GET", "/me", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
