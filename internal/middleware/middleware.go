// Package middleware provides HTTP middleware for security, logging,
// rate limiting, and error recovery.
package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityHeaders adds security-related HTTP headers to every response.
//
// Headers set:
//   - Strict-Transport-Security (HSTS)
//   - X-Content-Type-Options
//   - X-Frame-Options
//   - Referrer-Policy
//   - Permissions-Policy
//   - Cross-Origin-Opener-Policy
//   - Content-Security-Policy
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(self), microphone=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' https://cdn.jsdelivr.net https://unpkg.com 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' fonts.googleapis.com; "+
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
			"remote_addr", r.RemoteAddr,
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
// Auth endpoints get a stricter limit (10 requests/hour) while general
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
		// ~10 requests per hour for auth endpoints (burst of 5, then 1 per 6 min).
		limiter = rate.NewLimiter(rate.Every(6*time.Minute), 5)
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

// RateLimiter enforces per-IP rate limiting on API requests.
//
// Auth endpoints (/api/v1/auth/*) are limited to 10 requests per hour.
// Other API endpoints are limited to 100 requests per minute.
// Static file requests are not rate-limited.
func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for static files and health checks.
		if r.URL.Path == "/health" || len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/static" {
			next.ServeHTTP(w, r)
			return
		}

		// Only rate-limit API routes.
		if len(r.URL.Path) < 7 || r.URL.Path[:7] != "/api/v1" {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		strict := len(r.URL.Path) >= 11 && r.URL.Path[:11] == "/api/v1/auth"

		limiter := limiterStore.getLimiter(ip, strict)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded","retry_after":60}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
