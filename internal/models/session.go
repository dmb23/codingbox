package models

import "time"

// SandboxSession represents a single execution lifecycle of a sandbox environment.
type SandboxSession struct {
	ID             string     `json:"id"`
	VMID           string     `json:"vm_id"`
	VMSocketPath   string     `json:"vm_socket_path"`
	AgentName      string     `json:"agent_name"`
	Status         string     `json:"status"`
	ConfigSnapshot string     `json:"config_snapshot"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	StoppedAt      *time.Time `json:"stopped_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

const (
	StatusCreated = "created"
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusFailed  = "failed"
)

// ValidTransitions defines allowed state transitions.
var ValidTransitions = map[string][]string{
	StatusCreated: {StatusRunning, StatusFailed},
	StatusRunning: {StatusStopped, StatusFailed},
}

// CanTransitionTo checks if a status transition is valid.
func (s *SandboxSession) CanTransitionTo(newStatus string) bool {
	allowed, ok := ValidTransitions[s.Status]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == newStatus {
			return true
		}
	}
	return false
}
