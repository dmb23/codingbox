package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mischa/codingbox/internal/models"
	"github.com/spf13/viper"
)

// Load reads configuration using this precedence:
// 1. Explicit --config path
// 2. Local ./codingbox.yaml
// 3. Central per-directory config (~/.codingbox/directories.yaml)
// 4. Empty config (relies on CLI flags)
func Load(configPath string) (*models.SandboxConfig, error) {
	v := viper.New()

	v.SetDefault("workdir", ".")
	v.SetDefault("proxy_port", 0)

	explicit := configPath != ""
	if explicit {
		// Explicit config path — must exist.
		if _, err := os.Stat(configPath); err != nil {
			return nil, fmt.Errorf("config file %q not found", configPath)
		}
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config %q: %w", configPath, err)
		}
	} else {
		// Try local codingbox.yaml first.
		localPath := filepath.Join(".", "codingbox.yaml")
		if _, err := os.Stat(localPath); err == nil {
			v.SetConfigFile(localPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("reading config %q: %w", localPath, err)
			}
		} else {
			// Fall back to central per-directory config.
			centralCfg, err := loadFromCentralStore()
			if err == nil && centralCfg != nil {
				applyDefaults(centralCfg)
				return centralCfg, nil
			}
			// No config found — return empty config (will rely on CLI flags).
		}
	}

	var cfg models.SandboxConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// loadFromCentralStore looks up the current directory in the central config store.
func loadFromCentralStore() (*models.SandboxConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	canonical, err := CanonicalDir(cwd)
	if err != nil {
		return nil, err
	}

	store := NewDirectoryConfigStore(DefaultStorePath())
	if err := store.Load(); err != nil {
		return nil, err
	}

	cfg, _, ok := store.FindNearest(canonical)
	if !ok {
		return nil, nil
	}
	return cfg, nil
}

// ResolveEnvSecrets resolves secrets: reads values from host env,
// generates placeholders. Must be called before Validate().
func ResolveEnvSecrets(cfg *models.SandboxConfig) error {
	seen := make(map[string]bool)
	for i := range cfg.Secrets {
		s := &cfg.Secrets[i]
		if s.Env == "" {
			return fmt.Errorf("secret at index %d: 'env' is required", i)
		}
		if seen[s.Env] {
			return fmt.Errorf("duplicate env secret %q", s.Env)
		}
		seen[s.Env] = true

		// Read value from host env if not explicitly provided.
		if s.Value == "" {
			val, ok := os.LookupEnv(s.Env)
			if !ok {
				return fmt.Errorf("secret env var %q is not set on the host. Set it or provide an explicit value in the config", s.Env)
			}
			s.Value = val
		}

		// Auto-generate placeholder.
		s.Placeholder = GeneratePlaceholder(s.Env)
	}
	return nil
}

// ResolveDefaultImage sets the image to the global default or built-in default
// if no image is configured. Call after Load + MergeFlags, before Validate.
func ResolveDefaultImage(cfg *models.SandboxConfig) {
	if cfg.Image != "" {
		return
	}
	// Try global default from central store.
	store := NewDirectoryConfigStore(DefaultStorePath())
	if err := store.Load(); err == nil && store.Defaults.DefaultImage != "" {
		cfg.Image = store.Defaults.DefaultImage
		return
	}
	// Fall back to built-in default.
	cfg.Image = DefaultSandboxImage
}

// Validate checks that required fields are present and values are valid.
func Validate(cfg *models.SandboxConfig) error {
	if cfg.Image == "" {
		return fmt.Errorf("image is required. Set in config file, use --image flag, or run: codingbox config set --image <image>")
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
		s := &cfg.Secrets[i]
		if s.Env == "" {
			return fmt.Errorf("secret at index %d: 'env' is required", i)
		}
		if s.Placeholder == "" || s.Value == "" {
			return fmt.Errorf("secret env %q: not resolved (call ResolveEnvSecrets first)", s.Env)
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

// ParseEnvSecretFlag parses a --env-secret flag value in the format "ENV_NAME[:locations]".
func ParseEnvSecretFlag(val string) (models.SecretMapping, error) {
	s := models.SecretMapping{
		ReplaceIn: []string{models.ReplaceHeaders, models.ReplaceBody, models.ReplaceQuery},
	}

	colonIdx := strings.Index(val, ":")
	if colonIdx > 0 {
		s.Env = val[:colonIdx]
		locs := strings.Split(val[colonIdx+1:], ",")
		allValid := true
		for _, l := range locs {
			l = strings.TrimSpace(l)
			if l != models.ReplaceHeaders && l != models.ReplaceBody && l != models.ReplaceQuery {
				allValid = false
				break
			}
		}
		if allValid && len(locs) > 0 {
			s.ReplaceIn = locs
		}
	} else {
		s.Env = val
	}

	if s.Env == "" {
		return models.SecretMapping{}, fmt.Errorf("invalid env-secret format %q, expected ENV_NAME[:headers,body,query]", val)
	}

	return s, nil
}

// MergeFlags applies CLI flag overrides to an existing config.
func MergeFlags(cfg *models.SandboxConfig, image, workdir string, mountFlags, envSecretFlags []string, proxyPort int) error {
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
	for _, ef := range envSecretFlags {
		s, err := ParseEnvSecretFlag(ef)
		if err != nil {
			return err
		}
		cfg.Secrets = append(cfg.Secrets, s)
	}
	return nil
}
