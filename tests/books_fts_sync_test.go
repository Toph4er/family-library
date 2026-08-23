package tests

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Toph4er/family-library/internal/handlers"
	"github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/repository"
)

// Regression tests for the FTS5 auto-sync bug.
//
// books_fts is an EXTERNAL-CONTENT FTS5 table (content='books',
// content_rowid='id'), which does NOT auto-sync with the books table. The
// "no triggers needed" note in migration 00014 was incorrect, so books
// created/updated/deleted after the initial migration never reached the FTS
// index and /books?q= missed them. Migration 00017 adds three triggers that
// keep the index in sync; these tests lock that behaviour in.

var redirectBookID = regexp.MustCompile(`/books/(\d+)$`)

// ftsMatchCount returns the number of FTS rows (joined back to books) that
// match the safe-quoted query q.
func ftsMatchCount(t *testing.T, env *testEnv, q string) int {
	t.Helper()
	var n int
	err := env.db.QueryRow(
		"SELECT COUNT(*) FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ?",
		pages.SafeFTS5Query(q),
	).Scan(&n)
	if err != nil {
		t.Fatalf("FTS MATCH %q: %v", q, err)
	}
	return n
}

// createBookViaAPI posts form to POST /books/create and returns the new book ID.
func createBookViaAPI(t *testing.T, env *testEnv, form string) int64 {
	t.Helper()
	handler := handlers.HTMLCreateBookHandler(env.db.DB)
	req := httptest.NewRequest("POST", "/books/create", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /books/create -> %d: %s", rec.Code, rec.Body.String())
	}
	m := redirectBookID.FindStringSubmatch(rec.Header().Get("HX-Redirect"))
	if m == nil {
		t.Fatalf("no book ID in HX-Redirect %q", rec.Header().Get("HX-Redirect"))
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		t.Fatalf("parse book ID %q: %v", m[1], err)
	}
	return id
}

// A book created via the API must immediately be findable via FTS5 search.
func TestBookCreate_SyncsToFTS(t *testing.T) {
	env := setupTestEnv(t)

	createBookViaAPI(t, env,
		"title=The+Great+Gatsby&isbn=9780743273565&authors=F.+Scott+Fitzgerald&genres=classic+fiction")

	if n := ftsMatchCount(t, env, "Gatsby"); n != 1 {
		t.Errorf("expected 1 FTS match for title 'Gatsby' after create, got %d", n)
	}
	if n := ftsMatchCount(t, env, "Fitzgerald"); n != 1 {
		t.Errorf("expected 1 FTS match for author 'Fitzgerald' after create, got %d", n)
	}
	if n := ftsMatchCount(t, env, "classic"); n != 1 {
		t.Errorf("expected 1 FTS match for genre 'classic' after create, got %d", n)
	}
}

// Renaming a book must update the FTS index: new title matches, old title gone.
func TestBookUpdate_SyncsToFTS(t *testing.T) {
	env := setupTestEnv(t)
	id := createBookViaAPI(t, env, "title=Old+Title+Here&isbn=111111&authors=Someone")

	handler := handlers.HTMLUpdateBookHandler(env.db.DB)
	req := httptest.NewRequest("POST", "/books/0/update",
		strings.NewReader("title=Brand+New+Title&isbn=111111&authors=Someone"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = setURLParam(req, "id", strconv.FormatInt(id, 10))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /books/%d/update -> %d: %s", id, rec.Code, rec.Body.String())
	}

	if n := ftsMatchCount(t, env, "Brand"); n != 1 {
		t.Errorf("expected 1 FTS match for new title 'Brand' after update, got %d", n)
	}
	if n := ftsMatchCount(t, env, "Old"); n != 0 {
		t.Errorf("expected 0 FTS matches for old title 'Old' after update, got %d", n)
	}
}

// Deleting a book must remove it from the FTS index.
func TestBookDelete_RemovesFromFTS(t *testing.T) {
	env := setupTestEnv(t)
	id := createBookViaAPI(t, env, "title=To+Be+Deleted&isbn=222222&authors=Someone")

	if n := ftsMatchCount(t, env, "Deleted"); n != 1 {
		t.Fatalf("expected the book to be in FTS before delete, got %d", n)
	}

	repo := repository.NewBookRepository(env.db)
	handler := handlers.DeleteBookHandler(repo)
	req := httptest.NewRequest("DELETE", "/books/0", nil)
	req = setURLParam(req, "id", strconv.FormatInt(id, 10))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /books/%d -> %d: %s", id, rec.Code, rec.Body.String())
	}

	if n := ftsMatchCount(t, env, "Deleted"); n != 0 {
		t.Errorf("expected 0 FTS matches after delete, got %d", n)
	}
	var present int
	if err := env.db.QueryRow("SELECT COUNT(*) FROM books WHERE id = ?", id).Scan(&present); err != nil {
		t.Fatalf("check books table: %v", err)
	}
	if present != 0 {
		t.Errorf("expected book %d to be gone from books, still present", id)
	}
}
