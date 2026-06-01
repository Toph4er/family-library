package handlers

import (
	"os"
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
func LoadOLConfig() *OLConfig {
	return &OLConfig{
		BaseURL:     envOr("OL_BASE_URL", "https://openlibrary.org"),
		CoversURL:   envOr("OL_COVERS_URL", "https://covers.openlibrary.org"),
		UserAgent:   envOr("OL_USER_AGENT", "WoodlandLibrary/1.0 (personal children's library collection; contact@woodlandlibrary.local)"),
		HTTPTimeout: durationOr("OL_HTTP_TIMEOUT", 10*time.Second),
		CacheTTL:    durationOr("OL_CACHE_TTL", 24*time.Hour),
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
