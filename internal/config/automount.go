package config

import (
	"os"
	"path/filepath"

	"github.com/mischa/codingbox/internal/models"
)

// DefaultSandboxImage is the built-in default image used when no image is configured.
const DefaultSandboxImage = "codingbox/sandbox:latest"

// AutoMountEntry defines a host path to auto-mount into the container.
type AutoMountEntry struct {
	RelPath     string // Relative to $HOME (e.g. ".gitconfig")
	Mode        string // "ro" or "rw"
	Description string
}

// AutoMounts is the built-in registry of paths to auto-mount.
var AutoMounts = []AutoMountEntry{
	{RelPath: ".gitconfig", Mode: "ro", Description: "Git user identity"},
	{RelPath: ".config/git", Mode: "ro", Description: "Git config directory"},
	{RelPath: ".claude", Mode: "rw", Description: "Claude Code config and sessions"},
	{RelPath: ".claude.json", Mode: "rw", Description: "Claude Code global settings"},
	{RelPath: ".vibe", Mode: "rw", Description: "Mistral Vibe config"},
	{RelPath: ".config/opencode", Mode: "rw", Description: "OpenCode settings"},
	{RelPath: ".local/share/opencode", Mode: "rw", Description: "OpenCode data and credentials"},
}

// ResolveAutoMounts returns MountConfig entries for host paths that exist.
// Each mount maps source to the same path inside the container (target == source).
func ResolveAutoMounts(home string) []models.MountConfig {
	var mounts []models.MountConfig
	for _, entry := range AutoMounts {
		source := filepath.Join(home, entry.RelPath)
		if _, err := os.Stat(source); err != nil {
			continue // silently skip missing paths
		}
		mounts = append(mounts, models.MountConfig{
			Source: source,
			Target: source, // same path in container
			Mode:   entry.Mode,
		})
	}
	return mounts
}
