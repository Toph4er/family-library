// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/Toph4er/family-library/internal/middleware"
)

// The tests in this file pin the per-visitor rate-limit keying: the trusted
// client IP is the CF-Connecting-IP header — the real visitor IP that
// Cloudflare sets unconditionally and nginx forwards — never the
// attacker-controlled X-Real-IP / True-Client-IP / X-Forwarded-For headers.

// TestClientIP_UsesCFConnectingIP verifies that the rate-limit key is taken
// from CF-Connecting-IP when present, and that spoofed X-Real-IP /
// True-Client-IP / X-Forwarded-For values are ignored (chi's
// ClientIPFromHeader reads only the one configured header).
func TestClientIP_UsesCFConnectingIP(t *testing.T) {
	const visitorIP = "203.0.113.5"
	const spoofedIP = "198.51.100.66"

	var got string
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		got = middleware.ClientIP(req)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.50:41000"
	// Real visitor IP: set by Cloudflare (authoritative), forwarded by nginx.
	req.Header.Set("CF-Connecting-IP", visitorIP)
	// Attacker-spoofed headers — must be ignored by the new keying:
	req.Header.Set("X-Real-IP", spoofedIP)
	req.Header.Set("True-Client-IP", spoofedIP)
	req.Header.Set("X-Forwarded-For", spoofedIP+", 198.51.100.67")

	r.ServeHTTP(httptest.NewRecorder(), req)

	if got == "" {
		t.Fatal("ClientIP returned empty; expected the CF-Connecting-IP value")
	}
	if got == spoofedIP {
		t.Fatalf("ClientIP = %q equals a spoofed header value — the rate-limit key is attacker-controllable", got)
	}
	if got != visitorIP {
		t.Fatalf("ClientIP = %q; want %q (the CF-Connecting-IP value)", got, visitorIP)
	}
}

// TestClientIP_FallsBackToRemoteAddr verifies the graceful fallback: with no
// CF-Connecting-IP header present (request did not come through
// Cloudflare/nginx), ClientIP(req) returns the host part of r.RemoteAddr —
// the TCP peer — so the limiter still gets a stable, non-spoofable key.
func TestClientIP_FallsBackToRemoteAddr(t *testing.T) {
	const peerIP = "192.0.2.10"
	const spoofedIP = "198.51.100.66"

	var got string
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		got = middleware.ClientIP(req)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = peerIP + ":41000"
	// NO CF-Connecting-IP header. Spoofed headers must have no effect.
	req.Header.Set("X-Real-IP", spoofedIP)
	req.Header.Set("True-Client-IP", spoofedIP)
	req.Header.Set("X-Forwarded-For", spoofedIP+", 198.51.100.67")

	r.ServeHTTP(httptest.NewRecorder(), req)

	if got == "" {
		t.Fatal("ClientIP returned empty; expected the host part of r.RemoteAddr")
	}
	if got != peerIP {
		t.Fatalf("ClientIP = %q; want %q (the host part of r.RemoteAddr)", got, peerIP)
	}
}

// TestRateLimiter_PerVisitorBuckets verifies the limiter is keyed per visitor
// via CF-Connecting-IP:
//   - two distinct CF-Connecting-IP values (even from the SAME TCP peer) act
//     as independent buckets, and
//   - a single visitor that exceeds the burst still gets 429s.
//
// We use visitor IPs (198.51.100.201/202) that no other test touches, so the
// shared limiter store stays isolated to this test. The auth burst is 10, so
// 12 requests from one visitor yield exactly 10 OK + 2 throttled.
func TestRateLimiter_PerVisitorBuckets(t *testing.T) {
	const visitorA = "198.51.100.201"
	const visitorB = "198.51.100.202"
	const sharedPeer = "192.0.2.60"

	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.RateLimiter)
	r.Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	doRequests := func(visitor string) (ok, limited int) {
		for i := 0; i < 12; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = sharedPeer + ":41000" // same TCP peer for both visitors
			req.Header.Set("CF-Connecting-IP", visitor)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			switch rec.Code {
			case http.StatusTooManyRequests:
				limited++
			case http.StatusOK:
				ok++
			}
		}
		return ok, limited
	}

	// Visitor A: burst is 10 → the first 10 must pass, the rest throttled.
	okA, limitedA := doRequests(visitorA)
	if okA != 10 || limitedA != 2 {
		t.Fatalf("visitor A: ok=%d limited=%d; want ok=10 limited=2 (burst=10)", okA, limitedA)
	}

	// Visitor B: a distinct CF-Connecting-IP from the SAME TCP peer is an
	// independent bucket — a full burst of 10 still passes despite A's 429s.
	okB, limitedB := doRequests(visitorB)
	if okB != 10 || limitedB != 2 {
		t.Fatalf("visitor B: ok=%d limited=%d; want ok=10 limited=2 (independent bucket)", okB, limitedB)
	}

	t.Logf("visitor A: ok=%d limited=%d; visitor B: ok=%d limited=%d", okA, limitedA, okB, limitedB)
}
