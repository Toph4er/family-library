package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Toph4er/family-library/internal/handlers"
	"github.com/Toph4er/family-library/internal/repository"
)

func TestDeleteBookHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a book
	result, err := env.db.Exec("INSERT INTO books (title) VALUES (?)", "To Delete")
	if err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.DeleteBookHandler(repository.NewBookRepository(env.db))
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

	handler := handlers.DeleteBookHandler(repository.NewBookRepository(env.db))
	req := httptest.NewRequest("DELETE", "/books/999", nil)
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteBookHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.DeleteBookHandler(repository.NewBookRepository(env.db))
	req := httptest.NewRequest("DELETE", "/books/abc", nil)
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

	handler := handlers.LookupISBNHandler(env.db.DB)
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

	handler := handlers.LookupISBNHandler(env.db.DB)
	req := httptest.NewRequest("GET", "/books/lookup-isbn", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
