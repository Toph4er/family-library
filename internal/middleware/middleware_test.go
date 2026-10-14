package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSecurityHeadersCSP_Tightened pins M6: the CSP must NOT contain
// 'unsafe-eval' (Alpine is now served from the @alpinejs/csp build), and
// connect-src must be 'self' only (the app makes no cross-origin fetches),
// while still carrying a per-request nonce and the CDN script origins.
func TestSecurityHeadersCSP_Tightened(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := GenerateCSPNonce(SecurityHeaders(next))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("missing Content-Security-Policy header")
	}

	// M6: unsafe-eval must be gone.
	if strings.Contains(csp, "unsafe-eval") {
		t.Errorf("CSP must not contain 'unsafe-eval': %s", csp)
	}
	// M6: connect-src must be 'self' only (no CDN hosts).
	if !strings.Contains(csp, "connect-src 'self'") {
		t.Errorf("connect-src should be 'self' only: %s", csp)
	}
	if strings.Contains(csp, "connect-src 'self' http") {
		t.Errorf("connect-src must not include CDN hosts: %s", csp)
	}
	// script-src must still allow the CDN origins (Alpine/HTMX/qrcode) and
	// carry a per-request nonce for inline scripts.
	if !strings.Contains(csp, "https://cdn.jsdelivr.net") || !strings.Contains(csp, "https://unpkg.com") {
		t.Errorf("script-src must allow the CDN origins: %s", csp)
	}
	if !strings.Contains(csp, "nonce-") {
		t.Errorf("script-src must carry a per-request nonce: %s", csp)
	}
	// Sanity: the other hardening directives are still present.
	for _, want := range []string{
		"default-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP missing directive %q: %s", want, csp)
		}
	}
}

// TestSecurityHeaders_OtherHeaders pins the non-CSP security headers so they
// are not accidentally dropped while editing the CSP block.
func TestSecurityHeaders_OtherHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := GenerateCSPNonce(SecurityHeaders(next))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	wantHeaders := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for header, want := range wantHeaders {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("missing Strict-Transport-Security header")
	}
}
