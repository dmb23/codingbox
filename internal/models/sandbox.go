package models

import "time"

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	StateCreating SandboxState = "creating"
	StateRunning  SandboxState = "running"
	StateStopping SandboxState = "stopping"
	StateStopped  SandboxState = "stopped"
)

// Sandbox holds the runtime state of a running sandbox session.
type Sandbox struct {
	ID          string       `json:"id"`
	ContainerID string       `json:"container_id"`
	NetworkID   string       `json:"network_id"`
	ProxyAddr   string       `json:"proxy_addr"`
	Config      SandboxConfig `json:"config"`
	State       SandboxState `json:"state"`
	CreatedAt   time.Time    `json:"created_at"`
}
