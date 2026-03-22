package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codingbox/codingbox/internal/models"
	"gopkg.in/yaml.v3"
)

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults.
func DefaultGlobalConfig() *models.GlobalConfig {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".local", "share", "codingbox")

	return &models.GlobalConfig{
		DBPath:           filepath.Join(dataDir, "codingbox.db"),
		LogRetentionDays: 30,
		CACertPath:       filepath.Join(dataDir, "ca.pem"),
		CAKeyPath:        filepath.Join(dataDir, "ca-key.pem"),
	}
}

// GlobalConfigPath returns the default path for the global config file.
func GlobalConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "codingbox", "config.yml")
}

// LoadGlobalConfig loads the global config from disk, falling back to defaults.
func LoadGlobalConfig() (*models.GlobalConfig, error) {
	cfg := DefaultGlobalConfig()

	path := GlobalConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	cfg.DBPath = expandHome(cfg.DBPath)
	cfg.CACertPath = expandHome(cfg.CACertPath)
	cfg.CAKeyPath = expandHome(cfg.CAKeyPath)

	return cfg, nil
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
