package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/mischa/codingbox/internal/models"
)

// InsertLog writes a traffic log entry to the database.
func (s *Store) InsertLog(log *models.TrafficLog) error {
	_, err := s.db.Exec(`
		INSERT INTO traffic_logs (sandbox_id, timestamp, method, url, request_headers, request_body,
			response_status, response_headers, response_body, secrets_replaced, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.SandboxID, log.Timestamp, log.Method, log.URL,
		log.RequestHeaders, log.RequestBody,
		log.ResponseStatus, log.ResponseHeaders, log.ResponseBody,
		log.SecretsReplaced, log.DurationMs,
	)
	return err
}

// LogFilter specifies criteria for querying traffic logs.
type LogFilter struct {
	SandboxID string
	Method    string
	URL       string
	Status    int
	Since     time.Time
	Limit     int
}

// QueryLogs retrieves traffic logs matching the given filter.
func (s *Store) QueryLogs(f LogFilter) ([]models.TrafficLog, error) {
	var conditions []string
	var args []any

	if f.SandboxID != "" {
		conditions = append(conditions, "sandbox_id = ?")
		args = append(args, f.SandboxID)
	}
	if f.Method != "" {
		conditions = append(conditions, "method = ?")
		args = append(args, f.Method)
	}
	if f.URL != "" {
		conditions = append(conditions, "url LIKE ?")
		args = append(args, "%"+f.URL+"%")
	}
	if f.Status != 0 {
		conditions = append(conditions, "response_status = ?")
		args = append(args, f.Status)
	}
	if !f.Since.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, f.Since)
	}

	query := "SELECT id, sandbox_id, timestamp, method, url, request_headers, request_body, response_status, response_headers, response_body, secrets_replaced, duration_ms FROM traffic_logs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY timestamp DESC"

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TrafficLog
	for rows.Next() {
		var l models.TrafficLog
		if err := rows.Scan(&l.ID, &l.SandboxID, &l.Timestamp, &l.Method, &l.URL,
			&l.RequestHeaders, &l.RequestBody, &l.ResponseStatus,
			&l.ResponseHeaders, &l.ResponseBody, &l.SecretsReplaced, &l.DurationMs); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// LatestSandboxID returns the most recent sandbox session ID.
func (s *Store) LatestSandboxID() (string, error) {
	var id string
	err := s.db.QueryRow("SELECT sandbox_id FROM traffic_logs ORDER BY timestamp DESC LIMIT 1").Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}
