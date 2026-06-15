package handlers

import (
	"os"
	"strconv"
	"time"
)

// OLConfig holds all configuration for Open Library API interactions.
type OLConfig struct {
	BaseURL         string
	CoversURL       string
	UserAgent       string
	HTTPTimeout     time.Duration
	CacheTTL        time.Duration
	RateLimitPerSec int
}

// LoadOLConfig reads Open Library configuration from environment variables
// with sensible defaults.
//
// Tests can override values by calling os.Setenv() before the handlers package
// is imported (e.g. in a TestMain or testinit pattern).
func LoadOLConfig() *OLConfig {
	return &OLConfig{
		BaseURL:         envOr("OL_BASE_URL", "https://openlibrary.org"),
		CoversURL:       envOr("OL_COVERS_URL", "https://covers.openlibrary.org"),
		UserAgent:       envOr("OL_USER_AGENT", "family-library/1.0 (https://github.com/Toph4er/family-library)"),
		HTTPTimeout:     durationOr("OL_HTTP_TIMEOUT", 10*time.Second),
		CacheTTL:        durationOr("OL_CACHE_TTL", 24*time.Hour),
		RateLimitPerSec: intOr("OL_RATE_LIMIT_PER_SEC", 2),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func intOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
