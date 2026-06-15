package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Toph4er/family-library/internal/handlers"
)

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
