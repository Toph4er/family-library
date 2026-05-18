package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// buildAuthRouter creates a chi router that wraps the given handler with
// RequireAuth middleware, so we can test protected endpoints with a session cookie.
func buildAuthRouter(t *testing.T, env *testEnv, method, path string, handler http.HandlerFunc) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(path, env.auth.RequireAuth(handler))
	return r
}

// buildAdminRouter creates a chi router that wraps the given handler with
// RequireAdmin middleware.
func buildAdminRouter(t *testing.T, env *testEnv, method, path string, handler http.HandlerFunc) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(path, env.auth.RequireAdmin(handler))
	return r
}

func TestListBooksHandler_Empty(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if !resp["success"].(bool) {
		t.Fatal("expected success=true")
	}
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected total=0, got %v", resp["total"])
	}
}

func TestListBooksHandler_WithBooks(t *testing.T) {
	env := setupTestEnv(t)

	// Insert test books
	for i := 1; i <= 5; i++ {
		_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Book "+string(rune('A'+i-1)))
		if err != nil {
			t.Fatalf("failed to insert book: %v", err)
		}
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["total"].(float64) != 5 {
		t.Fatalf("expected total=5, got %v", resp["total"])
	}
}

func TestListBooksHandler_Pagination(t *testing.T) {
	env := setupTestEnv(t)

	// Insert 10 books
	for i := 1; i <= 10; i++ {
		_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Book "+string(rune('A'+i-1)))
		if err != nil {
			t.Fatalf("failed to insert book: %v", err)
		}
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/?per_page=3", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if resp["total"].(float64) != 10 {
		t.Fatalf("expected total=10, got %v", resp["total"])
	}
	if resp["per_page"].(float64) != 3 {
		t.Fatalf("expected per_page=3, got %v", resp["per_page"])
	}
}

func TestListBooksHandler_NotAuthenticated(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"The Great Gatsby","authors":["F. Scott Fitzgerald"]}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateBookHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var book map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&book)
	if book["title"] != "The Great Gatsby" {
		t.Fatalf("expected title='The Great Gatsby', got %v", book["title"])
	}
}

func TestCreateBookHandler_MissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"authors":["Someone"]}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateBookHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookHandler_DuplicateISBN(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book with an ISBN
	_, err := env.db.Exec("INSERT INTO books (isbn, title) VALUES (?, ?)", "9780061120084", "To Kill a Mockingbird")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	body := `{"title":"Duplicate","isbn":"9780061120084"}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateBookHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookHandler_ISBNNormalization(t *testing.T) {
	env := setupTestEnv(t)

	// Create a book with hyphenated ISBN
	body := `{"title":"Test Book","isbn":"978-0-06-112008-4"}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateBookHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify ISBN was stored without hyphens
	var storedISBN string
	err := env.db.QueryRow("SELECT isbn FROM books WHERE title = ?", "Test Book").Scan(&storedISBN)
	if err != nil {
		t.Fatalf("failed to query ISBN: %v", err)
	}
	if storedISBN != "9780061120084" {
		t.Fatalf("expected normalized ISBN '9780061120084', got '%s'", storedISBN)
	}
}

func TestGetBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Test Book")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.GetBookHandler(env.db)
	req := httptest.NewRequest("GET", "/books/1", nil)
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var book map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&book)
	if book["title"] != "Test Book" {
		t.Fatalf("expected title='Test Book', got %v", book["title"])
	}
}

func TestGetBookHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.GetBookHandler(env.db)
	req := httptest.NewRequest("GET", "/books/999", nil)
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetBookHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.GetBookHandler(env.db)
	req := httptest.NewRequest("GET", "/books/abc", nil)
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Old Title")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	newTitle := "New Title"
	body := `{"title":"` + newTitle + `"}`

	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", string(id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var book map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&book)
	if book["title"] != newTitle {
		t.Fatalf("expected title='%s', got %v", newTitle, book["title"])
	}
}

func TestUpdateBookHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"New Title"}`
	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "To Delete")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.DeleteBookHandler(env.db)
	req := httptest.NewRequest("DELETE", "/books/1", nil)
	req = setURLParam(req, "id", string(id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify book is actually deleted
	var count int
	err = env.db.QueryRow("SELECT COUNT(*) FROM books WHERE title = ?", "To Delete").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if count != 0 {
		t.Fatal("expected book to be deleted")
	}
}

func TestDeleteBookHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.DeleteBookHandler(env.db)
	req := httptest.NewRequest("DELETE", "/books/999", nil)
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
