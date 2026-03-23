package unit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mischa/codingbox/internal/models"
	"github.com/mischa/codingbox/internal/store"
)

func TestStoreInitAndInsert(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	log := &models.TrafficLog{
		SandboxID:       "test-123",
		Timestamp:       time.Now(),
		Method:          "GET",
		URL:             "https://example.com/api",
		RequestHeaders:  `{"Authorization":["Bearer token"]}`,
		ResponseStatus:  200,
		ResponseHeaders: `{"Content-Type":["application/json"]}`,
		ResponseBody:    []byte(`{"ok":true}`),
		SecretsReplaced: false,
		DurationMs:      42,
	}

	if err := st.InsertLog(log); err != nil {
		t.Fatalf("InsertLog: %v", err)
	}

	// Query back.
	logs, err := st.QueryLogs(store.LogFilter{SandboxID: "test-123"})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
	if logs[0].Method != "GET" {
		t.Errorf("Method = %q, want GET", logs[0].Method)
	}
	if logs[0].ResponseStatus != 200 {
		t.Errorf("ResponseStatus = %d, want 200", logs[0].ResponseStatus)
	}
}

func TestStoreQueryFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	now := time.Now()
	logs := []*models.TrafficLog{
		{SandboxID: "s1", Timestamp: now.Add(-3 * time.Second), Method: "GET", URL: "https://api.example.com/users", ResponseStatus: 200, DurationMs: 10},
		{SandboxID: "s1", Timestamp: now.Add(-2 * time.Second), Method: "POST", URL: "https://api.example.com/users", ResponseStatus: 201, DurationMs: 20},
		{SandboxID: "s1", Timestamp: now.Add(-1 * time.Second), Method: "GET", URL: "https://other.com/health", ResponseStatus: 500, DurationMs: 5},
		{SandboxID: "s2", Timestamp: now, Method: "GET", URL: "https://api.example.com/data", ResponseStatus: 200, DurationMs: 15},
	}
	for _, l := range logs {
		l.RequestHeaders = "{}"
		l.ResponseHeaders = "{}"
		if err := st.InsertLog(l); err != nil {
			t.Fatalf("InsertLog: %v", err)
		}
	}

	// Filter by method.
	result, _ := st.QueryLogs(store.LogFilter{Method: "POST"})
	if len(result) != 1 {
		t.Errorf("method filter: got %d, want 1", len(result))
	}

	// Filter by URL.
	result, _ = st.QueryLogs(store.LogFilter{URL: "example.com"})
	if len(result) != 3 {
		t.Errorf("URL filter: got %d, want 3", len(result))
	}

	// Filter by status.
	result, _ = st.QueryLogs(store.LogFilter{Status: 500})
	if len(result) != 1 {
		t.Errorf("status filter: got %d, want 1", len(result))
	}

	// Filter by session.
	result, _ = st.QueryLogs(store.LogFilter{SandboxID: "s2"})
	if len(result) != 1 {
		t.Errorf("session filter: got %d, want 1", len(result))
	}

	// Filter by limit.
	result, _ = st.QueryLogs(store.LogFilter{Limit: 2})
	if len(result) != 2 {
		t.Errorf("limit filter: got %d, want 2", len(result))
	}
}

func TestLatestSandboxID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	now := time.Now()
	st.InsertLog(&models.TrafficLog{SandboxID: "old", Timestamp: now.Add(-1 * time.Hour), Method: "GET", URL: "http://x", RequestHeaders: "{}", ResponseHeaders: "{}"})
	st.InsertLog(&models.TrafficLog{SandboxID: "new", Timestamp: now, Method: "GET", URL: "http://y", RequestHeaders: "{}", ResponseHeaders: "{}"})

	id, err := st.LatestSandboxID()
	if err != nil {
		t.Fatalf("LatestSandboxID: %v", err)
	}
	if id != "new" {
		t.Errorf("got %q, want 'new'", id)
	}
}
