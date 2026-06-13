// Package tests provides integration and unit tests for the library backend.
package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/db"
	"git.rcsmaine.com/chris/library/internal/handlers"
)

const (
	testUsername      = "admin"
	testPassword      = "password123"
	testGuestPassword = "guest123"
)

// testEnv holds the dependencies for a test.
type testEnv struct {
	db   *sql.DB
	auth *auth.Auth
}

// setupTestEnv creates a fresh in-memory database with all migrations applied,
// seeds an admin user and guest password, and returns a configured auth service.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	runMigrations(t, database)

	// Seed admin user
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	_, err = database.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')",
		testUsername, hash,
	)
	if err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	// Seed guest password
	guestHash, err := auth.HashPassword(testGuestPassword)
	if err != nil {
		t.Fatalf("failed to hash guest password: %v", err)
	}
	_, err = database.Exec(
		"UPDATE settings SET value = ? WHERE key = 'guest_password_hash'",
		guestHash,
	)
	if err != nil {
		t.Fatalf("failed to seed guest password: %v", err)
	}

	// Create auth service with test secret
	secret := []byte("test-session-secret-that-is-32-bytes!!")
	authSvc := auth.New(database, secret)
	authSvc.Store().Options.Secure = false

	return &testEnv{db: database, auth: authSvc}
}

// runMigrations reads all SQL migration files from the migrations directory
// and executes the "Up" portion of each in order.
func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	migrationsDir := filepath.Join("..", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
		if err != nil {
			t.Fatalf("failed to read migration file %s: %v", f.Name(), err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		_, err = db.Exec(upSQL)
		if err != nil {
			t.Fatalf("failed to execute migration %s: %v", f.Name(), err)
		}
	}
}

// extractUpMigration extracts the SQL between "-- +goose Up" and "-- +goose Down".
func extractUpMigration(content string) string {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"

	upIdx := strings.Index(content, upMarker)
	if upIdx == -1 {
		return ""
	}

	upStart := upIdx + len(upMarker)

	downIdx := strings.Index(content[upStart:], downMarker)
	if downIdx == -1 {
		return strings.TrimSpace(content[upStart:])
	}

	return strings.TrimSpace(content[upStart : upStart+downIdx])
}

// getSessionCookie extracts the library session cookie from a response recorder.
func getSessionCookie(rec *httptest.ResponseRecorder) string {
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == auth.SessionID {
			return fmt.Sprintf("%s=%s", c.Name, c.Value)
		}
	}
	return ""
}

// loginAndGetCookie performs a login and returns the session cookie.
// Fails the test if login is unsuccessful.
func loginAndGetCookie(t *testing.T, env *testEnv) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, testUsername, testPassword)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(env.auth).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status %d, body: %s", rec.Code, rec.Body.String())
	}

	cookie := getSessionCookie(rec)
	if cookie == "" {
		t.Fatal("no session cookie in login response")
	}

	return cookie
}

// setURLParam adds a chi URL parameter to the request context.
func setURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// buildAuthRouter creates a chi router that wraps the given handler with
// RequireAuth middleware, so we can test protected endpoints with a session cookie.
func buildAuthRouter(t *testing.T, env *testEnv, method, path string, handler http.HandlerFunc) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(path, env.auth.RequireAuth(handler))
	return r
}

// buildAdminRouter creates a chi router that wraps the given handler with
// RequireAdmin middleware.
func buildAdminRouter(t *testing.T, env *testEnv, method, path string, handler http.HandlerFunc) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(path, env.auth.RequireAdmin(handler))
	return r
}
