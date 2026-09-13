// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestRateLimiter_OptionsPreflightNotCounted verifies L9(a): CORS preflights
// (OPTIONS) do not consume the per-visitor budget. The burst is 10, so
// 9 POSTs + an OPTIONS + a 10th POST must pass (the 11th is then throttled).
// If OPTIONS were counted, the 10th POST would already be throttled.
func TestRateLimiter_OptionsPreflightNotCounted(t *testing.T) {
	const visitor = "198.51.100.203"

	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.RateLimiter)
	r.Post("/api/v1/thing", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	do := func(method string) int {
		req := httptest.NewRequest(method, "/api/v1/thing", nil)
		req.RemoteAddr = "192.0.2.61:41000"
		req.Header.Set("CF-Connecting-IP", visitor)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	// 9 POSTs (9 of the 10-burst used).
	for i := 0; i < 9; i++ {
		if code := do("POST"); code != http.StatusOK {
			t.Fatalf("POST #%d: expected 200, got %d", i+1, code)
		}
	}
	// OPTIONS preflight — must be free, and must not be throttled itself.
	if code := do("OPTIONS"); code == http.StatusTooManyRequests {
		t.Fatalf("OPTIONS preflight was throttled: %d", code)
	}
	// 10th POST: passes only if the OPTIONS consumed no budget.
	if code := do("POST"); code != http.StatusOK {
		t.Fatalf("POST after OPTIONS: expected 200, got %d — OPTIONS preflight consumed rate-limit budget", code)
	}
	// 11th POST: over the burst — must be throttled either way.
	if code := do("POST"); code != http.StatusTooManyRequests {
		t.Fatalf("11th POST after OPTIONS: expected 429 (burst exhausted), got %d", code)
	}
}

// TestRateLimiter_APIAuthStrictLimit verifies L9(c): /api/v1/auth/* actually
// gets the strict limiter (burst 10, then ~1 request/2min). Before the fix
// the prefix check compared path[:11] against the 12-char "/api/v1/auth",
// which can never match — so API auth routes silently got the general
// limiter (~1 request/600ms) instead.
func TestRateLimiter_APIAuthStrictLimit(t *testing.T) {
	const strictVisitor = "198.51.100.204" // → /api/v1/auth/login
	const generalVisitor = "198.51.100.205" // → /api/v1/other

	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
	r.Use(middleware.RateLimiter)
	both := func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(http.StatusOK) }
	r.Post("/api/v1/auth/login", both)
	r.Post("/api/v1/other", both)

	do := func(visitor, path string) int {
		req := httptest.NewRequest("POST", path, nil)
		req.RemoteAddr = "192.0.2.62:41000"
		req.Header.Set("CF-Connecting-IP", visitor)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	// Exhaust both bursts (10) and confirm the 11th is throttled for both.
	for _, pair := range []struct {
		visitor, path string
	}{
		{strictVisitor, "/api/v1/auth/login"},
		{generalVisitor, "/api/v1/other"},
	} {
		for i := 0; i < 10; i++ {
			if code := do(pair.visitor, pair.path); code != http.StatusOK {
				t.Fatalf("%s POST #%d: expected 200, got %d", pair.path, i+1, code)
			}
		}
		if code := do(pair.visitor, pair.path); code != http.StatusTooManyRequests {
			t.Fatalf("%s POST #11: expected 429, got %d", pair.path, code)
		}
	}

	// After 800ms the GENERAL limiter (~1/600ms) has refilled one token, but
	// the STRICT limiter (~1/2min) has not. If /api/v1/auth silently used the
	// general limiter, the strict visitor's 12th POST would now pass.
	time.Sleep(800 * time.Millisecond)
	if code := do(generalVisitor, "/api/v1/other"); code != http.StatusOK {
		t.Fatalf("general route after 800ms: expected 200 (refilled), got %d", code)
	}
	if code := do(strictVisitor, "/api/v1/auth/login"); code != http.StatusTooManyRequests {
		t.Fatalf("/api/v1/auth after 800ms: expected 429 (strict limiter), got %d — API auth routes are not getting the strict limit", code)
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
