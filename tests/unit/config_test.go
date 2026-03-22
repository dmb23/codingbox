package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSandboxConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test-sandbox
agent: claude
workspace: /tmp/workspace
mounts:
  - host: /tmp/readonly
    sandbox: /mnt/readonly
    mode: ro
  - host: /tmp/readwrite
    sandbox: /mnt/readwrite
    mode: rw
secrets:
  - name: openai-key
    host: api.openai.com
    header: Authorization
    template: "Bearer {secret}"
    value: sk-test-123
proxy:
  port: 0
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := config.LoadSandboxConfig(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "test-sandbox", cfg.Name)
	assert.Equal(t, "claude", cfg.Agent)
	assert.Equal(t, "/tmp/workspace", cfg.WorkspaceDir)
	assert.Len(t, cfg.Mounts, 2)
	assert.Equal(t, "ro", cfg.Mounts[0].Mode)
	assert.Equal(t, "rw", cfg.Mounts[1].Mode)
	assert.Len(t, cfg.Secrets, 1)
	assert.Equal(t, "openai-key", cfg.Secrets[0].Name)
	assert.Equal(t, "api.openai.com", cfg.Secrets[0].TargetHost)
	assert.Equal(t, "Authorization", cfg.Secrets[0].HeaderName)
	assert.Equal(t, "Bearer {secret}", cfg.Secrets[0].HeaderTemplate)
	assert.Equal(t, "sk-test-123", cfg.Secrets[0].SecretValue)
	// Placeholder UUID should be assigned
	assert.NotEmpty(t, cfg.Secrets[0].ID)
}

func TestLoadSandboxConfig_MissingName(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
agent: claude
workspace: /tmp/workspace
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := config.LoadSandboxConfig(cfgPath)
	assert.ErrorContains(t, err, "'name' is required")
}

func TestLoadSandboxConfig_MissingAgent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test
workspace: /tmp/workspace
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := config.LoadSandboxConfig(cfgPath)
	assert.ErrorContains(t, err, "'agent' is required")
}

func TestLoadSandboxConfig_MissingWorkspace(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test
agent: claude
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := config.LoadSandboxConfig(cfgPath)
	assert.ErrorContains(t, err, "'workspace' is required")
}

func TestLoadSandboxConfig_RelativeWorkspace(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test
agent: claude
workspace: relative/path
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := config.LoadSandboxConfig(cfgPath)
	assert.ErrorContains(t, err, "must be an absolute path")
}

func TestLoadSandboxConfig_InvalidMountMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test
agent: claude
workspace: /tmp/ws
mounts:
  - host: /tmp/foo
    sandbox: /mnt/foo
    mode: rwx
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	_, err := config.LoadSandboxConfig(cfgPath)
	assert.ErrorContains(t, err, "'mode' must be 'rw' or 'ro'")
}

func TestLoadSandboxConfig_DefaultMountMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yml")

	yaml := `
name: test
agent: claude
workspace: /tmp/ws
mounts:
  - host: /tmp/foo
    sandbox: /mnt/foo
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := config.LoadSandboxConfig(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "ro", cfg.Mounts[0].Mode)
}

func TestLoadSandboxConfig_SecretMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		errText string
	}{
		{
			name: "missing secret name",
			yaml: `
name: test
agent: claude
workspace: /tmp/ws
secrets:
  - host: api.example.com
    header: Authorization
    template: "Bearer {secret}"
    value: secret123
`,
			errText: "'name' is required",
		},
		{
			name: "missing secret host",
			yaml: `
name: test
agent: claude
workspace: /tmp/ws
secrets:
  - name: my-key
    header: Authorization
    template: "Bearer {secret}"
    value: secret123
`,
			errText: "'host' is required",
		},
		{
			name: "missing secret header",
			yaml: `
name: test
agent: claude
workspace: /tmp/ws
secrets:
  - name: my-key
    host: api.example.com
    template: "Bearer {secret}"
    value: secret123
`,
			errText: "'header' is required",
		},
		{
			name: "missing secret template",
			yaml: `
name: test
agent: claude
workspace: /tmp/ws
secrets:
  - name: my-key
    host: api.example.com
    header: Authorization
    value: secret123
`,
			errText: "'template' is required",
		},
		{
			name: "missing secret value",
			yaml: `
name: test
agent: claude
workspace: /tmp/ws
secrets:
  - name: my-key
    host: api.example.com
    header: Authorization
    template: "Bearer {secret}"
`,
			errText: "'value' is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "codingbox.yml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.yaml), 0644))

			_, err := config.LoadSandboxConfig(cfgPath)
			assert.ErrorContains(t, err, tt.errText)
		})
	}
}

func TestLoadSandboxConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadSandboxConfig("/nonexistent/path.yml")
	assert.Error(t, err)
}

func TestLoadGlobalConfig_Defaults(t *testing.T) {
	cfg, err := config.LoadGlobalConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.DBPath)
	assert.Equal(t, 30, cfg.LogRetentionDays)
	assert.NotEmpty(t, cfg.CACertPath)
	assert.NotEmpty(t, cfg.CAKeyPath)
}
