// Package middleware provides HTTP middleware for security, logging,
// rate limiting, and error recovery.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

// nonceContextKey is the key type for storing the CSP nonce in request context.
type nonceContextKey struct{}

// GenerateCSPNonce generates a random 16-byte nonce and stores it in the
// request context. Templates can access it via .Nonce on pageContext.
func GenerateCSPNonce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 16)
		_, _ = rand.Read(buf)
		nonce := base64.RawURLEncoding.EncodeToString(buf)
		ctx := context.WithValue(r.Context(), nonceContextKey{}, nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCSPNonce returns the nonce for the current request, or empty string
// if no nonce was generated (e.g., for API responses that skip HTML rendering).
func GetCSPNonce(r *http.Request) string {
	if v := r.Context().Value(nonceContextKey{}); v != nil {
		return v.(string)
	}
	return ""
}

// SecurityHeaders adds security-related HTTP headers to every response.
//
// Headers set:
//   - Strict-Transport-Security (HSTS)
//   - X-Content-Type-Options
//   - X-Frame-Options
//   - Referrer-Policy
//   - Permissions-Policy
//   - Cross-Origin-Opener-Policy
//   - Content-Security-Policy (with per-request nonce for scripts)
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(self), microphone=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		// Build CSP with per-request nonce for inline scripts.
		// style-src includes 'unsafe-inline' because inline style= attributes
		// are pervasive in templates and cannot be nonce-protected (nonces only
		// work for <script nonce="..."> and <link nonce="..."> tags).
		nonce := GetCSPNonce(r)
		// Alpine.js requires 'unsafe-eval' to evaluate x-data/@click/x-show
		// expressions. This is a known requirement — Alpine.js uses eval()
		// internally to parse reactive expressions. See:
		// https://alpinejs.dev/advanced/csp
		scriptSrc := "'self' 'unsafe-eval' https://cdn.jsdelivr.net https://unpkg.com"
		if nonce != "" {
			scriptSrc += " 'nonce-" + nonce + "'"
		}
		styleSrc := "'self' fonts.googleapis.com 'unsafe-inline'"
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src "+scriptSrc+"; "+
				"style-src "+styleSrc+"; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' fonts.gstatic.com; "+
				"connect-src 'self' https://cdn.jsdelivr.net; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
		)
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs each request using the standard library slog package.
//
// It records the method, path, query string, client IP, response status,
// and elapsed time. The request ID (set by chi's RequestID middleware)
// is included for traceability.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture the status code.
		wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", ClientIP(r),
			"status", wrapped.status,
			"duration", duration.String(),
			"request_id", r.Context().Value("request.id"),
		)
	})
}

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// limiterEntry wraps a rate limiter with its last-access timestamp.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// rateLimiterStore holds per-IP rate limiters.
type rateLimiterStore struct {
	mu      sync.Mutex
	limits  map[string]*limiterEntry
	cleanup *time.Ticker
}

var limiterStore = &rateLimiterStore{
	limits: make(map[string]*limiterEntry),
}

func init() {
	// Clean up stale entries every 5 minutes to prevent memory leaks.
	limiterStore.cleanup = time.NewTicker(5 * time.Minute)
	go func() {
		for range limiterStore.cleanup.C {
			limiterStore.cleanupStale()
		}
	}()
}

// cleanupStale removes rate limiter entries that haven't been accessed
// in over an hour, preventing unbounded memory growth from stale IPs.
func (s *rateLimiterStore) cleanupStale() {
	cutoff := time.Now().Add(-1 * time.Hour)

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, entry := range s.limits {
		if entry.lastAccess.Before(cutoff) {
			delete(s.limits, key)
		}
	}
}

// getLimiter returns (or creates) a rate limiter for the given IP.
//
// Auth endpoints get a stricter limit (~30 requests/hour) while general
// API endpoints get 100 requests/minute.
func (s *rateLimiterStore) getLimiter(ip string, strict bool) *rate.Limiter {
	key := ip
	if strict {
		key += "|strict"
	} else {
		key += "|general"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.limits[key]; ok {
		entry.lastAccess = time.Now()
		return entry.limiter
	}

	var limiter *rate.Limiter
	if strict {
		// ~30 requests per hour for auth endpoints (burst of 10, then 1 per 2 min).
		limiter = rate.NewLimiter(rate.Every(2*time.Minute), 10)
	} else {
		// ~100 requests per minute for general API (burst of 10, then 1 per 600ms).
		limiter = rate.NewLimiter(rate.Every(600*time.Millisecond), 10)
	}

	s.limits[key] = &limiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	return limiter
}

// ClientIP returns the trusted client IP for the request. It reads the value
// stored by chi's ClientIPFromHeader("CF-Connecting-IP") middleware — the real
// visitor IP that Cloudflare sets (it unconditionally overwrites the header)
// and nginx forwards to us. If that middleware is not in the chain (e.g. in
// unit tests) or the CF-Connecting-IP header is absent, it falls back to the
// host part of r.RemoteAddr — the TCP peer — which is network-set and not
// attacker-controllable per-request.
//
// This is deliberately NOT derived from any other client-supplied header
// (X-Forwarded-For, X-Real-IP, True-Client-IP). chi's legacy RealIP middleware
// trusted those headers and rewrote r.RemoteAddr, letting an attacker reset
// this limiter per request (CVE GO-2026-5774/5775/5777). See the comment in
// internal/api/router.go for the topology and trust rationale.
func ClientIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimiter enforces per-IP rate limiting on API and auth requests.
//
// Auth endpoints (/api/v1/auth/* and /auth/login, /auth/guest-login) are
// limited to 10 requests per hour.  Other API endpoints are limited to
// 100 requests per minute.  Static file requests and health checks are not
// rate-limited.
func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for static files and health checks.
		if r.URL.Path == "/health" || len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/static" {
			next.ServeHTTP(w, r)
			return
		}

		// Determine if this is a strict (auth) or general (API) path.
		isAuth := r.URL.Path == "/auth/login" || r.URL.Path == "/auth/guest-login"
		isAPIAuth := len(r.URL.Path) >= 11 && r.URL.Path[:11] == "/api/v1/auth"

		if isAuth {
			// Auth endpoints get the stricter limit regardless of prefix.
			// Key on the trusted TCP peer (ClientIP), never on r.RemoteAddr or
			// any client header — see ClientIP for why.
			limiter := limiterStore.getLimiter(ClientIP(r), true)
			if !limiter.Allow() {
				writeRateLimitResponse(w, r)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Only rate-limit API routes.
		if len(r.URL.Path) < 7 || r.URL.Path[:7] != "/api/v1" {
			next.ServeHTTP(w, r)
			return
		}

		strict := isAPIAuth
		// Same trusted-IP key as the auth path above (see ClientIP).
		limiter := limiterStore.getLimiter(ClientIP(r), strict)

		if !limiter.Allow() {
			writeRateLimitResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeRateLimitResponse writes a JSON rate-limit response.
func writeRateLimitResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"rate limit exceeded","retry_after":60}`))
}
