package pages

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// openSettingsTestDB opens an in-memory SQLite DB with just the settings
// table (what loadGuestVisibility / BookSearchHandler need). A single
// connection is enforced so the :memory: DB is shared across queries.
func openSettingsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create settings table: %v", err)
	}
	return db
}

func TestSafeFTS5Query(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"apostrophe", "We're going", `"We're"* "going"*`},
		{"contraction", "don't", `"don't"*`},
		{"embedded double quote", `it's "quoted"`, `"it's"* """quoted"""*`},
		{"dash operator", "a - b", `"a"* "-"* "b"*`},
		{"column filter", "col1:col2", `"col1:col2"*`},
		{"prefix star", "*", `"*"*`},
		{"unclosed paren", "(unclosed", `"(unclosed"*`},
		{"AND keyword", "a AND b", `"a"* "AND"* "b"*`},
		{"NEAR operator", "NEAR(a b)", `"NEAR(a"* "b)"*`},
		{"extra whitespace", "  hello    world  ", `"hello"* "world"*`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeFTS5Query(tt.in); got != tt.want {
				t.Fatalf("SafeFTS5Query(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- M3: loadGuestVisibility reads the admin's default_guest_visibility ---

func TestLoadGuestVisibility_Present(t *testing.T) {
	db := openSettingsTestDB(t)
	blob := `{"title":true,"authors":true,"isbn":false,"notes":false}`
	if _, err := db.Exec(
		`INSERT INTO settings (key, value) VALUES ('default_guest_visibility', ?)`, blob,
	); err != nil {
		t.Fatal(err)
	}
	vis := loadGuestVisibility(db)
	if vis == nil {
		t.Fatal("expected a non-nil visibility map")
	}
	if !vis["title"] || !vis["authors"] {
		t.Errorf("title/authors should be visible: %v", vis)
	}
	if vis["isbn"] || vis["notes"] {
		t.Errorf("isbn/notes should be hidden: %v", vis)
	}
}

func TestLoadGuestVisibility_Absent(t *testing.T) {
	db := openSettingsTestDB(t)
	if got := loadGuestVisibility(db); got != nil {
		t.Errorf("expected nil when the setting is absent, got %v", got)
	}
}

func TestLoadGuestVisibility_Malformed(t *testing.T) {
	db := openSettingsTestDB(t)
	if _, err := db.Exec(
		`INSERT INTO settings (key, value) VALUES ('default_guest_visibility', 'not-json')`,
	); err != nil {
		t.Fatal(err)
	}
	if got := loadGuestVisibility(db); got != nil {
		t.Errorf("expected nil when the setting is malformed, got %v", got)
	}
}

func TestLoadGuestVisibility_EmptyString(t *testing.T) {
	db := openSettingsTestDB(t)
	if _, err := db.Exec(
		`INSERT INTO settings (key, value) VALUES ('default_guest_visibility', '')`,
	); err != nil {
		t.Fatal(err)
	}
	if got := loadGuestVisibility(db); got != nil {
		t.Errorf("expected nil for an empty-string setting, got %v", got)
	}
}

// --- L2: BookSearchHandler must not leak internal error details ---

func TestBookSearchHandler_ErrorIsGeneric(t *testing.T) {
	// L2: on a DB error the client must get a generic message, never the
	// internal driver error (e.g. "no such table: books_fts").
	db := openSettingsTestDB(t) // no books_fts table → the query fails
	h := BookSearchHandler(db)

	req := httptest.NewRequest("GET", "/books/search?q=hello", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body=%q)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Internal server error") {
		t.Errorf("expected the generic message in the body, got %q", body)
	}
	// Must NOT leak the raw driver error or the table name.
	if strings.Contains(body, "no such table") || strings.Contains(body, "books_fts") {
		t.Errorf("error body must not leak internal details: %q", body)
	}
}
