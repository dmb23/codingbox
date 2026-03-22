package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/codingbox/codingbox/internal/models"
)

// SessionStore provides CRUD operations for sandbox sessions.
type SessionStore struct {
	db *DB
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create inserts a new session record.
func (s *SessionStore) Create(session *models.SandboxSession) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, vm_id, vm_socket_path, agent_name, status, config_snapshot, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.VMID,
		session.VMSocketPath,
		session.AgentName,
		session.Status,
		session.ConfigSnapshot,
		session.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(id string) (*models.SandboxSession, error) {
	row := s.db.QueryRow(
		`SELECT id, vm_id, vm_socket_path, agent_name, status, config_snapshot, created_at, started_at, stopped_at, error_message
		 FROM sessions WHERE id = ?`, id)
	return scanSession(row)
}

// List returns sessions, optionally filtered by status.
func (s *SessionStore) List(statusFilter string) ([]*models.SandboxSession, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		rows, err = s.db.Query(
			`SELECT id, vm_id, vm_socket_path, agent_name, status, config_snapshot, created_at, started_at, stopped_at, error_message
			 FROM sessions WHERE status = ? ORDER BY created_at DESC`, statusFilter)
	} else {
		rows, err = s.db.Query(
			`SELECT id, vm_id, vm_socket_path, agent_name, status, config_snapshot, created_at, started_at, stopped_at, error_message
			 FROM sessions ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.SandboxSession
	for rows.Next() {
		session, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

// UpdateStatus transitions a session to a new status with state machine validation.
func (s *SessionStore) UpdateStatus(id, newStatus string, errorMessage string) error {
	session, err := s.Get(id)
	if err != nil {
		return fmt.Errorf("getting session for status update: %w", err)
	}

	if !session.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid state transition: %s -> %s", session.Status, newStatus)
	}

	now := time.Now().Format(time.RFC3339)

	switch newStatus {
	case models.StatusRunning:
		_, err = s.db.Exec(
			`UPDATE sessions SET status = ?, started_at = ? WHERE id = ?`,
			newStatus, now, id)
	case models.StatusStopped:
		_, err = s.db.Exec(
			`UPDATE sessions SET status = ?, stopped_at = ? WHERE id = ?`,
			newStatus, now, id)
	case models.StatusFailed:
		_, err = s.db.Exec(
			`UPDATE sessions SET status = ?, stopped_at = ?, error_message = ? WHERE id = ?`,
			newStatus, now, errorMessage, id)
	default:
		_, err = s.db.Exec(
			`UPDATE sessions SET status = ? WHERE id = ?`,
			newStatus, id)
	}

	if err != nil {
		return fmt.Errorf("updating session status: %w", err)
	}
	return nil
}

// FindByName finds a session by name prefix from the config snapshot.
func (s *SessionStore) FindByName(name string) (*models.SandboxSession, error) {
	row := s.db.QueryRow(
		`SELECT id, vm_id, vm_socket_path, agent_name, status, config_snapshot, created_at, started_at, stopped_at, error_message
		 FROM sessions WHERE json_extract(config_snapshot, '$.name') = ? ORDER BY created_at DESC LIMIT 1`, name)
	return scanSession(row)
}

func scanSession(row *sql.Row) (*models.SandboxSession, error) {
	var session models.SandboxSession
	var createdAt string
	var startedAt, stoppedAt, errorMessage sql.NullString

	err := row.Scan(
		&session.ID, &session.VMID, &session.VMSocketPath, &session.AgentName,
		&session.Status, &session.ConfigSnapshot, &createdAt,
		&startedAt, &stoppedAt, &errorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("scanning session: %w", err)
	}

	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		session.StartedAt = &t
	}
	if stoppedAt.Valid {
		t, _ := time.Parse(time.RFC3339, stoppedAt.String)
		session.StoppedAt = &t
	}
	if errorMessage.Valid {
		session.ErrorMessage = errorMessage.String
	}

	return &session, nil
}

func scanSessionRow(rows *sql.Rows) (*models.SandboxSession, error) {
	var session models.SandboxSession
	var createdAt string
	var startedAt, stoppedAt, errorMessage sql.NullString

	err := rows.Scan(
		&session.ID, &session.VMID, &session.VMSocketPath, &session.AgentName,
		&session.Status, &session.ConfigSnapshot, &createdAt,
		&startedAt, &stoppedAt, &errorMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning session row: %w", err)
	}

	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		session.StartedAt = &t
	}
	if stoppedAt.Valid {
		t, _ := time.Parse(time.RFC3339, stoppedAt.String)
		session.StoppedAt = &t
	}
	if errorMessage.Valid {
		session.ErrorMessage = errorMessage.String
	}

	return &session, nil
}
