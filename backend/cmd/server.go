// Package main is the entry point for the library server.
package main

import (
	"context"
	"flag"
	"encoding/json"
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

	"git.rcsmaine.com/chris/library/backend/internal/api"
	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/db"
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
		if err := runMigrations(database); err != nil {
			slog.Error("Migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Migrations applied successfully")
	}

	// -- Initialize auth --
	authSvc := auth.New(database, []byte(sessionSecret))

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
	tmpl, err := loadTemplates("./internal/templates")
	if err != nil {
		slog.Warn("Failed to load templates, SPA mode only", "error", err)
		tmpl = nil
	}

	// -- Setup router --
	router := api.NewRouter(database, authSvc, &api.RouterConfig{
		Templates: tmpl,
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

// loadTemplates parses all .html files in the given directory and returns
// a compiled *template.Template with a FuncMap of common helpers.
func loadTemplates(dir string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 at 3:04 PM")
		},
		"formatISBN": func(isbn string) string {
			// Format ISBN-13 with dashes: XXX-X-XX-XXXXX-X-X
			if len(isbn) == 13 {
				return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
					isbn[0:3], isbn[3:4], isbn[4:6], isbn[6:11], isbn[11:12], isbn[12:13])
			}
			return isbn
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
		"split": func(s string) []string {
			if s == "" {
				return []string{}
			}
			// Try JSON array first
			var result []string
			if err := json.Unmarshal([]byte(s), &result); err == nil {
				return result
			}
			// Fall back to comma-separated
			parts := strings.Split(s, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return parts
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
	}

	// Parse all .html files in the directory and subdirectories
	patterns := []string{
		filepath.Join(dir, "*.html"),
		filepath.Join(dir, "partials", "*.html"),
	}
	var allFiles []string
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob templates: %w", err)
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no template files found in %s", dir)
	}

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles(allFiles...)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return tmpl, nil
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
