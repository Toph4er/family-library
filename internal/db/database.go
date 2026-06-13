// Package db provides database initialization and connection management.
package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database connection at the given path.
//
// It enables WAL mode for better concurrency and sets a busy timeout
// to handle concurrent read/write contention gracefully.
//
// The returned *sql.DB is safe for concurrent use by multiple goroutines.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Verify the connection is alive.
	if err := database.Ping(); err != nil {
		database.Close() // #nosec G104 — cleanup on error path, primary error is returned
		return nil, err
	}

	// Enable Write-Ahead Logging for better concurrent read performance
	// and safer backup behavior.
	if _, err := database.Exec("PRAGMA journal_mode = WAL"); err != nil {
		database.Close() // #nosec G104 — cleanup on error path, primary error is returned
		return nil, err
	}

	// Set busy timeout to 10 seconds. This tells SQLite how long to wait
	// before returning a "database is locked" error when another connection
	// holds a write lock.
	if _, err := database.Exec("PRAGMA busy_timeout = 10000"); err != nil {
		database.Close() // #nosec G104 — cleanup on error path, primary error is returned
		return nil, err
	}

	// Enable foreign key enforcement (disabled by default in SQLite).
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close() // #nosec G104 — cleanup on error path, primary error is returned
		return nil, err
	}

	// Configure connection pool. SQLite is a file-based database that uses
	// file-level locking for concurrency. A single connection is sufficient
	// for most workloads; WAL mode allows concurrent readers with one writer.
	// We use 2 connections as a small buffer for concurrent reads during writes.
	database.SetMaxOpenConns(2)
	database.SetMaxIdleConns(2)

	return database, nil
}
