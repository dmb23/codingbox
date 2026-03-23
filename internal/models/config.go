package models

// SandboxConfig is the user-provided configuration for a sandbox session.
type SandboxConfig struct {
	Image     string          `yaml:"image" mapstructure:"image"`
	Workdir   string          `yaml:"workdir" mapstructure:"workdir"`
	Mounts    []MountConfig   `yaml:"mounts" mapstructure:"mounts"`
	Secrets   []SecretMapping `yaml:"secrets" mapstructure:"secrets"`
	ProxyPort int             `yaml:"proxy_port" mapstructure:"proxy_port"`
}

// MountConfig describes an additional directory mount.
type MountConfig struct {
	Source string `yaml:"source" mapstructure:"source"`
	Target string `yaml:"target" mapstructure:"target"`
	Mode   string `yaml:"mode" mapstructure:"mode"` // "ro" or "rw", default "ro"
}

// SecretMapping maps a placeholder to a real secret value.
type SecretMapping struct {
	Placeholder string   `yaml:"placeholder" mapstructure:"placeholder"`
	Value       string   `yaml:"value" mapstructure:"value"`
	ReplaceIn   []string `yaml:"replace_in" mapstructure:"replace_in"` // "headers", "body", "query"
}

// ReplacementLocation constants.
const (
	ReplaceHeaders = "headers"
	ReplaceBody    = "body"
	ReplaceQuery   = "query"
)
