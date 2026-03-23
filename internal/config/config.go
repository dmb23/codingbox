package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mischa/codingbox/internal/models"
	"github.com/spf13/viper"
)

// Load reads the configuration file and returns a SandboxConfig with defaults applied.
func Load(configPath string) (*models.SandboxConfig, error) {
	v := viper.New()

	v.SetDefault("workdir", ".")
	v.SetDefault("proxy_port", 0)

	explicit := configPath != ""
	if !explicit {
		configPath = filepath.Join(".", "codingbox.yaml")
	}

	if _, err := os.Stat(configPath); err == nil {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config %q: %w", configPath, err)
		}
	} else if explicit {
		return nil, fmt.Errorf("config file %q not found", configPath)
	}
	// If default config doesn't exist, proceed with defaults + CLI flags only.

	var cfg models.SandboxConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// Validate checks that required fields are present and values are valid.
func Validate(cfg *models.SandboxConfig) error {
	if cfg.Image == "" {
		return fmt.Errorf("image is required (set in config file or use --image flag)")
	}

	// Resolve workdir to absolute path.
	absWorkdir, err := filepath.Abs(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("resolving workdir: %w", err)
	}
	info, err := os.Stat(absWorkdir)
	if err != nil {
		return fmt.Errorf("workdir %q: %w", absWorkdir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workdir %q is not a directory", absWorkdir)
	}
	cfg.Workdir = absWorkdir

	for i, m := range cfg.Mounts {
		if err := validateMount(&cfg.Mounts[i], m); err != nil {
			return err
		}
	}

	for i := range cfg.Secrets {
		if cfg.Secrets[i].Placeholder == "" {
			return fmt.Errorf("secret at index %d: placeholder is required", i)
		}
		if cfg.Secrets[i].Value == "" {
			return fmt.Errorf("secret %q: value is required", cfg.Secrets[i].Placeholder)
		}
	}

	return nil
}

func validateMount(dst *models.MountConfig, m models.MountConfig) error {
	if m.Source == "" {
		return fmt.Errorf("mount target %q: source is required", m.Target)
	}
	if m.Target == "" {
		return fmt.Errorf("mount source %q: target is required", m.Source)
	}
	if !filepath.IsAbs(m.Target) {
		return fmt.Errorf("mount target %q must be an absolute path", m.Target)
	}
	if m.Mode != "" && m.Mode != "ro" && m.Mode != "rw" {
		return fmt.Errorf("mount %q: mode must be 'ro' or 'rw', got %q", m.Source, m.Mode)
	}
	// Resolve source to absolute path and verify it exists.
	absSrc, err := filepath.Abs(m.Source)
	if err != nil {
		return fmt.Errorf("resolving mount source %q: %w", m.Source, err)
	}
	if _, err := os.Stat(absSrc); err != nil {
		return fmt.Errorf("mount source %q: %w", absSrc, err)
	}
	dst.Source = absSrc
	return nil
}

func applyDefaults(cfg *models.SandboxConfig) {
	if cfg.Workdir == "" {
		cfg.Workdir = "."
	}
	for i := range cfg.Mounts {
		if cfg.Mounts[i].Mode == "" {
			cfg.Mounts[i].Mode = "ro"
		}
	}
	for i := range cfg.Secrets {
		if len(cfg.Secrets[i].ReplaceIn) == 0 {
			cfg.Secrets[i].ReplaceIn = []string{
				models.ReplaceHeaders,
				models.ReplaceBody,
				models.ReplaceQuery,
			}
		}
	}
}

// ParseMountFlag parses a --mount flag value in the format "source:target[:ro|rw]".
func ParseMountFlag(val string) (models.MountConfig, error) {
	parts := strings.SplitN(val, ":", 3)
	if len(parts) < 2 {
		return models.MountConfig{}, fmt.Errorf("invalid mount format %q, expected source:target[:ro|rw]", val)
	}
	m := models.MountConfig{
		Source: parts[0],
		Target: parts[1],
		Mode:   "ro",
	}
	if len(parts) == 3 {
		m.Mode = parts[2]
	}
	return m, nil
}

// ParseSecretFlag parses a --secret flag value in the format "placeholder=value[:locations]".
func ParseSecretFlag(val string) (models.SecretMapping, error) {
	eqIdx := strings.Index(val, "=")
	if eqIdx < 1 {
		return models.SecretMapping{}, fmt.Errorf("invalid secret format %q, expected placeholder=value[:headers,body,query]", val)
	}
	placeholder := val[:eqIdx]
	rest := val[eqIdx+1:]

	// Check for optional location suffix after the last ':'
	s := models.SecretMapping{
		Placeholder: placeholder,
		ReplaceIn:   []string{models.ReplaceHeaders, models.ReplaceBody, models.ReplaceQuery},
	}

	colonIdx := strings.LastIndex(rest, ":")
	if colonIdx > 0 {
		possibleLocations := rest[colonIdx+1:]
		// Heuristic: if it looks like a location list, parse it
		locs := strings.Split(possibleLocations, ",")
		allValid := true
		for _, l := range locs {
			l = strings.TrimSpace(l)
			if l != models.ReplaceHeaders && l != models.ReplaceBody && l != models.ReplaceQuery {
				allValid = false
				break
			}
		}
		if allValid && len(locs) > 0 {
			s.Value = rest[:colonIdx]
			s.ReplaceIn = locs
		} else {
			s.Value = rest
		}
	} else {
		s.Value = rest
	}

	return s, nil
}

// MergeFlags applies CLI flag overrides to an existing config.
func MergeFlags(cfg *models.SandboxConfig, image, workdir string, mountFlags, secretFlags []string, proxyPort int) error {
	if image != "" {
		cfg.Image = image
	}
	if workdir != "" {
		cfg.Workdir = workdir
	}
	if proxyPort != 0 {
		cfg.ProxyPort = proxyPort
	}
	for _, mf := range mountFlags {
		m, err := ParseMountFlag(mf)
		if err != nil {
			return err
		}
		cfg.Mounts = append(cfg.Mounts, m)
	}
	for _, sf := range secretFlags {
		s, err := ParseSecretFlag(sf)
		if err != nil {
			return err
		}
		cfg.Secrets = append(cfg.Secrets, s)
	}
	return nil
}
