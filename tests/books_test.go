package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"git.rcsmaine.com/chris/library/internal/handlers"
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

	body := `{"title":"The Great Gatsby","isbn":"9780743273565","authors":"[\"F. Scott Fitzgerald\"]"}`
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
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
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
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
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

func TestListBooksHandler_SearchQuery(t *testing.T) {
	env := setupTestEnv(t)

	// Insert books with distinct titles
	_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "The Great Gatsby")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	_, err = env.db.Exec("INSERT INTO books (title) VALUES (?)", "Moby Dick")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	_, err = env.db.Exec("INSERT INTO books (title) VALUES (?)", "The Catcher in the Rye")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/?q=Gatsby", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item for search 'Gatsby', got %d", len(items))
	}
	if resp["total"].(float64) != 1 {
		t.Fatalf("expected total=1, got %v", resp["total"])
	}
}

func TestListBooksHandler_PerPageCap(t *testing.T) {
	env := setupTestEnv(t)

	// Insert 150 books
	for i := 0; i < 150; i++ {
		_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", fmt.Sprintf("Book %d", i))
		if err != nil {
			t.Fatalf("failed to insert book: %v", err)
		}
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	// Request per_page=200, should be capped at 100
	req := httptest.NewRequest("GET", "/?per_page=200", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	items := resp["items"].([]interface{})
	if len(items) != 100 {
		t.Fatalf("expected 100 items (capped), got %d", len(items))
	}
	if resp["per_page"].(float64) != 100 {
		t.Fatalf("expected per_page=100 (capped), got %v", resp["per_page"])
	}
	if resp["total"].(float64) != 150 {
		t.Fatalf("expected total=150, got %v", resp["total"])
	}
}

func TestListBooksHandler_Page2(t *testing.T) {
	env := setupTestEnv(t)

	// Insert 10 books
	for i := 1; i <= 10; i++ {
		_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", fmt.Sprintf("Book %d", i))
		if err != nil {
			t.Fatalf("failed to insert book: %v", err)
		}
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	// Request page 2 with per_page=3
	req := httptest.NewRequest("GET", "/?page=2&per_page=3", nil)
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
		t.Fatalf("expected 3 items on page 2, got %d", len(items))
	}
	if resp["total"].(float64) != 10 {
		t.Fatalf("expected total=10, got %v", resp["total"])
	}
	if resp["page"].(float64) != 2 {
		t.Fatalf("expected page=2, got %v", resp["page"])
	}
	if resp["per_page"].(float64) != 3 {
		t.Fatalf("expected per_page=3, got %v", resp["per_page"])
	}
}

func TestCreateBookHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateBookHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader("{not valid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookHandler_EmptyTitle(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":""}`
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

func TestUpdateBookHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"Updated"}`
	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBookHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Test Book")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/1", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBookHandler_MultiField(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "Old Title")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	newTitle := "New Title"
	newAuthors := `["Author One","Author Two"]`
	body := `{"title":"` + newTitle + `","authors":"` + strings.ReplaceAll(newAuthors, `"`, `\"`) + `"}`

	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
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
	if book["authors"] != newAuthors {
		t.Fatalf("expected authors='%s', got %v", newAuthors, book["authors"])
	}
}

func TestUpdateBookHandler_DuplicateISBN(t *testing.T) {
	env := setupTestEnv(t)

	// Insert two books with different ISBNs
	_, err := env.db.Exec("INSERT INTO books (isbn, title) VALUES (?, ?)", "9780061120084", "Book A")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	result, err := env.db.Exec("INSERT INTO books (isbn, title) VALUES (?, ?)", "9780743273565", "Book B")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	// Try to update Book B's ISBN to Book A's ISBN
	body := `{"isbn":"9780061120084"}`
	handler := handlers.UpdateBookHandler(env.db)
	req := httptest.NewRequest("PUT", "/books/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteBookHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.DeleteBookHandler(env.db)
	req := httptest.NewRequest("DELETE", "/books/abc", nil)
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImportISBNHandler_Duplicate(t *testing.T) {
	env := setupTestEnv(t)

	// First, create a book with the ISBN so that importing it again returns 409
	_, err := env.db.Exec("INSERT INTO books (isbn, title) VALUES (?, ?)", "9780061120084", "To Kill a Mockingbird")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	body := `{"isbn":"9780061120084"}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.ImportISBNHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}

	var book map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&book)
	if book["title"] != "To Kill a Mockingbird" {
		t.Fatalf("expected title='To Kill a Mockingbird', got %v", book["title"])
	}
}

func TestImportISBNHandler_InvalidISBN(t *testing.T) {
	env := setupTestEnv(t)

	// Send an empty ISBN
	body := `{"isbn":""}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.ImportISBNHandler(env.db))
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

func TestImportISBNHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAdminRouter(t, env, "POST", "/", handlers.ImportISBNHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader("{not valid"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLookupISBNHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Pre-seed the isbn_cache table so the handler doesn't need to hit the real Open Library API.
	testISBN := "9780061120084"
	cacheData := `{"title":"To Kill a Mockingbird","authors":"[\"Harper Lee\"]","publisher":"HarperCollins","publication_year":1960,"page_count":281,"description":"A classic novel.","cover_image_url":"https://example.com/cover.jpg","cover_source":"open_library"}`
	_, err := env.db.Exec(
		`INSERT INTO isbn_cache (isbn, data, fetched_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))`,
		testISBN, cacheData,
	)
	if err != nil {
		t.Fatalf("failed to seed isbn_cache: %v", err)
	}

	handler := handlers.LookupISBNHandler(env.db)
	req := httptest.NewRequest("GET", fmt.Sprintf("/books/lookup-isbn?isbn=%s", testISBN), nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["title"] != "To Kill a Mockingbird" {
		t.Fatalf("expected title='To Kill a Mockingbird', got %v", resp["title"])
	}
}

func TestLookupISBNHandler_MissingISBN(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.LookupISBNHandler(env.db)
	req := httptest.NewRequest("GET", "/books/lookup-isbn", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchBooksHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert books
	_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "The Great Gatsby")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	_, err = env.db.Exec("INSERT INTO books (title) VALUES (?)", "Moby Dick")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.SearchBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/?q=Gatsby", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item for search 'Gatsby', got %d", len(items))
	}
	if resp["total"].(float64) != 1 {
		t.Fatalf("expected total=1, got %v", resp["total"])
	}
}

func TestSearchBooksHandler_NoResults(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	_, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "The Great Gatsby")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.SearchBooksHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/?q=nonexistentterm", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected total=0, got %v", resp["total"])
	}
}
