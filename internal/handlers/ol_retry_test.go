package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestParseRetryAfter tests the Retry-After header parser.
func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{
			name:     "decimal seconds",
			header:   "30",
			expected: 30 * time.Second,
		},
		{
			name:     "decimal seconds with whitespace",
			header:   "  5  ",
			expected: 5 * time.Second,
		},
		{
			name:     "zero seconds",
			header:   "0",
			expected: 0,
		},
		{
			name:     "empty string",
			header:   "",
			expected: 0,
		},
		{
			name:     "non-numeric string",
			header:   "wait",
			expected: 0,
		},
		{
			name:     "negative number",
			header:   "-5",
			expected: 0,
		},
		{
			name:     "large value",
			header:   "300",
			expected: 300 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRetryAfter(tc.header)
			if result != tc.expected {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.header, result, tc.expected)
			}
		})
	}
}

// TestExponentialBackoff tests the exponential backoff calculation.
func TestExponentialBackoff(t *testing.T) {
	base := 500 * time.Millisecond
	maxDelay := 5 * time.Second

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 0, expected: 500 * time.Millisecond},
		{attempt: 1, expected: 1 * time.Second},
		{attempt: 2, expected: 2 * time.Second},
		{attempt: 3, expected: 4 * time.Second},
		{attempt: 4, expected: 5 * time.Second}, // capped at maxDelay
		{attempt: 10, expected: 5 * time.Second}, // still capped
	}

	for _, tc := range tests {
		result := exponentialBackoff(tc.attempt, base, maxDelay)
		if result != tc.expected {
			t.Errorf("exponentialBackoff(%d, %v, %v) = %v, want %v",
				tc.attempt, base, maxDelay, result, tc.expected)
		}
	}
}

// TestSleepCtx tests the context-aware sleep function.
func TestSleepCtx(t *testing.T) {
	t.Run("completes normally", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		err := sleepCtx(ctx, 50*time.Millisecond)
		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if elapsed < 40*time.Millisecond {
			t.Errorf("sleep was too short: %v", elapsed)
		}
	})

	t.Run("cancels on context done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		start := time.Now()
		err := sleepCtx(ctx, 2*time.Second)
		elapsed := time.Since(start)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
		if elapsed > 100*time.Millisecond {
			t.Errorf("sleep should have returned immediately, took %v", elapsed)
		}
	})

	t.Run("zero duration returns immediately", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		err := sleepCtx(ctx, 0)
		if err != nil {
			t.Errorf("expected no error for zero duration, got %v", err)
		}
		if time.Since(start) > 10*time.Millisecond {
			t.Errorf("zero duration should return immediately")
		}
	})

	t.Run("negative duration returns immediately", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		err := sleepCtx(ctx, -1*time.Second)
		if err != nil {
			t.Errorf("expected no error for negative duration, got %v", err)
		}
		if time.Since(start) > 10*time.Millisecond {
			t.Errorf("negative duration should return immediately")
		}
	})
}

// TestOLRequestWithRetry_Success tests that a successful response returns immediately.
func TestOLRequestWithRetry_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"title":"Test Book"}`))
	}))
	defer server.Close()

	// Temporarily point olConfig at our test server.
	origBaseURL := olConfig.BaseURL
	olConfig.BaseURL = server.URL
	defer func() { olConfig.BaseURL = origBaseURL }()

	body, status, err := olRequestWithRetry(context.Background(), server.URL+"/test", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if string(body) != `{"title":"Test Book"}` {
		t.Errorf("unexpected body: %s", string(body))
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestOLRequestWithRetry_RetriesOn5xx tests that 5xx errors trigger retries.
func TestOLRequestWithRetry_RetriesOn5xx(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	body, status, err := olRequestWithRetry(context.Background(), server.URL, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", string(body))
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", callCount)
	}
}

// TestOLRequestWithRetry_ExhaustedRetries tests that retries are eventually exhausted.
func TestOLRequestWithRetry_ExhaustedRetries(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, status, err := olRequestWithRetry(context.Background(), server.URL, 2)
	if err == nil {
		t.Fatal("expected an error after exhausting retries")
	}
	if status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", status)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", callCount)
	}
}

// TestOLRequestWithRetry_ContextCancellation tests that a cancelled context
// stops retries promptly.
func TestOLRequestWithRetry_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response to give the context cancellation time to fire.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, _, err := olRequestWithRetry(ctx, server.URL, 2)
	if err == nil {
		t.Fatal("expected an error due to context cancellation")
	}
}

// TestOLRequestWithRetry_429HonoursRetryAfter tests that a 429 response with
// a Retry-After header causes the request to wait before retrying.
func TestOLRequestWithRetry_429HonoursRetryAfter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	start := time.Now()
	body, status, err := olRequestWithRetry(context.Background(), server.URL, 2)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", string(body))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	// With Retry-After: 0, the backoff should be minimal.
	// Just verify it completed in a reasonable time (< 5s).
	if elapsed > 5*time.Second {
		t.Errorf("request took too long: %v", elapsed)
	}
}

// TestOLRequestWithRetry_429WithoutRetryAfter tests that a 429 without Retry-After
// still retries (using the default exponential backoff).
func TestOLRequestWithRetry_429WithoutRetryAfter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	body, status, err := olRequestWithRetry(context.Background(), server.URL, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", string(body))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}
