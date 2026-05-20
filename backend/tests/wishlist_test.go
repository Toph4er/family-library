package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.rcsmaine.com/chris/library/backend/internal/handlers"
)

func TestListWishlistHandler_Empty(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListWishlistHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// ListWishlistHandler returns the raw slice (no success/data wrapper)
	var items []interface{}
	json.NewDecoder(rec.Body).Decode(&items)
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestListWishlistHandler_WithItems(t *testing.T) {
	env := setupTestEnv(t)

	// Insert test wishlist items
	_, err := env.db.Exec(
		"INSERT INTO wishlist (title, author, priority) VALUES (?, ?, ?)",
		"Dune", "Frank Herbert", 8,
	)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}

	_, err = env.db.Exec(
		"INSERT INTO wishlist (title, author, priority) VALUES (?, ?, ?)",
		"Foundation", "Isaac Asimov", 3,
	)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}

	r := buildAuthRouter(t, env, "GET", "/", handlers.ListWishlistHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// ListWishlistHandler returns the raw slice (no success/data wrapper)
	var items []interface{}
	json.NewDecoder(rec.Body).Decode(&items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify ordering: priority DESC, so Dune (8) comes before Foundation (3)
	first := items[0].(map[string]interface{})
	if first["title"] != "Dune" {
		t.Fatalf("expected first item 'Dune', got %v", first["title"])
	}
}

func TestCreateWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"1984","author":"George Orwell","priority":7}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var item map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&item)
	if item["title"] != "1984" {
		t.Fatalf("expected title='1984', got %v", item["title"])
	}
	if item["author"] != "George Orwell" {
		t.Fatalf("expected author='George Orwell', got %v", item["author"])
	}
	if item["priority"].(float64) != 7 {
		t.Fatalf("expected priority=7, got %v", item["priority"])
	}
}

func TestCreateWishlistItemHandler_EmptyTitle(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":""}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
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

func TestCreateWishlistItemHandler_MissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"author":"Someone"}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
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

func TestCreateWishlistItemHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateWishlistItemHandler_AllOptionalFields(t *testing.T) {
	env := setupTestEnv(t)

	amazonURL := "https://www.amazon.com/dp/ABC123"
	thriftbooksURL := "https://www.thriftbooks.com/w/xyz"
	otherURLs := "https://example.com/book"

	body := fmt.Sprintf(`{
		"title":"War and Peace",
		"author":"Leo Tolstoy",
		"isbn":"9780199232765",
		"reason":"Classic literature",
		"priority":10,
		"amazon_url":"%s",
		"thriftbooks_url":"%s",
		"other_urls":"%s",
		"notes":"Gift idea"
	}`, amazonURL, thriftbooksURL, otherURLs)

	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
	cookie := loginAndGetCookie(t, env)

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var item map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&item)
	if item["isbn"] != "9780199232765" {
		t.Fatalf("expected isbn='9780199232765', got %v", item["isbn"])
	}
	if item["reason"] != "Classic literature" {
		t.Fatalf("expected reason='Classic literature', got %v", item["reason"])
	}
	if item["priority"].(float64) != 10 {
		t.Fatalf("expected priority=10, got %v", item["priority"])
	}
}

func TestCreateWishlistItemHandler_TitleWhitespaceOnly(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"   "}`
	r := buildAdminRouter(t, env, "POST", "/", handlers.CreateWishlistItemHandler(env.db))
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

func TestUpdateWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a wishlist item
	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "Old Title", 5)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	newTitle := "Updated Title"
	body := fmt.Sprintf(`{"title":"%s"}`, newTitle)

	handler := handlers.UpdateWishlistItemHandler(env.db)
	req := httptest.NewRequest("PUT", "/wishlist/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// UpdateWishlistItemHandler returns the raw item (no success/data wrapper)
	var item map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&item)
	if item["title"] != newTitle {
		t.Fatalf("expected title='%s', got %v", newTitle, item["title"])
	}
}

func TestUpdateWishlistItemHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"New Title"}`
	handler := handlers.UpdateWishlistItemHandler(env.db)
	req := httptest.NewRequest("PUT", "/wishlist/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateWishlistItemHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"title":"New Title"}`
	handler := handlers.UpdateWishlistItemHandler(env.db)
	req := httptest.NewRequest("PUT", "/wishlist/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateWishlistItemHandler_NoFields(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a wishlist item
	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "No Change", 5)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	// Send an empty update (no fields)
	body := `{}`
	handler := handlers.UpdateWishlistItemHandler(env.db)
	req := httptest.NewRequest("PUT", "/wishlist/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// UpdateWishlistItemHandler returns the raw item (no success/data wrapper)
	var item map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&item)
	if item["title"] != "No Change" {
		t.Fatalf("expected title='No Change', got %v", item["title"])
	}
}

func TestDeleteWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a wishlist item
	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "To Delete", 5)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.DeleteWishlistItemHandler(env.db)
	req := httptest.NewRequest("DELETE", "/wishlist/1", nil)
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Fatal("expected success=true")
	}

	// Verify the item is actually deleted
	var count int
	err = env.db.QueryRow("SELECT COUNT(*) FROM wishlist WHERE title = ?", "To Delete").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if count != 0 {
		t.Fatal("expected wishlist item to be deleted")
	}
}

func TestDeleteWishlistItemHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.DeleteWishlistItemHandler(env.db)
	req := httptest.NewRequest("DELETE", "/wishlist/999", nil)
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteWishlistItemHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.DeleteWishlistItemHandler(env.db)
	req := httptest.NewRequest("DELETE", "/wishlist/abc", nil)
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFulfillWishlistItemHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Insert a wishlist item
	result, err := env.db.Exec("INSERT INTO wishlist (title, priority) VALUES (?, ?)", "To Fulfill", 5)
	if err != nil {
		t.Fatalf("failed to insert wishlist item: %v", err)
	}
	id, _ := result.LastInsertId()

	handler := handlers.FulfillWishlistItemHandler(env.db)
	req := httptest.NewRequest("POST", "/wishlist/1/fulfill", nil)
	req = setURLParam(req, "id", fmt.Sprintf("%d", id))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s: %s", rec.Code, rec.Body.String(), rec.Body.String())
	}

	// FulfillWishlistItemHandler returns the raw item (no success/data wrapper)
	var item map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&item)
	if item["fulfilled"] != true {
		t.Fatal("expected fulfilled=true")
	}
	if item["fulfilled_at"] == nil {
		t.Fatal("expected fulfilled_at to be set")
	}
}

func TestFulfillWishlistItemHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.FulfillWishlistItemHandler(env.db)
	req := httptest.NewRequest("POST", "/wishlist/999/fulfill", nil)
	req = setURLParam(req, "id", "999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFulfillWishlistItemHandler_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.FulfillWishlistItemHandler(env.db)
	req := httptest.NewRequest("POST", "/wishlist/abc/fulfill", nil)
	req = setURLParam(req, "id", "abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
