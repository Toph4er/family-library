package tests

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"git.rcsmaine.com/chris/library/internal/handlers"
)

func TestImportISBNHandler_Debug(t *testing.T) {
	env := setupTestEnv(t)

	testISBN := "9780061120084"
	cacheData := `{"title":"To Kill a Mockingbird","authors":"[\"Harper Lee\"]","publisher":"HarperCollins","publication_year":1960,"page_count":281,"description":"A classic novel.","cover_image_url":"https://example.com/cover.jpg","cover_source":"open_library"}`
	_, err := env.db.Exec(
		`INSERT INTO isbn_cache (isbn, data, fetched_at) VALUES (?, ?, datetime('now'))`,
		testISBN, cacheData,
	)
	if err != nil {
		t.Fatalf("failed to seed isbn_cache: %v", err)
	}

	body := fmt.Sprintf(`{"isbn":"%s"}`, testISBN)
	r := buildAdminRouter(t, env, "POST", "/", handlers.ImportISBNHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	t.Logf("Status: %d", rec.Code)
	t.Logf("Body: %s", rec.Body.String())

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	t.Logf("Response: %+v", resp)
}
