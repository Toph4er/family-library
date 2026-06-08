package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

func TestThemeCSSHandler(t *testing.T) {
	handler := ThemeCSSHandler()

	req := httptest.NewRequest("GET", "/api/v1/theme/woodland/css", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "woodland")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/css; charset=utf-8" {
		t.Fatalf("expected Content-Type 'text/css; charset=utf-8', got %q", ct)
	}

	body := rec.Body.String()

	// Body should contain the primary CSS variable
	if !strings.Contains(body, "--color-primary") {
		t.Fatal("expected body to contain --color-primary")
	}

	// Body should NOT contain <style> wrapper tags — ThemeCSSHandler strips them
	if strings.Contains(body, "<style>") {
		t.Fatal("body should NOT contain <style> wrapper tags")
	}
	if strings.Contains(body, "</style>") {
		t.Fatal("body should NOT contain </style> wrapper tags")
	}

	// Body should contain the woodland primary color
	if !strings.Contains(body, "--color-primary: #2d5016") {
		t.Fatal("expected body to contain woodland primary color --color-primary: #2d5016")
	}
}

func TestThemeCSSHandler_unknownID(t *testing.T) {
	handler := ThemeCSSHandler()

	req := httptest.NewRequest("GET", "/api/v1/theme/nonexistent/css", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should NOT 404 — falls back to woodland
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (fallback), got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/css; charset=utf-8" {
		t.Fatalf("expected Content-Type 'text/css; charset=utf-8', got %q", ct)
	}

	body := rec.Body.String()

	// Should contain the fallback (woodland) primary color
	if !strings.Contains(body, "--color-primary: #2d5016") {
		t.Fatal("expected fallback body to contain woodland primary color #2d5016")
	}

	// Should not contain <style> tags
	if strings.Contains(body, "<style>") {
		t.Fatal("fallback body should NOT contain <style> wrapper tags")
	}
}

func TestBuildPageContext_nilDB_returnsDefaultTheme(t *testing.T) {
	// BuildPageContextForTest calls buildPageContext with db=nil.
	// It should NOT panic — loadActiveTheme(nil) returns WoodlandFairytale.
	req := httptest.NewRequest("GET", "/", nil)

	// Create a minimal CookieStore so buildPageContext doesn't panic on store.Get()
	secret := []byte("test-session-secret-that-is-32-bytes!!")
	store := sessions.NewCookieStore(secret)

	result := BuildPageContextForTest(req, store, "library_session")

	if result.ActiveTheme.ID != "woodland" {
		t.Errorf("ActiveTheme.ID = %q, want %q", result.ActiveTheme.ID, "woodland")
	}
	if result.ActiveTheme.Name != "Woodland Fairytale" {
		t.Errorf("ActiveTheme.Name = %q, want %q", result.ActiveTheme.Name, "Woodland Fairytale")
	}
	if result.ActiveTheme.Primary != "#2d5016" {
		t.Errorf("ActiveTheme.Primary = %q, want %q", result.ActiveTheme.Primary, "#2d5016")
	}
}

func TestThemeCSSHandler_allThemes(t *testing.T) {
	handler := ThemeCSSHandler()

	themeIDs := []string{"woodland", "space", "dinosaurs", "princesses"}
	for _, id := range themeIDs {
		t.Run(id, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/theme/"+id+"/css", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
			}

			body := rec.Body.String()
			if len(body) == 0 {
				t.Fatal("expected non-empty CSS body")
			}
			if strings.Contains(body, "<style>") {
				t.Fatal("body should NOT contain <style> wrapper tags")
			}
			if !strings.Contains(body, ":root") {
				t.Fatal("body should contain :root")
			}
		})
	}
}
