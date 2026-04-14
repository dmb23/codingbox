package unit

import (
	"os"
	"strings"
	"testing"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/models"
)

func TestResolveEnvSecrets_ReadsFromHostEnv(t *testing.T) {
	os.Setenv("TEST_RESOLVE_KEY", "host-secret-value")
	defer os.Unsetenv("TEST_RESOLVE_KEY")

	cfg := &models.SandboxConfig{
		Secrets: []models.SecretMapping{
			{Env: "TEST_RESOLVE_KEY", ReplaceIn: []string{"headers"}},
		},
	}

	if err := config.ResolveEnvSecrets(cfg); err != nil {
		t.Fatalf("ResolveEnvSecrets: %v", err)
	}

	s := cfg.Secrets[0]
	if s.Value != "host-secret-value" {
		t.Errorf("Value = %q, want 'host-secret-value'", s.Value)
	}
	if s.Placeholder == "" {
		t.Error("Placeholder not generated")
	}
	if !strings.Contains(s.Placeholder, "TEST_RESOLVE_KEY") {
		t.Errorf("Placeholder should contain env name: %q", s.Placeholder)
	}
}

func TestResolveEnvSecrets_ExplicitValueOverride(t *testing.T) {
	os.Setenv("TEST_OVERRIDE_KEY", "from-host")
	defer os.Unsetenv("TEST_OVERRIDE_KEY")

	cfg := &models.SandboxConfig{
		Secrets: []models.SecretMapping{
			{Env: "TEST_OVERRIDE_KEY", Value: "explicit-override"},
		},
	}

	if err := config.ResolveEnvSecrets(cfg); err != nil {
		t.Fatalf("ResolveEnvSecrets: %v", err)
	}

	if cfg.Secrets[0].Value != "explicit-override" {
		t.Errorf("Value = %q, want 'explicit-override' (should not read from host)", cfg.Secrets[0].Value)
	}
}

func TestResolveEnvSecrets_MissingHostEnv(t *testing.T) {
	os.Unsetenv("NONEXISTENT_SECRET_VAR_XYZ")

	cfg := &models.SandboxConfig{
		Secrets: []models.SecretMapping{
			{Env: "NONEXISTENT_SECRET_VAR_XYZ"},
		},
	}

	err := config.ResolveEnvSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for missing host env var")
	}
	if !strings.Contains(err.Error(), "NONEXISTENT_SECRET_VAR_XYZ") {
		t.Errorf("error should mention env var name: %v", err)
	}
}

func TestResolveEnvSecrets_DuplicateEnvName(t *testing.T) {
	os.Setenv("DUPE_KEY", "value")
	defer os.Unsetenv("DUPE_KEY")

	cfg := &models.SandboxConfig{
		Secrets: []models.SecretMapping{
			{Env: "DUPE_KEY"},
			{Env: "DUPE_KEY"},
		},
	}

	err := config.ResolveEnvSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate env name")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate: %v", err)
	}
}

func TestResolveEnvSecrets_RejectsMissingEnv(t *testing.T) {
	cfg := &models.SandboxConfig{
		Secrets: []models.SecretMapping{
			{Value: "orphan-value"},
		},
	}

	err := config.ResolveEnvSecrets(cfg)
	if err == nil {
		t.Fatal("expected error when env is not set")
	}
}

func TestValidate_AcceptsResolvedEnvSecret(t *testing.T) {
	cfg := &models.SandboxConfig{
		Image:   "ubuntu:22.04",
		Workdir: ".",
		Secrets: []models.SecretMapping{
			{Env: "MY_KEY", Placeholder: "__CODINGBOX_MY_KEY_abc123__", Value: "val"},
		},
	}

	err := config.Validate(cfg)
	if err != nil {
		t.Fatalf("should accept resolved env secret: %v", err)
	}
}

func TestValidate_RejectsMissingEnv(t *testing.T) {
	cfg := &models.SandboxConfig{
		Image:   "ubuntu:22.04",
		Workdir: ".",
		Secrets: []models.SecretMapping{
			{Value: "orphan-value"},
		},
	}

	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected error when env is not set")
	}
}

func TestParseEnvSecretFlag_Basic(t *testing.T) {
	s, err := config.ParseEnvSecretFlag("ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatalf("ParseEnvSecretFlag: %v", err)
	}
	if s.Env != "ANTHROPIC_API_KEY" {
		t.Errorf("Env = %q, want ANTHROPIC_API_KEY", s.Env)
	}
	if len(s.ReplaceIn) != 3 {
		t.Errorf("ReplaceIn should default to all 3, got %v", s.ReplaceIn)
	}
}

func TestParseEnvSecretFlag_WithLocations(t *testing.T) {
	s, err := config.ParseEnvSecretFlag("MY_TOKEN:headers,body")
	if err != nil {
		t.Fatalf("ParseEnvSecretFlag: %v", err)
	}
	if s.Env != "MY_TOKEN" {
		t.Errorf("Env = %q, want MY_TOKEN", s.Env)
	}
	if len(s.ReplaceIn) != 2 || s.ReplaceIn[0] != "headers" || s.ReplaceIn[1] != "body" {
		t.Errorf("ReplaceIn = %v, want [headers body]", s.ReplaceIn)
	}
}
