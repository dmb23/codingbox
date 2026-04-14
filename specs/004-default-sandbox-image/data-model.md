# Data Model: Default Sandbox Image and Config Defaults

**Date**: 2026-03-27
**Feature**: 004-default-sandbox-image

## New Entities

### AutoMount

A mount automatically added by codingbox based on host path detection.

| Field | Type | Description |
|-------|------|-------------|
| source | string | Absolute path on the host (e.g. `/Users/dev/.gitconfig`) |
| target | string | Same absolute path (mirrors host) |
| mode | enum(ro, rw) | Access mode |
| description | string | Human-readable purpose (for `--help` / docs) |

Auto-mounts are defined as a built-in list, not user-configurable. They are skipped if the source path doesn't exist on the host.

### GlobalDefaults

System-wide defaults stored in the central config store, separate from per-directory entries.

| Field | Type | Description |
|-------|------|-------------|
| default_image | string | Default image when no image is specified anywhere. Default: `codingbox/sandbox:latest` |

**Storage**: Added as a top-level `defaults` key in `~/.codingbox/directories.yaml`:

```yaml
defaults:
  default_image: "codingbox/sandbox:latest"

directories:
  /Users/dev/project-a:
    image: "custom:latest"
```

## Changes to Existing Entities

### SandboxConfig

No structural changes. The `Image` field can now be empty at load time — it gets filled by the default image fallback during resolution.

### Manager (sandbox.go)

`buildEnv()` and mount construction now include auto-mounts. `Start()` passes host UID/GID to the container.

## Auto-Mount Registry (built-in)

| Source Pattern | Mode | Purpose |
|----------------|------|---------|
| `$HOME/.gitconfig` | ro | Git user identity |
| `$HOME/.config/git/` | ro | Git config directory |
| `$HOME/.claude/` | rw | Claude Code config + sessions |
| `$HOME/.claude.json` | rw | Claude Code global settings |
| `$HOME/.vibe/` | rw | Mistral Vibe config |
| `$HOME/.config/opencode/` | rw | OpenCode settings |
| `$HOME/.local/share/opencode/` | rw | OpenCode data + credentials |
