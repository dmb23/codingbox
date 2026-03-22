package models

import "time"

// RequestLogEntry records a single HTTP request/response pair captured by the MITM proxy.
type RequestLogEntry struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	Method          string    `json:"method"`
	URL             string    `json:"url"`
	Host            string    `json:"host"`
	RequestHeaders  string    `json:"request_headers"`
	RequestBody     []byte    `json:"request_body,omitempty"`
	ResponseStatus  *int      `json:"response_status,omitempty"`
	ResponseHeaders string    `json:"response_headers,omitempty"`
	ResponseBody    []byte    `json:"response_body,omitempty"`
	LatencyMs       *int64    `json:"latency_ms,omitempty"`
	Error           string    `json:"error,omitempty"`
	SecretsInjected string    `json:"secrets_injected,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}
