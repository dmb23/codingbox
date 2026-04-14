# Research: Default Sandbox Image and Config Defaults

**Date**: 2026-03-27
**Feature**: 004-default-sandbox-image

## R1: Agent Installation Methods

### Claude Code
**Decision**: Install via npm (`npm install -g @anthropic-ai/claude-code`)
**Config dirs**: `~/.claude/` (rw, session state), `~/.claude.json` (rw, global settings)
**Notes**: Requires Node.js 18+. Auth handled via API key env var.

### Mistral Vibe
**Decision**: Install via `curl -LsSf https://mistral.ai/vibe/install.sh | bash`
**Config dirs**: `~/.vibe/` (rw, config + credentials)
**Notes**: Requires Python 3.12+. Config in `~/.vibe/config.toml`.

### OpenCode
**Decision**: Install via `curl -fsSL https://raw.githubusercontent.com/opencode-ai/opencode/refs/heads/main/install | bash`
**Config dirs**: `~/.config/opencode/` (rw, settings), `~/.local/share/opencode/` (rw, credentials + logs)
**Notes**: Go binary. Can also install via npm (`npm i -g opencode-ai`).

## R2: Auto-Mount Paths

**Decision**: Mount the following host paths (same path inside container) when they exist:

| Host Path | Mode | Purpose |
|-----------|------|---------|
| `~/.gitconfig` | ro | Git identity |
| `~/.config/git/` | ro | Git config directory |
| `~/.claude/` | rw | Claude Code session state + config |
| `~/.claude.json` | rw | Claude Code global settings |
| `~/.vibe/` | rw | Mistral Vibe config + credentials |
| `~/.config/opencode/` | rw | OpenCode settings |
| `~/.local/share/opencode/` | rw | OpenCode credentials + data |

**Rationale**: All paths use the same absolute path in the container as on the host. Agent config directories need rw for session persistence. Git config is ro to prevent accidental modification.

## R3: Container User and UID/GID Matching

**Decision**: Use an entrypoint script that creates a user matching the host's UID/GID.

**Approach**:
1. Pass host UID and GID as environment variables at container start
2. Entrypoint script creates a user with matching UID/GID
3. Sets HOME to match the host's HOME path
4. Execs the shell as that user

**Rationale**: Bind mounts use kernel-level UID/GID. Matching ensures the container user can read/write mounted files without permission issues. Setting HOME to match the host ensures `~/` resolves the same way.

## R4: Default Image Name and Registry

**Decision**: Built-in default image name: `codingbox/sandbox:latest`

**Rationale**: Short, memorable, follows Docker Hub conventions. The image can be pushed to Docker Hub or GitHub Container Registry for distribution.

## R5: Container Base Image

**Decision**: Ubuntu 24.04 (noble)

**Rationale**: Latest LTS, broad package availability, widely used as container base. Ubuntu 22.04 works too but 24.04 has newer Python (3.12+ needed for Vibe) without extra PPA setup.

## R6: Development Tools to Include

**Decision**: Include the following in the default image:
- **Editors**: neovim
- **VCS**: git
- **Languages**: Node.js 22 LTS, Python 3.12+, Go (latest stable)
- **Build tools**: build-essential, curl, wget, jq, ripgrep, fd-find
- **Shell**: bash, zsh
