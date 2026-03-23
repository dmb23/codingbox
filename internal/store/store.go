package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for traffic log storage.
type Store struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at the given path.
func Open(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure for concurrent access.
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting synchronous mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying sql.DB for direct queries if needed.
func (s *Store) DB() *sql.DB {
	return s.db
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS traffic_logs (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		sandbox_id      TEXT NOT NULL,
		timestamp       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		method          TEXT NOT NULL,
		url             TEXT NOT NULL,
		request_headers TEXT NOT NULL DEFAULT '{}',
		request_body    BLOB,
		response_status INTEGER NOT NULL DEFAULT 0,
		response_headers TEXT NOT NULL DEFAULT '{}',
		response_body   BLOB,
		secrets_replaced INTEGER NOT NULL DEFAULT 0,
		duration_ms     INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_traffic_sandbox_id ON traffic_logs(sandbox_id);
	CREATE INDEX IF NOT EXISTS idx_traffic_timestamp ON traffic_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_traffic_url ON traffic_logs(url);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrating database: %w", err)
	}
	return nil
}

// DefaultDBPath returns the default traffic log database path.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codingbox", "traffic.db")
}
