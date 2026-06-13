// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/handlers"
)

// buildAdminHTMLRouter creates a chi router that wraps the given handler with
// RequireAdminHTML middleware (redirects instead of JSON for HTML endpoints).
func buildAdminHTMLRouter(t *testing.T, env *testEnv, method, path string, handler http.HandlerFunc) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(path, env.auth.RequireAdminHTML(handler))
	return r
}

// ---------- HTML Login Handler ----------

func TestHTMLLoginHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("username", testUsername)
	form.Set("password", testPassword)

	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should set HX-Redirect header for HTMX
	if redirect := rec.Header().Get("HX-Redirect"); redirect != "/books" {
		t.Fatalf("expected HX-Redirect=/books, got %q", redirect)
	}

	// Verify session cookie was set
	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("expected session cookie to be set")
	}
}

func TestHTMLLoginHandler_WrongPassword(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("username", testUsername)
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "Invalid credentials") {
		t.Fatalf("expected error message in body: %s", rec.Body.String())
	}
}

func TestHTMLLoginHandler_MissingFields(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("username", testUsername)
	// No password

	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- HTML Guest Login Handler ----------

func TestHTMLGuestLoginHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("password", testGuestPassword)

	req := httptest.NewRequest("POST", "/auth/guest-login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if redirect := rec.Header().Get("HX-Redirect"); redirect != "/books" {
		t.Fatalf("expected HX-Redirect=/books, got %q", redirect)
	}

	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("expected session cookie to be set")
	}
}

func TestHTMLGuestLoginHandler_WrongPassword(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest("POST", "/auth/guest-login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLGuestLoginHandler_MissingPassword(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/auth/guest-login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- HTML Create Book Handler ----------

func TestHTMLCreateBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "Test Book")
	form.Set("authors", "Test Author")
	form.Set("isbn", "978-0-06-112008-4")

	req := httptest.NewRequest("POST", "/books/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/books/create", handlers.HTMLCreateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify HX-Redirect header is present
	if loc := rec.Header().Get("HX-Redirect"); loc == "" {
		t.Fatalf("expected HX-Redirect header, got none: %s", rec.Body.String())
	}

	// Verify book was created in DB
	var count int
	err := env.db.QueryRow("SELECT COUNT(*) FROM books WHERE title = ?", "Test Book").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 book created, got %d", count)
	}

	// Verify ISBN was normalized
	var storedISBN string
	err = env.db.QueryRow("SELECT isbn FROM books WHERE title = ?", "Test Book").Scan(&storedISBN)
	if err != nil || storedISBN != "9780061120084" {
		t.Fatalf("expected normalized ISBN, got %q", storedISBN)
	}
}

func TestHTMLCreateBookHandler_MissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("authors", "Test Author")
	// No title

	req := httptest.NewRequest("POST", "/books/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/books/create", handlers.HTMLCreateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLCreateBookHandler_DuplicateISBN(t *testing.T) {
	env := setupTestEnv(t)

	// Pre-insert a book with an ISBN
	_, err := env.db.Exec("INSERT INTO books (isbn, title) VALUES (?, ?)", "9780061120084", "Existing Book")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	form := url.Values{}
	form.Set("title", "Duplicate Book")
	form.Set("isbn", "978-0-06-112008-4")

	req := httptest.NewRequest("POST", "/books/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/books/create", handlers.HTMLCreateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLCreateBookHandler_NotAuthenticated(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "Test Book")

	req := httptest.NewRequest("POST", "/books/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/books/create", handlers.HTMLCreateBookHandler(env.db))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// RequireAdminHTML redirects unauthenticated users to /
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect (302), got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

// ---------- HTML Update Book Handler ----------

func TestHTMLUpdateBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	result, err := env.db.Exec("INSERT INTO books (title, authors) VALUES (?, ?)", "Old Title", "Old Author")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("isbn", "9780743273565")
	form.Set("authors", "New Author")

	req := httptest.NewRequest("POST", "/books/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))

	r := buildAdminHTMLRouter(t, env, "POST", "/books/{id}/update", handlers.HTMLUpdateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify HX-Redirect header is present
	if loc := rec.Header().Get("HX-Redirect"); loc == "" {
		t.Fatalf("expected HX-Redirect header, got none: %s", rec.Body.String())
	}

	// Verify update persisted
	var title, authors string
	err = env.db.QueryRow("SELECT title, authors FROM books WHERE id = ?", id).Scan(&title, &authors)
	if err != nil {
		t.Fatalf("failed to query book: %v", err)
	}
	if title != "New Title" {
		t.Fatalf("expected title='New Title', got %q", title)
	}
	if authors != "New Author" {
		t.Fatalf("expected authors='New Author', got %q", authors)
	}
}

func TestHTMLUpdateBookHandler_ClearFieldToNull(t *testing.T) {
	env := setupTestEnv(t)

	result, err := env.db.Exec("INSERT INTO books (title, authors, notes) VALUES (?, ?, ?)", "Test Book", "Author", "Some notes")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	// Send empty string for notes to clear it to NULL
	form := url.Values{}
	form.Set("title", "Test Book")
	form.Set("isbn", "9780743273565")
	form.Set("notes", "")

	req := httptest.NewRequest("POST", "/books/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))

	r := buildAdminHTMLRouter(t, env, "POST", "/books/{id}/update", handlers.HTMLUpdateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify HX-Redirect header is present
	if loc := rec.Header().Get("HX-Redirect"); loc == "" {
		t.Fatalf("expected HX-Redirect header, got none: %s", rec.Body.String())
	}

	// Verify notes is NULL
	var notes sql.NullString
	err = env.db.QueryRow("SELECT notes FROM books WHERE id = ?", id).Scan(&notes)
	if err != nil {
		t.Fatalf("failed to query book: %v", err)
	}
	if notes.Valid {
		t.Fatalf("expected notes to be NULL, got %q", notes.String)
	}
}

func TestHTMLUpdateBookHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("isbn", "9780743273565")

	req := httptest.NewRequest("POST", "/books/999/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", "999")

	r := buildAdminHTMLRouter(t, env, "POST", "/books/{id}/update", handlers.HTMLUpdateBookHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- HTML Update Setting Handler ----------

func TestHTMLUpdateSettingHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("value", "amazon")

	req := httptest.NewRequest("PUT", "/settings/update/cover_image_provider", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "key", "cover_image_provider")

	r := buildAdminHTMLRouter(t, env, "PUT", "/settings/update/{key}", handlers.HTMLUpdateSettingHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the update persisted
	var value string
	err := env.db.QueryRow("SELECT value FROM settings WHERE key = ?", "cover_image_provider").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query setting: %v", err)
	}
	if value != "amazon" {
		t.Fatalf("expected value='amazon', got %q", value)
	}
}

func TestHTMLUpdateSettingHandler_SensitiveKey(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("value", "hacked")

	req := httptest.NewRequest("PUT", "/settings/update/guest_password_hash", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "key", "guest_password_hash")

	r := buildAdminHTMLRouter(t, env, "PUT", "/settings/update/{key}", handlers.HTMLUpdateSettingHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLUpdateSettingHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("value", "test")

	req := httptest.NewRequest("PUT", "/settings/update/nonexistent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "key", "nonexistent")

	r := buildAdminHTMLRouter(t, env, "PUT", "/settings/update/{key}", handlers.HTMLUpdateSettingHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- HTML Update Guest Visibility Handler ----------

func TestHTMLUpdateGuestVisibilityHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Handler expects JSON body: {"field": "...", "visible": true/false}
	body := `{"field":"title","visible":true}`
	req := httptest.NewRequest("POST", "/settings/guest-visibility/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	r := buildAdminHTMLRouter(t, env, "POST", "/settings/guest-visibility/update", handlers.HTMLUpdateGuestVisibilityHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the setting was updated in default_guest_visibility (JSON blob)
	var value string
	err := env.db.QueryRow("SELECT value FROM settings WHERE key = ?", "default_guest_visibility").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query setting: %v", err)
	}
	if !strings.Contains(value, `"title":true`) {
		t.Fatalf("expected title:true in visibility blob, got %q", value)
	}
}

func TestHTMLUpdateGuestVisibilityHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/settings/guest-visibility/update", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")

	r := buildAdminHTMLRouter(t, env, "POST", "/settings/guest-visibility/update", handlers.HTMLUpdateGuestVisibilityHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- Middleware: RequireAdminHTML redirects ----------

func TestRequireAdminHTML_RedirectsGuest(t *testing.T) {
	env := setupTestEnv(t)

	// Login as guest
	form := url.Values{}
	form.Set("password", testGuestPassword)

	req := httptest.NewRequest("POST", "/auth/guest-login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	guestCookie := getSessionCookie(rec)
	if guestCookie == "" {
		t.Fatal("expected guest session cookie")
	}

	// Try to access an admin-only HTML endpoint
	r := buildAdminHTMLRouter(t, env, "GET", "/settings/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not see this"))
	})

	req = httptest.NewRequest("GET", "/settings/test", nil)
	req.Header.Set("Cookie", guestCookie)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect (302), got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

// ---------- Auth helper: RequireAuth/RequireAdmin ----------

func TestRequireAuth_AllowsAuthenticated(t *testing.T) {
	env := setupTestEnv(t)

	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		env.auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})).ServeHTTP(w, r)
	})

	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestRequireAuth_RejectsUnauthenticated(t *testing.T) {
	env := setupTestEnv(t)

	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		env.auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})).ServeHTTP(w, r)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAdmin_RejectsNonAdmin(t *testing.T) {
	env := setupTestEnv(t)

	// Login as guest first
	form := url.Values{}
	form.Set("password", testGuestPassword)

	req := httptest.NewRequest("POST", "/auth/guest-login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handlers.HTMLGuestLoginHandler(env.auth).ServeHTTP(rec, req)

	guestCookie := getSessionCookie(rec)

	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		env.auth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})).ServeHTTP(w, r)
	})

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", guestCookie)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

// ---------- CSRF Token Handler ----------

func TestCSRFTokenHandler(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/csrf", nil)
	rec := httptest.NewRecorder()

	handlers.CSRFTokenHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should return a CSRF token in the JSON body
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if _, ok := resp["csrf_token"]; !ok {
		t.Fatalf("expected csrf_token in response body, got: %s", rec.Body.String())
	}
}

// ---------- HTML User Handlers ----------

func TestHTMLCreateUserHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("username", "newuser")
	form.Set("password", "newpass123")
	form.Set("role", "admin")
	form.Set("display_name", "New User")

	req := httptest.NewRequest("POST", "/settings/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/settings/users", handlers.HTMLCreateUserHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify user was created
	var count int
	err := env.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "newuser").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 user created, got %d", count)
	}
}

func TestHTMLCreateUserHandler_DuplicateUsername(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("username", "admin") // Already exists
	form.Set("password", "pass123")
	form.Set("role", "admin")

	req := httptest.NewRequest("POST", "/settings/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/settings/users", handlers.HTMLCreateUserHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLDeleteUserHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Create a user to delete
	hash, _ := auth.HashPassword("pass123")
	result, err := env.db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		"todelete", hash, "admin",
	)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	id, _ := result.LastInsertId()

	// Use the actual ID in the URL path so chi extracts the correct value
	req := httptest.NewRequest("DELETE", "/settings/users/"+fmt.Sprintf("%d", id), nil)

	r := buildAdminHTMLRouter(t, env, "DELETE", "/settings/users/{id}", handlers.HTMLDeleteUserHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify user was deleted
	var count int
	err = env.db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", id).Scan(&count)
	if err != nil || count != 0 {
		t.Fatalf("expected user (id=%d) to be deleted, got count=%d", id, count)
	}
}

func TestHTMLDeleteUserHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest("DELETE", "/settings/users/999", nil)
	req = setURLParam(req, "id", "999")

	r := buildAdminHTMLRouter(t, env, "DELETE", "/settings/users/{id}", handlers.HTMLDeleteUserHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- HTML Wishlist Handlers ----------

func TestHTMLCreateWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "Wishlist Test Book")
	form.Set("author", "Test Author")
	form.Set("isbn", "9780061120084")
	form.Set("reason", "Gift idea")
	form.Set("priority", "4")
	form.Set("amazon_url", "https://www.amazon.com/dp/ABC123")
	form.Set("thriftbooks_url", "https://www.thriftbooks.com/w/xyz")
	form.Set("cover_image_url", "https://example.com/cover.jpg")
	form.Set("notes", "Some notes")

	req := httptest.NewRequest("POST", "/wishlist/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/create", handlers.HTMLCreateWishlistItemHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if loc := rec.Header().Get("HX-Redirect"); loc != "/wishlist" {
		t.Fatalf("expected HX-Redirect=/wishlist, got %q", loc)
	}

	// Verify item was created in DB with store links
	var title, reason, amazonURL, thriftbooksURL, coverImageURL, notesStr string
	var priority int
	var author, isbn sql.NullString
	err := env.db.QueryRow("SELECT title, author, isbn, reason, priority, amazon_url, thriftbooks_url, cover_image_url, notes FROM wishlist WHERE title = ?", "Wishlist Test Book").Scan(
		&title, &author, &isbn, &reason, &priority, &amazonURL, &thriftbooksURL, &coverImageURL, &notesStr,
	)
	if err != nil {
		t.Fatalf("failed to query wishlist item: %v", err)
	}
	if title != "Wishlist Test Book" {
		t.Fatalf("expected title='Wishlist Test Book', got %q", title)
	}
	if priority != 4 {
		t.Fatalf("expected priority=4, got %d", priority)
	}
	if amazonURL != "https://www.amazon.com/dp/ABC123" {
		t.Fatalf("expected amazon_url, got %q", amazonURL)
	}
	if thriftbooksURL != "https://www.thriftbooks.com/w/xyz" {
		t.Fatalf("expected thriftbooks_url, got %q", thriftbooksURL)
	}
}

func TestHTMLCreateWishlistItemHandler_MissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("author", "Test Author")
	// No title

	req := httptest.NewRequest("POST", "/wishlist/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/create", handlers.HTMLCreateWishlistItemHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLCreateWishlistItemHandler_NotAuthenticated(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "Test Book")

	req := httptest.NewRequest("POST", "/wishlist/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/create", handlers.HTMLCreateWishlistItemHandler(env.db))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// RequireAdminHTML redirects unauthenticated users to /
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect (302), got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

func TestHTMLUpdateWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "Old Title", 3)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	form := url.Values{}
	form.Set("title", "Updated Title")
	form.Set("author", "Updated Author")
	form.Set("priority", "5")
	form.Set("amazon_url", "https://www.amazon.com/dp/NEW123")
	form.Set("reason", "Updated reason")

	req := httptest.NewRequest("POST", "/wishlist/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/{id}/update", handlers.HTMLUpdateWishlistItemHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if loc := rec.Header().Get("HX-Redirect"); loc != "/wishlist" {
		t.Fatalf("expected HX-Redirect=/wishlist, got %q", loc)
	}

	// Verify update persisted
	var title, reason, amazonURL string
	var priority int
	var author sql.NullString
	err = env.db.QueryRow("SELECT title, author, reason, priority, amazon_url FROM wishlist WHERE id = ?", id).Scan(&title, &author, &reason, &priority, &amazonURL)
	if err != nil {
		t.Fatalf("failed to query wishlist item: %v", err)
	}
	if title != "Updated Title" {
		t.Fatalf("expected title='Updated Title', got %q", title)
	}
	if priority != 5 {
		t.Fatalf("expected priority=5, got %d", priority)
	}
	if amazonURL != "https://www.amazon.com/dp/NEW123" {
		t.Fatalf("expected amazon_url, got %q", amazonURL)
	}
}

func TestHTMLUpdateWishlistItemHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	form := url.Values{}
	form.Set("title", "New Title")

	req := httptest.NewRequest("POST", "/wishlist/999/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", "999")

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/{id}/update", handlers.HTMLUpdateWishlistItemHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTMLUpdateWishlistItemHandler_MissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "Test", 3)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	form := url.Values{}
	// No title

	req := httptest.NewRequest("POST", "/wishlist/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))

	r := buildAdminHTMLRouter(t, env, "POST", "/wishlist/{id}/update", handlers.HTMLUpdateWishlistItemHandler(env.db))

	cookie := loginAndGetCookie(t, env)
	req.Header.Set("Cookie", cookie)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------- CSRF Token in Page Context ----------

// TestBuildPageContext_CSRFTokenWithAuthHTMLMiddleware verifies that
// buildPageContext correctly reads the CSRF token from the session store
// when the user is in the request context (set by RequireAuthHTML) but the
// session is not in the context. This is the regression test for the bug
// where HTMX DELETE requests failed with "CSRF token missing" because the
// page template rendered an empty CSRF token.
func TestBuildPageContext_CSRFTokenWithAuthHTMLMiddleware(t *testing.T) {
	env := setupTestEnv(t)

	// Step 1: Login to create a session with a CSRF token.
	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, testUsername, testPassword)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status %d", rec.Code)
	}

	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("no session cookie in login response")
	}

	// Step 2: Fetch the CSRF token via the /api/v1/csrf endpoint.
	req = httptest.NewRequest("GET", "/api/v1/csrf", nil)
	req.Header.Set("Cookie", cookie)
	rec = httptest.NewRecorder()
	handlers.CSRFTokenHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("csrf endpoint failed: status %d", rec.Code)
	}

	var csrfResp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&csrfResp)
	csrfToken, ok := csrfResp["csrf_token"].(string)
	if !ok || csrfToken == "" {
		t.Fatal("expected non-empty csrf_token from /api/v1/csrf")
	}

	// The CSRF endpoint saves the session (with the token) via Set-Cookie.
	// Use the updated cookie for subsequent requests.
	updatedCookie := getSessionCookie(rec)
	if updatedCookie != "" {
		cookie = updatedCookie
	}

	// Step 3: Simulate a GET request through RequireAuthHTML middleware.
	// RequireAuthHTML puts the user in context but does NOT put the session
	// in context. buildPageContext should still be able to read the CSRF
	// token from the session store.
	r := chi.NewRouter()
	var capturedCtx handlers.PageContextForTest
	r.Handle("/test", env.auth.RequireAuthHTML(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = handlers.BuildPageContextForTest(r, env.auth.Store(), auth.SessionID)
		w.WriteHeader(http.StatusOK)
	})))

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", cookie)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Verify the CSRF token was correctly read from the session store.
	if capturedCtx.CSRFToken != csrfToken {
		t.Fatalf("expected CSRF token %q in page context, got %q", csrfToken, capturedCtx.CSRFToken)
	}
	if capturedCtx.CSRFToken == "" {
		t.Fatal("expected non-empty CSRF token in page context")
	}
	if !capturedCtx.IsAuthenticated {
		t.Fatal("expected IsAuthenticated=true")
	}
	if !capturedCtx.IsAdmin {
		t.Fatal("expected IsAdmin=true")
	}
}
