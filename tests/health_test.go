package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Toph4er/family-library/internal/handlers"
)

func TestHealthHandler_OK(t *testing.T) {
	env := setupTestEnv(t)

	h := handlers.NewHealthHandler(env.db)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Check(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["status"] != "ok" {
		t.Fatalf("expected status='ok', got %v", resp["status"])
	}
}

func TestHealthHandler_ContentType(t *testing.T) {
	env := setupTestEnv(t)

	h := handlers.NewHealthHandler(env.db)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Check(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}
