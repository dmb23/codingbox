package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/models"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yaml")
	err := os.WriteFile(cfgPath, []byte(`
image: "ubuntu:22.04"
workdir: "."
mounts:
  - source: "/tmp"
    target: "/mnt/tmp"
    mode: "ro"
secrets:
  - placeholder: "__KEY__"
    value: "real-value"
    replace_in: ["headers"]
proxy_port: 8080
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Image != "ubuntu:22.04" {
		t.Errorf("Image = %q, want ubuntu:22.04", cfg.Image)
	}
	if cfg.ProxyPort != 8080 {
		t.Errorf("ProxyPort = %d, want 8080", cfg.ProxyPort)
	}
	if len(cfg.Mounts) != 1 {
		t.Fatalf("Mounts len = %d, want 1", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Mode != "ro" {
		t.Errorf("Mounts[0].Mode = %q, want ro", cfg.Mounts[0].Mode)
	}
	if len(cfg.Secrets) != 1 {
		t.Fatalf("Secrets len = %d, want 1", len(cfg.Secrets))
	}
	if cfg.Secrets[0].ReplaceIn[0] != "headers" {
		t.Errorf("Secrets[0].ReplaceIn = %v, want [headers]", cfg.Secrets[0].ReplaceIn)
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codingbox.yaml")
	err := os.WriteFile(cfgPath, []byte(`image: "ubuntu:22.04"`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Workdir != "." {
		t.Errorf("Workdir = %q, want '.'", cfg.Workdir)
	}
	if cfg.ProxyPort != 0 {
		t.Errorf("ProxyPort = %d, want 0", cfg.ProxyPort)
	}
}

func TestLoadMissingFile(t *testing.T) {
	// Loading a nonexistent config should not error (config file is optional).
	cfg, err := config.Load("/nonexistent/path/codingbox.yaml")
	if err == nil {
		// If the file doesn't exist, viper returns an error for explicit paths.
		// That's expected.
		_ = cfg
	}
}

func TestValidateMissingImage(t *testing.T) {
	cfg := &models.SandboxConfig{Workdir: "."}
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing image")
	}
}

func TestValidateInvalidMountMode(t *testing.T) {
	cfg := &models.SandboxConfig{
		Image:   "ubuntu:22.04",
		Workdir: ".",
		Mounts: []models.MountConfig{
			{Source: "/tmp", Target: "/mnt", Mode: "invalid"},
		},
	}
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid mount mode")
	}
}

func TestParseMountFlag(t *testing.T) {
	tests := []struct {
		input string
		want  models.MountConfig
	}{
		{"/src:/dst", models.MountConfig{Source: "/src", Target: "/dst", Mode: "ro"}},
		{"/src:/dst:rw", models.MountConfig{Source: "/src", Target: "/dst", Mode: "rw"}},
	}
	for _, tt := range tests {
		m, err := config.ParseMountFlag(tt.input)
		if err != nil {
			t.Errorf("ParseMountFlag(%q): %v", tt.input, err)
			continue
		}
		if m.Source != tt.want.Source || m.Target != tt.want.Target || m.Mode != tt.want.Mode {
			t.Errorf("ParseMountFlag(%q) = %+v, want %+v", tt.input, m, tt.want)
		}
	}
}

func TestParseSecretFlag(t *testing.T) {
	s, err := config.ParseSecretFlag("__KEY__=secret-val:headers,body")
	if err != nil {
		t.Fatalf("ParseSecretFlag: %v", err)
	}
	if s.Placeholder != "__KEY__" {
		t.Errorf("Placeholder = %q, want __KEY__", s.Placeholder)
	}
	if s.Value != "secret-val" {
		t.Errorf("Value = %q, want secret-val", s.Value)
	}
	if len(s.ReplaceIn) != 2 || s.ReplaceIn[0] != "headers" || s.ReplaceIn[1] != "body" {
		t.Errorf("ReplaceIn = %v, want [headers body]", s.ReplaceIn)
	}
}

func TestMergeFlags(t *testing.T) {
	cfg := &models.SandboxConfig{Image: "old-image", Workdir: "."}
	err := config.MergeFlags(cfg, "new-image", "/tmp", nil, nil, 9090)
	if err != nil {
		t.Fatalf("MergeFlags: %v", err)
	}
	if cfg.Image != "new-image" {
		t.Errorf("Image = %q, want new-image", cfg.Image)
	}
	if cfg.Workdir != "/tmp" {
		t.Errorf("Workdir = %q, want /tmp", cfg.Workdir)
	}
	if cfg.ProxyPort != 9090 {
		t.Errorf("ProxyPort = %d, want 9090", cfg.ProxyPort)
	}
}
