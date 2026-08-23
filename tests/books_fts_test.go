package tests

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/handlers"
	"github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/models"
)

// Regression tests for the production 500 on /books?q= with apostrophes.
// The SQL was already parameterized; the failure was raw user input reaching
// the FTS5 MATCH expression (e.g. "We're going" -> fts5: syntax error near "'").

// booksPageTemplate returns a minimal parseable template for RenderBooksPage tests.
func booksPageTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("stub").Parse(`{{define "books.html"}}OK{{end}}`)
	if err != nil {
		t.Fatalf("parse stub template: %v", err)
	}
	return tmpl
}

// apostropheQueries are inputs that previously produced a 500 (FTS5 syntax error).
var apostropheQueries = []string{
	"We're going",
	"don't",
	`it's "quoted"`,
	"a - b",
	"col1:col2",
	"*",
	"(unclosed",
}

func TestBooksPage_ApostropheQueryDoesNot500(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")
	cookie := loginAndGetCookie(t, env)

	handler := pages.RenderBooksPage(booksPageTemplate(t), env.db.DB, env.auth.Store(), auth.SessionID)

	for _, q := range apostropheQueries {
		req := httptest.NewRequest("GET", "/books", nil)
		req.URL.RawQuery = url.Values{"q": {q}}.Encode()
		req.Header.Set("Cookie", cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusInternalServerError {
			t.Fatalf("GET /books?q=%s -> 500 (FTS5 syntax error): %s", q, rec.Body.String())
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /books?q=%s -> %d: %s", q, rec.Code, rec.Body.String())
		}
	}
}

func TestBooksPage_ApostropheQueryStillMatches(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")
	seedBook(t, env, "A clean title", "B. Jones")

	// The escaped query must still find the book whose title contains the apostrophe.
	var n int
	err := env.db.QueryRow(
		"SELECT COUNT(*) FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ?",
		pages.SafeFTS5Query("We're going"),
	).Scan(&n)
	if err != nil {
		t.Fatalf("query with escaped apostrophe: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 match for \"We're going\", got %d", n)
	}
}

func TestBookSearchHandler_ApostropheDoesNot500(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")

	handler := pages.BookSearchHandler(env.db.DB)
	req := httptest.NewRequest("GET", "/books/search", nil)
	req.URL.RawQuery = url.Values{"q": {"We're going"}}.Encode()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books/search?q=We're+going -> %d: %s", rec.Code, rec.Body.String())
	}

	var books []models.Book
	if err := json.Unmarshal(rec.Body.Bytes(), &books); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	found := false
	for _, b := range books {
		if strings.Contains(b.Title, "going on a trip") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the apostrophe-titled book in results, got %v", books)
	}
}

func TestBookSelector_ApostropheDoesNot500(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")

	handler := handlers.HTMLBookSelectorHandler(env.db.DB)
	req := httptest.NewRequest("GET", "/reading-logs/book-selector", nil)
	req.URL.RawQuery = url.Values{"q": {"We're going"}}.Encode()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /reading-logs/book-selector?q=We're+going -> %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "going on a trip") {
		t.Fatalf("expected the apostrophe-titled book in selector results: %s", rec.Body.String())
	}
}

// seedBook inserts a book (title + authors) into the test database and keeps
// the FTS index in sync (external-content FTS5 tables are not auto-synced; the
// app writes both tables explicitly, e.g. reading_log.go external-book path).
func seedBook(t *testing.T, env *testEnv, title, authors string) {
	t.Helper()
	result, err := env.db.Exec("INSERT INTO books (title, authors) VALUES (?, ?)", title, authors)
	if err != nil {
		t.Fatalf("insert book %q: %v", title, err)
	}
	id, _ := result.LastInsertId()
	if _, err := env.db.Exec("INSERT INTO books_fts (rowid, title, authors) VALUES (?, ?, ?)", id, title, authors); err != nil {
		t.Fatalf("insert book into FTS index %q: %v", title, err)
	}
}
