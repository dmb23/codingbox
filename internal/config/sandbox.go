package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// LoadSandboxConfig reads and validates a codingbox.yml configuration file.
func LoadSandboxConfig(path string) (*models.SandboxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg models.SandboxConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := validateSandboxConfig(&cfg); err != nil {
		return nil, err
	}

	// Assign placeholder UUIDs to secrets
	for i := range cfg.Secrets {
		cfg.Secrets[i].ID = uuid.New().String()
	}

	return &cfg, nil
}

func validateSandboxConfig(cfg *models.SandboxConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("config validation: 'name' is required")
	}
	if cfg.Agent == "" {
		return fmt.Errorf("config validation: 'agent' is required")
	}
	if cfg.WorkspaceDir == "" {
		return fmt.Errorf("config validation: 'workspace' is required")
	}
	if !filepath.IsAbs(cfg.WorkspaceDir) {
		return fmt.Errorf("config validation: 'workspace' must be an absolute path")
	}

	for i, m := range cfg.Mounts {
		if m.HostPath == "" {
			return fmt.Errorf("config validation: mount[%d] 'host' is required", i)
		}
		if m.SandboxPath == "" {
			return fmt.Errorf("config validation: mount[%d] 'sandbox' is required", i)
		}
		if m.Mode == "" {
			cfg.Mounts[i].Mode = "ro"
		} else if m.Mode != "rw" && m.Mode != "ro" {
			return fmt.Errorf("config validation: mount[%d] 'mode' must be 'rw' or 'ro'", i)
		}
	}

	for i, s := range cfg.Secrets {
		if s.Name == "" {
			return fmt.Errorf("config validation: secret[%d] 'name' is required", i)
		}
		if s.TargetHost == "" {
			return fmt.Errorf("config validation: secret[%d] 'host' is required", i)
		}
		if s.HeaderName == "" {
			return fmt.Errorf("config validation: secret[%d] 'header' is required", i)
		}
		if s.HeaderTemplate == "" {
			return fmt.Errorf("config validation: secret[%d] 'template' is required", i)
		}
		if s.SecretValue == "" {
			return fmt.Errorf("config validation: secret[%d] 'value' is required", i)
		}
	}

	return nil
}
