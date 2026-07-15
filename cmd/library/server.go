// Package main is the entry point for the library server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"database/sql"

	"github.com/pressly/goose/v3"

	"github.com/Toph4er/family-library/internal/api"
	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/db"
)

func main() {
	// -- CLI flags --
	migrate := flag.Bool("migrate", false, "Run database migrations on startup")
	flag.Parse()

	// -- Configuration from environment --
	port := getEnvOrDefault("PORT", "8080")
	dbPath := getEnvOrDefault("DATABASE_PATH", "/app/data/library.db")
	sessionSecret := os.Getenv("SESSION_SECRET")
	logLevel := getEnvOrDefault("LOG_LEVEL", "info")

	// -- Validate required config --
	if sessionSecret == "" {
		slog.Error("SESSION_SECRET is required")
		os.Exit(1)
	}
	if len(sessionSecret) < 32 {
		slog.Error("SESSION_SECRET must be at least 32 characters", "length", len(sessionSecret))
		os.Exit(1)
	}

	// -- Setup logging --
	setupLogging(logLevel)
	slog.Info("Starting library server", "port", port, "db", dbPath)

	// -- Ensure database directory exists --
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0750); err != nil {
		slog.Error("Failed to create database directory", "dir", dbDir, "error", err)
		os.Exit(1)
	}

	// -- Initialize database --
	database, err := db.Open(dbPath)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// -- Run migrations if requested --
	if *migrate {
		if err := runMigrations(database.DB); err != nil {
			slog.Error("Migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Migrations applied successfully")
	}

	// -- Initialize auth --
	authSvc := auth.New(database.DB, []byte(sessionSecret))

	// -- Seed initial admin user --
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminUsername != "" && adminPassword != "" {
		if err := authSvc.SeedAdminUser(adminUsername, adminPassword); err != nil {
			slog.Error("Failed to seed admin user", "error", err)
			os.Exit(1)
		}
		slog.Info("Admin user seeded", "username", adminUsername)
	}

	// -- Seed initial guest password --
	guestPassword := os.Getenv("GUEST_PASSWORD")
	if guestPassword != "" {
		if err := authSvc.SeedGuestPassword(guestPassword); err != nil {
			slog.Error("Failed to seed guest password", "error", err)
			os.Exit(1)
		}
		slog.Info("Guest password seeded")
	}

	// -- Load HTML templates --
	tmpls, err := loadTemplates("./internal/web")
	if err != nil {
		slog.Error("Failed to load templates", "error", err)
		os.Exit(1)
	}

	// -- Setup router --
	router := api.NewRouter(database, authSvc, &api.RouterConfig{
		Templates: tmpls,
	})

	// -- Create HTTP server --
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// -- Graceful shutdown --
	go func() {
		slog.Info("Server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}

// loadTemplates parses each page template in isolation so that {{define}}
// blocks (e.g. "content", "title") do not leak across pages.
// It returns a map keyed by page name (e.g. "login", "wishlist") where each
// value is an independently parsed *template.Template that includes base.html,
// the page's own HTML file, and all shared partials.
func loadTemplates(dir string) (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"formatTime": func(s string, tzName string) string {
			if tzName == "" {
				tzName = "America/New_York"
			}
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return s
			}
			t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
			if err != nil {
				return s
			}
			return t.In(loc).Format("Jan 2, 2006 at 3:04 PM")
		},
		"formatISBN": func(isbn interface{}) string {
			var s string
			switch v := isbn.(type) {
			case string:
				s = v
			case *string:
				if v == nil {
					return ""
				}
				s = *v
			default:
				return fmt.Sprintf("%v", isbn)
			}
			// Format ISBN-13 with dashes: XXX-X-XX-XXXXX-X-X
			if len(s) == 13 {
				return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
					s[0:3], s[3:4], s[4:6], s[6:11], s[11:12], s[12:13])
			}
			return s
		},
		"year": func() int {
			return time.Now().Year()
		},
		// --- Sequence helpers ---
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		// --- String helpers ---
		"join": func(sep string, parts []string) string {
			return strings.Join(parts, sep)
		},
		"split": func(s interface{}) []string {
			var str string
			switch v := s.(type) {
			case string:
				str = v
			case *string:
				if v == nil {
					return []string{}
				}
				str = *v
			default:
				return []string{}
			}
			if str == "" {
				return []string{}
			}
			// Try JSON array first
			var result []string
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result
			}
			// Fall back to comma-separated
			parts := strings.Split(str, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return parts
		},
		// parseJSONArray converts a JSON array string (e.g. ["A","B"]) to a
		// comma-separated display string. If the input is not a JSON array,
		// it returns it unchanged. Used in form inputs for fields stored as
		// JSON arrays.
		"parseJSONArray": func(s string) string {
			if s == "" {
				return ""
			}
			var result []string
			if err := json.Unmarshal([]byte(s), &result); err == nil {
				return strings.Join(result, ", ")
			}
			return s
		},
		// --- Comparison helpers ---
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		// --- Map helpers ---
		"dict": func(args ...interface{}) (map[string]interface{}, error) {
			if len(args)%2 != 0 {
				return nil, fmt.Errorf("dict expects an even number of arguments, got %d", len(args))
			}
			result := make(map[string]interface{}, len(args)/2)
			for i := 0; i < len(args); i += 2 {
				key, ok := args[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings, got %T", args[i])
				}
				result[key] = args[i+1]
			}
			return result, nil
		},
		// --- Humanize setting keys for display ---
		"humanizeKey": func(key string) string {
			parts := strings.Split(key, "_")
			for i, p := range parts {
				if len(p) > 0 {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return strings.Join(parts, " ")
		},
		// --- Guest visibility fields in a consistent order ---
		"guestVisibilityFields": func() []string {
			return []string{
				"title", "subtitle", "authors", "illustrators",
				"publisher", "publication_year", "page_count", "quantity", "book_type",
				"reading_levels", "genres", "themes", "awards",
				"gift_from", "gift_relationship", "child_rating", "read_count",
				"cover_image_url", "cover_source",
				"isbn", "condition", "location", "notes",
				"date_received", "last_read_date",
			}
		},
	}

	// Collect all page templates (exclude base.html which is shared)
	pageFiles, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	// Collect all partial templates
	partialFiles, err := filepath.Glob(filepath.Join(dir, "partials", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("glob partial templates: %w", err)
	}

	// base.html is the shared layout — every page needs it
	baseFile := filepath.Join(dir, "base.html")

	templates := make(map[string]*template.Template)

	for _, pageFile := range pageFiles {
		// Skip base.html — it's the shared layout, not a page
		if pageFile == baseFile {
			continue
		}

		// Derive the page key from the filename (e.g. "login.html" → "login")
		pageName := strings.TrimSuffix(filepath.Base(pageFile), ".html")

		// Build the file list for this page: base + page + partials
		files := append([]string{baseFile, pageFile}, partialFiles...)

		tmpl, err := template.New("").Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", pageName, err)
		}

		templates[pageName] = tmpl
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no page templates found in %s", dir)
	}

	return templates, nil
}

func runMigrations(database *sql.DB) error {
	// Find migrations directory relative to executable or use default
	migrationsDir := "./migrations"

	// Try to find migrations directory
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try relative to working directory
		migrationsDir = "migrations"
	}

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(database, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupLogging(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	})))
}
