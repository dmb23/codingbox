package models

// SandboxConfig is the declarative specification for a sandbox environment.
type SandboxConfig struct {
	Name         string          `yaml:"name"`
	Agent        string          `yaml:"agent"`
	WorkspaceDir string          `yaml:"workspace"`
	Mounts       []Mount         `yaml:"mounts,omitempty"`
	Secrets      []SecretMapping `yaml:"secrets,omitempty"`
	BaseImage    string          `yaml:"base_image,omitempty"`
	Tools        []string        `yaml:"tools,omitempty"`
	Proxy        ProxyConfig     `yaml:"proxy,omitempty"`
}

// Mount defines a host-to-sandbox directory mapping.
type Mount struct {
	HostPath    string `yaml:"host" json:"host_path"`
	SandboxPath string `yaml:"sandbox" json:"sandbox_path"`
	Mode        string `yaml:"mode" json:"mode"`
}

// ProxyConfig holds proxy-specific configuration.
type ProxyConfig struct {
	Port int `yaml:"port,omitempty"`
}

// GlobalConfig holds global codingbox configuration.
type GlobalConfig struct {
	DBPath           string `yaml:"db_path"`
	LogRetentionDays int    `yaml:"log_retention_days"`
	CACertPath       string `yaml:"ca_cert_path"`
	CAKeyPath        string `yaml:"ca_key_path"`
}
