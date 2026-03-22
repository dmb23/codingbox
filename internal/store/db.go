package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	*sql.DB
}

// Open opens or creates the SQLite database at the given path and runs migrations.
func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY NOT NULL,
			vm_id TEXT NOT NULL,
			vm_socket_path TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('created','running','stopped','failed')),
			config_snapshot JSON NOT NULL,
			created_at TEXT NOT NULL,
			started_at TEXT,
			stopped_at TEXT,
			error_message TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent_name ON sessions(agent_name)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id TEXT PRIMARY KEY NOT NULL,
			session_id TEXT NOT NULL REFERENCES sessions(id),
			method TEXT NOT NULL,
			url TEXT NOT NULL,
			host TEXT NOT NULL,
			request_headers JSON NOT NULL,
			request_body BLOB,
			response_status INTEGER,
			response_headers JSON,
			response_body BLOB,
			latency_ms INTEGER,
			error TEXT,
			secrets_injected JSON,
			timestamp TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_session_id ON request_logs(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_host ON request_logs(host)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_response_status ON request_logs(response_status)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w\nSQL: %s", err, m)
		}
	}

	return nil
}
