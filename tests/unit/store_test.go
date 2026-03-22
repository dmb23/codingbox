package unit

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

func setupTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSessionStore_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-123",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusCreated,
		ConfigSnapshot: `{"name":"test"}`,
		CreatedAt:      time.Now().Truncate(time.Second),
	}

	err := ss.Create(session)
	require.NoError(t, err)

	got, err := ss.Get(session.ID)
	require.NoError(t, err)

	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, session.VMID, got.VMID)
	assert.Equal(t, session.VMSocketPath, got.VMSocketPath)
	assert.Equal(t, session.AgentName, got.AgentName)
	assert.Equal(t, session.Status, got.Status)
	assert.Equal(t, session.ConfigSnapshot, got.ConfigSnapshot)
}

func TestSessionStore_GetNotFound(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	_, err := ss.Get("nonexistent-id")
	assert.ErrorContains(t, err, "session not found")
}

func TestSessionStore_List(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	// Create multiple sessions
	for i, status := range []string{models.StatusCreated, models.StatusRunning, models.StatusStopped} {
		session := &models.SandboxSession{
			ID:             ulid.Make().String(),
			VMID:           "vm-" + string(rune('a'+i)),
			VMSocketPath:   "/tmp/vm.sock",
			AgentName:      "agent",
			Status:         status,
			ConfigSnapshot: `{"name":"test"}`,
			CreatedAt:      time.Now().Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, ss.Create(session))
	}

	// List all
	all, err := ss.List("")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// List filtered by status
	running, err := ss.List(models.StatusRunning)
	require.NoError(t, err)
	assert.Len(t, running, 1)
	assert.Equal(t, models.StatusRunning, running[0].Status)
}

func TestSessionStore_UpdateStatus_ValidTransition(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-1",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusCreated,
		ConfigSnapshot: `{"name":"test"}`,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, ss.Create(session))

	// created -> running
	err := ss.UpdateStatus(session.ID, models.StatusRunning, "")
	require.NoError(t, err)

	got, err := ss.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusRunning, got.Status)
	assert.NotNil(t, got.StartedAt)

	// running -> stopped
	err = ss.UpdateStatus(session.ID, models.StatusStopped, "")
	require.NoError(t, err)

	got, err = ss.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusStopped, got.Status)
	assert.NotNil(t, got.StoppedAt)
}

func TestSessionStore_UpdateStatus_InvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-1",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusCreated,
		ConfigSnapshot: `{"name":"test"}`,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, ss.Create(session))

	// created -> stopped is invalid
	err := ss.UpdateStatus(session.ID, models.StatusStopped, "")
	assert.ErrorContains(t, err, "invalid state transition")
}

func TestSessionStore_UpdateStatus_Failed(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-1",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusCreated,
		ConfigSnapshot: `{"name":"test"}`,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, ss.Create(session))

	err := ss.UpdateStatus(session.ID, models.StatusFailed, "VM creation timed out")
	require.NoError(t, err)

	got, err := ss.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusFailed, got.Status)
	assert.Equal(t, "VM creation timed out", got.ErrorMessage)
	assert.NotNil(t, got.StoppedAt)
}

func TestSessionStore_FindByName(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-1",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusRunning,
		ConfigSnapshot: `{"name":"my-project"}`,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, ss.Create(session))

	got, err := ss.FindByName("my-project")
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)
}

func TestSessionStore_FindByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)

	_, err := ss.FindByName("nonexistent")
	assert.Error(t, err)
}

// --- LogStore tests ---

func TestLogStore_InsertAndQuery(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)
	ls := store.NewLogStore(db)

	// Create a session first (FK constraint)
	session := &models.SandboxSession{
		ID:             ulid.Make().String(),
		VMID:           "vm-1",
		VMSocketPath:   "/tmp/vm.sock",
		AgentName:      "claude",
		Status:         models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, ss.Create(session))

	status200 := 200
	latency := int64(145)

	entry := &models.RequestLogEntry{
		ID:              ulid.Make().String(),
		SessionID:       session.ID,
		Method:          "POST",
		URL:             "https://api.openai.com/v1/chat/completions",
		Host:            "api.openai.com",
		RequestHeaders:  `{"Authorization":["Bearer placeholder-uuid"]}`,
		ResponseStatus:  &status200,
		ResponseHeaders: `{"Content-Type":["application/json"]}`,
		LatencyMs:       &latency,
		SecretsInjected: `["openai-key"]`,
		Timestamp:       time.Now(),
	}

	err := ls.Insert(entry)
	require.NoError(t, err)

	// Query by session
	entries, err := ls.Query(store.LogQuery{SessionID: session.ID})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	got := entries[0]
	assert.Equal(t, entry.ID, got.ID)
	assert.Equal(t, "POST", got.Method)
	assert.Equal(t, "api.openai.com", got.Host)
	assert.Equal(t, 200, *got.ResponseStatus)
	assert.Equal(t, int64(145), *got.LatencyMs)
	assert.Equal(t, `["openai-key"]`, got.SecretsInjected)
}

func TestLogStore_QueryByHost(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)
	ls := store.NewLogStore(db)

	session := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-1", VMSocketPath: "/tmp/vm.sock",
		AgentName: "claude", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session))

	hosts := []string{"api.openai.com", "api.github.com", "api.openai.com"}
	for _, host := range hosts {
		status200 := 200
		require.NoError(t, ls.Insert(&models.RequestLogEntry{
			ID: ulid.Make().String(), SessionID: session.ID,
			Method: "GET", URL: "https://" + host + "/test", Host: host,
			RequestHeaders: "{}", ResponseStatus: &status200,
			Timestamp: time.Now(),
		}))
	}

	entries, err := ls.Query(store.LogQuery{
		SessionID: session.ID,
		Host:      "api.openai.com",
	})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestLogStore_QueryByStatus(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)
	ls := store.NewLogStore(db)

	session := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-1", VMSocketPath: "/tmp/vm.sock",
		AgentName: "claude", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session))

	statuses := []int{200, 404, 200, 500}
	for _, s := range statuses {
		sc := s
		require.NoError(t, ls.Insert(&models.RequestLogEntry{
			ID: ulid.Make().String(), SessionID: session.ID,
			Method: "GET", URL: "https://example.com/test", Host: "example.com",
			RequestHeaders: "{}", ResponseStatus: &sc,
			Timestamp: time.Now(),
		}))
	}

	status404 := 404
	entries, err := ls.Query(store.LogQuery{
		SessionID: session.ID,
		Status:    &status404,
	})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogStore_QueryLimit(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)
	ls := store.NewLogStore(db)

	session := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-1", VMSocketPath: "/tmp/vm.sock",
		AgentName: "claude", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session))

	for i := 0; i < 10; i++ {
		status200 := 200
		require.NoError(t, ls.Insert(&models.RequestLogEntry{
			ID: ulid.Make().String(), SessionID: session.ID,
			Method: "GET", URL: "https://example.com/test", Host: "example.com",
			RequestHeaders: "{}", ResponseStatus: &status200,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}))
	}

	entries, err := ls.Query(store.LogQuery{
		SessionID: session.ID,
		Limit:     3,
	})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestLogStore_DeleteOlderThan(t *testing.T) {
	db := setupTestDB(t)
	ss := store.NewSessionStore(db)
	ls := store.NewLogStore(db)

	session := &models.SandboxSession{
		ID: ulid.Make().String(), VMID: "vm-1", VMSocketPath: "/tmp/vm.sock",
		AgentName: "claude", Status: models.StatusRunning,
		ConfigSnapshot: `{"name":"test"}`, CreatedAt: time.Now(),
	}
	require.NoError(t, ss.Create(session))

	// Insert an old entry and a recent entry
	status200 := 200
	require.NoError(t, ls.Insert(&models.RequestLogEntry{
		ID: ulid.Make().String(), SessionID: session.ID,
		Method: "GET", URL: "https://example.com/old", Host: "example.com",
		RequestHeaders: "{}", ResponseStatus: &status200,
		Timestamp: time.Now().AddDate(0, 0, -60), // 60 days ago
	}))
	require.NoError(t, ls.Insert(&models.RequestLogEntry{
		ID: ulid.Make().String(), SessionID: session.ID,
		Method: "GET", URL: "https://example.com/recent", Host: "example.com",
		RequestHeaders: "{}", ResponseStatus: &status200,
		Timestamp: time.Now(),
	}))

	deleted, err := ls.DeleteOlderThan(30)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	entries, err := ls.Query(store.LogQuery{SessionID: session.ID})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Contains(t, entries[0].URL, "recent")
}
