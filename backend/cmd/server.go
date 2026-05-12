// Package main is the entry point for the library server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	if err := os.MkdirAll(dbDir, 0755); err != nil {
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

	// -- Setup router --
	router := api.NewRouter(database, authSvc)

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
