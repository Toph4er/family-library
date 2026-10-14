package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// webDir is the internal/web directory relative to this test's working dir
// (internal/api/), which is where NewRouter points staticAssetServer.
const webDir = "../web"

// TestStaticAssetServer_ServesCSS pins L3: real CSS assets must still be
// served under /static.
func TestStaticAssetServer_ServesCSS(t *testing.T) {
	h := staticAssetServer(webDir)
	req := httptest.NewRequest("GET", "/styles.css", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /styles.css, got %d (body=%q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("expected a CSS content type, got %q", ct)
	}
}

// TestStaticAssetServer_DeniesHTMLTemplateSource pins L3: the .html template
// sources in internal/web must never be served under /static.
func TestStaticAssetServer_DeniesHTMLTemplateSource(t *testing.T) {
	h := staticAssetServer(webDir)
	for _, path := range []string{"/base.html", "/book-detail.html", "/partials/book-card.html"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %s (template source), got %d", path, rec.Code)
		}
	}
}

// TestStaticAssetServer_DeniesDirectoryListing pins L3: a directory request
// must 404, not auto-list the internal/web contents.
func TestStaticAssetServer_DeniesDirectoryListing(t *testing.T) {
	h := staticAssetServer(webDir)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for the directory root, got %d (body=%q)", rec.Code, rec.Body.String())
	}
}
