package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mischa/codingbox/internal/models"
	"go.yaml.in/yaml/v3"
)

// GlobalDefaults holds system-wide default settings.
type GlobalDefaults struct {
	DefaultImage string `yaml:"default_image,omitempty"`
}

// DirectoryConfigStore manages per-directory configurations in a central YAML file.
type DirectoryConfigStore struct {
	path        string
	Defaults    GlobalDefaults                  `yaml:"defaults"`
	Directories map[string]models.SandboxConfig `yaml:"directories"`
}

// NewDirectoryConfigStore creates a store backed by the given YAML file path.
func NewDirectoryConfigStore(path string) *DirectoryConfigStore {
	return &DirectoryConfigStore{
		path:        path,
		Directories: make(map[string]models.SandboxConfig),
	}
}

// DefaultStorePath returns the default central config file path.
func DefaultStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codingbox", "directories.yaml")
}

// Load reads the store from disk. Returns an empty store if the file doesn't exist.
func (s *DirectoryConfigStore) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.Directories = make(map[string]models.SandboxConfig)
			return nil
		}
		return fmt.Errorf("reading central config: %w", err)
	}

	if err := yaml.Unmarshal(data, s); err != nil {
		return fmt.Errorf("parsing central config: %w", err)
	}
	if s.Directories == nil {
		s.Directories = make(map[string]models.SandboxConfig)
	}
	return nil
}

// Save writes the store to disk, creating parent directories as needed.
func (s *DirectoryConfigStore) Save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshalling central config: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("writing central config: %w", err)
	}
	return nil
}

// Get returns the config for an exact directory match.
func (s *DirectoryConfigStore) Get(dir string) (*models.SandboxConfig, bool) {
	cfg, ok := s.Directories[dir]
	if !ok {
		return nil, false
	}
	return &cfg, true
}

// FindNearest walks up from dir to root looking for a matching entry.
// Returns the config, the matched directory path, and whether a match was found.
func (s *DirectoryConfigStore) FindNearest(dir string) (*models.SandboxConfig, string, bool) {
	current := dir
	for {
		if cfg, ok := s.Directories[current]; ok {
			return &cfg, current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break // reached root
		}
		current = parent
	}
	return nil, "", false
}

// Set creates or updates a config entry for the given directory.
func (s *DirectoryConfigStore) Set(dir string, cfg models.SandboxConfig) {
	s.Directories[dir] = cfg
}

// Remove deletes the config entry for the given directory. Returns false if not found.
func (s *DirectoryConfigStore) Remove(dir string) bool {
	if _, ok := s.Directories[dir]; !ok {
		return false
	}
	delete(s.Directories, dir)
	return true
}

// List returns all directory configurations.
func (s *DirectoryConfigStore) List() map[string]models.SandboxConfig {
	return s.Directories
}

// CanonicalDir resolves a directory path to its absolute, symlink-resolved form.
func CanonicalDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If the path doesn't exist yet, just use absolute path.
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}
	return resolved, nil
}
