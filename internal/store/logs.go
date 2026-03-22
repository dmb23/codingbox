package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/codingbox/codingbox/internal/models"
)

// LogStore provides CRUD and query operations for request logs.
type LogStore struct {
	db *DB
}

// NewLogStore creates a new LogStore.
func NewLogStore(db *DB) *LogStore {
	return &LogStore{db: db}
}

// Insert persists a request log entry.
func (s *LogStore) Insert(entry *models.RequestLogEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO request_logs (id, session_id, method, url, host, request_headers, request_body,
		 response_status, response_headers, response_body, latency_ms, error, secrets_injected, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.SessionID, entry.Method, entry.URL, entry.Host,
		entry.RequestHeaders, entry.RequestBody,
		entry.ResponseStatus, entry.ResponseHeaders, entry.ResponseBody,
		entry.LatencyMs, entry.Error, entry.SecretsInjected,
		entry.Timestamp.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("inserting log entry: %w", err)
	}
	return nil
}

// LogQuery defines filters for querying request logs.
type LogQuery struct {
	SessionID string
	Host      string
	Status    *int
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// Query returns log entries matching the given filters.
func (s *LogStore) Query(q LogQuery) ([]*models.RequestLogEntry, error) {
	query := `SELECT id, session_id, method, url, host, request_headers, request_body,
		response_status, response_headers, response_body, latency_ms, error, secrets_injected, timestamp
		FROM request_logs WHERE session_id = ?`
	args := []any{q.SessionID}

	if q.Host != "" {
		query += " AND host = ?"
		args = append(args, q.Host)
	}
	if q.Status != nil {
		query += " AND response_status = ?"
		args = append(args, *q.Status)
	}
	if q.Since != nil {
		query += " AND timestamp >= ?"
		args = append(args, q.Since.Format(time.RFC3339Nano))
	}
	if q.Until != nil {
		query += " AND timestamp <= ?"
		args = append(args, q.Until.Format(time.RFC3339Nano))
	}

	query += " ORDER BY timestamp ASC"

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying logs: %w", err)
	}
	defer rows.Close()

	var entries []*models.RequestLogEntry
	for rows.Next() {
		entry, err := scanLogEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// DeleteOlderThan removes log entries older than the given duration.
func (s *LogStore) DeleteOlderThan(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339Nano)
	result, err := s.db.Exec("DELETE FROM request_logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("deleting old logs: %w", err)
	}
	return result.RowsAffected()
}

func scanLogEntry(rows *sql.Rows) (*models.RequestLogEntry, error) {
	var entry models.RequestLogEntry
	var timestamp string
	var responseStatus sql.NullInt64
	var responseHeaders, errMsg, secretsInjected sql.NullString
	var latencyMs sql.NullInt64
	var requestBody, responseBody []byte

	err := rows.Scan(
		&entry.ID, &entry.SessionID, &entry.Method, &entry.URL, &entry.Host,
		&entry.RequestHeaders, &requestBody,
		&responseStatus, &responseHeaders, &responseBody,
		&latencyMs, &errMsg, &secretsInjected, &timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning log entry: %w", err)
	}

	entry.Timestamp, _ = time.Parse(time.RFC3339Nano, timestamp)
	entry.RequestBody = requestBody
	entry.ResponseBody = responseBody

	if responseStatus.Valid {
		s := int(responseStatus.Int64)
		entry.ResponseStatus = &s
	}
	if responseHeaders.Valid {
		entry.ResponseHeaders = responseHeaders.String
	}
	if latencyMs.Valid {
		l := latencyMs.Int64
		entry.LatencyMs = &l
	}
	if errMsg.Valid {
		entry.Error = errMsg.String
	}
	if secretsInjected.Valid {
		entry.SecretsInjected = secretsInjected.String
	}

	return &entry, nil
}
