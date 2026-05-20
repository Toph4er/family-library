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

func TestListSettingsHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.ListSettingsHandler(env.db)
	req := httptest.NewRequest("GET", "/api/v1/settings/", nil)
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

	data := resp["data"].(map[string]interface{})

	// Verify non-sensitive settings are present
	if _, ok := data["cover_image_provider"]; !ok {
		t.Fatal("expected cover_image_provider in settings")
	}
	if data["cover_image_provider"] != "google_books" {
		t.Fatalf("expected cover_image_provider='google_books', got %v", data["cover_image_provider"])
	}

	if _, ok := data["site_name"]; !ok {
		t.Fatal("expected site_name in settings")
	}
	if data["site_name"] != "Our Library" {
		t.Fatalf("expected site_name='Our Library', got %v", data["site_name"])
	}

	// Verify sensitive key is excluded
	if _, ok := data["guest_password_hash"]; ok {
		t.Fatal("expected guest_password_hash to be excluded from settings")
	}
}

func TestListSettingsHandler_EmptyDB(t *testing.T) {
	env := setupTestEnv(t)

	// Clear all settings (migration seeds them, so we delete them)
	_, err := env.db.Exec("DELETE FROM settings")
	if err != nil {
		t.Fatalf("failed to clear settings: %v", err)
	}

	handler := handlers.ListSettingsHandler(env.db)
	req := httptest.NewRequest("GET", "/api/v1/settings/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	data := resp["data"].(map[string]interface{})
	if len(data) != 0 {
		t.Fatalf("expected 0 settings, got %d", len(data))
	}
}

func TestUpdateSettingHandler_Success(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"value":"amazon"}`
	handler := handlers.UpdateSettingHandler(env.db)
	req := httptest.NewRequest("PUT", "/api/v1/settings/cover_image_provider", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

	data := resp["data"].(map[string]interface{})
	if data["key"] != "cover_image_provider" {
		t.Fatalf("expected key='cover_image_provider', got %v", data["key"])
	}
	if data["value"] != "amazon" {
		t.Fatalf("expected value='amazon', got %v", data["value"])
	}

	// Verify the update persisted in the database
	var value string
	err := env.db.QueryRow("SELECT value FROM settings WHERE key = ?", "cover_image_provider").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query updated setting: %v", err)
	}
	if value != "amazon" {
		t.Fatalf("expected persisted value='amazon', got '%s'", value)
	}
}

func TestUpdateSettingHandler_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"value":"new_value"}`
	handler := handlers.UpdateSettingHandler(env.db)
	req := httptest.NewRequest("PUT", "/api/v1/settings/nonexistent_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateSettingHandler_SensitiveKey(t *testing.T) {
	env := setupTestEnv(t)

	body := `{"value":"hacked_password"}`
	handler := handlers.UpdateSettingHandler(env.db)
	req := httptest.NewRequest("PUT", "/api/v1/settings/guest_password_hash", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateSettingHandler_InvalidJSON(t *testing.T) {
	env := setupTestEnv(t)

	handler := handlers.UpdateSettingHandler(env.db)
	req := httptest.NewRequest("PUT", "/api/v1/settings/site_name", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateSettingHandler_SiteNameUpdate(t *testing.T) {
	env := setupTestEnv(t)

	newName := "New Library Name"
	body := fmt.Sprintf(`{"value":"%s"}`, newName)
	handler := handlers.UpdateSettingHandler(env.db)
	req := httptest.NewRequest("PUT", "/api/v1/settings/site_name", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify it persisted
	var value string
	err := env.db.QueryRow("SELECT value FROM settings WHERE key = ?", "site_name").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query updated setting: %v", err)
	}
	if value != newName {
		t.Fatalf("expected persisted value='%s', got '%s'", newName, value)
	}
}
