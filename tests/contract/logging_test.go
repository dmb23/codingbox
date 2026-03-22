package contract

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*store.DB, *store.SessionStore, *store.LogStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, store.NewSessionStore(db), store.NewLogStore(db)
}

func createTestSession(t *testing.T, ss *store.SessionStore) *models.SandboxSession {
	t.Helper()
	session := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-1", VMSocketPath: "/tmp/vm.sock",
		AgentName: "claude", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session))
	return session
}

func TestLogging_AllRequiredFieldsPersisted(t *testing.T) {
	_, ss, ls := setupTestDB(t)
	session := createTestSession(t, ss)

	status200 := 200
	latency := int64(145)

	entry := &models.RequestLogEntry{
		ID:              ulid.Make().String(),
		SessionID:       session.ID,
		Method:          "POST",
		URL:             "https://api.openai.com/v1/chat/completions",
		Host:            "api.openai.com",
		RequestHeaders:  `{"Authorization":["Bearer placeholder-uuid"],"Content-Type":["application/json"]}`,
		RequestBody:     []byte(`{"model":"gpt-4"}`),
		ResponseStatus:  &status200,
		ResponseHeaders: `{"Content-Type":["application/json"]}`,
		ResponseBody:    []byte(`{"choices":[]}`),
		LatencyMs:       &latency,
		SecretsInjected: `["openai-key"]`,
		Timestamp:       time.Now(),
	}

	err := ls.Insert(entry)
	require.NoError(t, err)

	entries, err := ls.Query(store.LogQuery{SessionID: session.ID})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	got := entries[0]
	assert.Equal(t, entry.ID, got.ID, "id")
	assert.Equal(t, session.ID, got.SessionID, "session_id")
	assert.Equal(t, "POST", got.Method, "method")
	assert.Equal(t, "https://api.openai.com/v1/chat/completions", got.URL, "url")
	assert.Equal(t, "api.openai.com", got.Host, "host")
	assert.NotEmpty(t, got.RequestHeaders, "request_headers")
	assert.Equal(t, []byte(`{"model":"gpt-4"}`), got.RequestBody, "request_body")
	assert.NotNil(t, got.ResponseStatus, "response_status")
	assert.Equal(t, 200, *got.ResponseStatus, "response_status value")
	assert.NotEmpty(t, got.ResponseHeaders, "response_headers")
	assert.Equal(t, []byte(`{"choices":[]}`), got.ResponseBody, "response_body")
	assert.NotNil(t, got.LatencyMs, "latency_ms")
	assert.Equal(t, int64(145), *got.LatencyMs, "latency_ms value")
	assert.Equal(t, `["openai-key"]`, got.SecretsInjected, "secrets_injected")
	assert.False(t, got.Timestamp.IsZero(), "timestamp")
}

func TestLogging_FailedRequestHasErrorContext(t *testing.T) {
	_, ss, ls := setupTestDB(t)
	session := createTestSession(t, ss)

	entry := &models.RequestLogEntry{
		ID:             ulid.Make().String(),
		SessionID:      session.ID,
		Method:         "GET",
		URL:            "https://unreachable.example.com/api",
		Host:           "unreachable.example.com",
		RequestHeaders: "{}",
		Error:          "dial tcp: lookup unreachable.example.com: no such host",
		Timestamp:      time.Now(),
	}

	err := ls.Insert(entry)
	require.NoError(t, err)

	entries, err := ls.Query(store.LogQuery{SessionID: session.ID})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	got := entries[0]
	assert.Nil(t, got.ResponseStatus, "failed request should have nil status")
	assert.Contains(t, got.Error, "no such host")
}

func TestLogging_SessionCorrelation(t *testing.T) {
	_, ss, ls := setupTestDB(t)

	// Create two sessions
	session1 := createTestSession(t, ss)
	session2 := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-2", VMSocketPath: "/tmp/vm2.sock",
		AgentName: "codex", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test2"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session2))

	status200 := 200

	// Insert logs for both sessions
	for i := 0; i < 3; i++ {
		require.NoError(t, ls.Insert(&models.RequestLogEntry{
			ID: ulid.Make().String(), SessionID: session1.ID,
			Method: "GET", URL: "https://example.com/s1", Host: "example.com",
			RequestHeaders: "{}", ResponseStatus: &status200,
			Timestamp: time.Now(),
		}))
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, ls.Insert(&models.RequestLogEntry{
			ID: ulid.Make().String(), SessionID: session2.ID,
			Method: "GET", URL: "https://example.com/s2", Host: "example.com",
			RequestHeaders: "{}", ResponseStatus: &status200,
			Timestamp: time.Now(),
		}))
	}

	// Query session1 logs
	entries1, err := ls.Query(store.LogQuery{SessionID: session1.ID})
	require.NoError(t, err)
	assert.Len(t, entries1, 3)
	for _, e := range entries1 {
		assert.Equal(t, session1.ID, e.SessionID, "log should belong to session1")
	}

	// Query session2 logs
	entries2, err := ls.Query(store.LogQuery{SessionID: session2.ID})
	require.NoError(t, err)
	assert.Len(t, entries2, 2)
	for _, e := range entries2 {
		assert.Equal(t, session2.ID, e.SessionID, "log should belong to session2")
	}
}
