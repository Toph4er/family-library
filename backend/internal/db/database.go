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
		database.Close()
		return nil, err
	}

	// Enable Write-Ahead Logging for better concurrent read performance
	// and safer backup behavior.
	if _, err := database.Exec("PRAGMA journal_mode = WAL"); err != nil {
		database.Close()
		return nil, err
	}

	// Set busy timeout to 10 seconds. This tells SQLite how long to wait
	// before returning a "database is locked" error when another connection
	// holds a write lock.
	if _, err := database.Exec("PRAGMA busy_timeout = 10000"); err != nil {
		database.Close()
		return nil, err
	}

	// Enable foreign key enforcement (disabled by default in SQLite).
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close()
		return nil, err
	}

	// Configure connection pool. SQLite doesn't benefit from a large pool
	// since the driver handles connections efficiently.
	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)

	return database, nil
}
