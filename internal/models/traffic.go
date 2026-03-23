package models

import "time"

// TrafficLog records a single proxied HTTP request-response pair.
type TrafficLog struct {
	ID              int64     `json:"id" db:"id"`
	SandboxID       string    `json:"sandbox_id" db:"sandbox_id"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	Method          string    `json:"method" db:"method"`
	URL             string    `json:"url" db:"url"`
	RequestHeaders  string    `json:"request_headers" db:"request_headers"`   // JSON-encoded
	RequestBody     []byte    `json:"request_body" db:"request_body"`
	ResponseStatus  int       `json:"response_status" db:"response_status"`
	ResponseHeaders string    `json:"response_headers" db:"response_headers"` // JSON-encoded
	ResponseBody    []byte    `json:"response_body" db:"response_body"`
	SecretsReplaced bool      `json:"secrets_replaced" db:"secrets_replaced"`
	DurationMs      int64     `json:"duration_ms" db:"duration_ms"`
}
