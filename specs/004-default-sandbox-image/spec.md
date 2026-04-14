# Feature Specification: Default Sandbox Image and Config Defaults

**Feature Branch**: `004-default-sandbox-image`
**Created**: 2026-03-27
**Status**: Draft
**Input**: User description: "My main use case is a sandbox for agentic development. The sandbox should allow me to do manual development (relevant packages, nvim as editor) and use the coding agents claude, mistral vibe, and OpenCode. I still want to have write access to the current project directory, additionally it will be necessary to have access to the config directories for these agents. My git config should also be forwarded to the container. Please create a corresponding default container and adjust the config handling to set according defaults."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run a Coding Agent in the Default Sandbox (Priority: P1)

A developer wants to run `codingbox run` and immediately have a working environment where they can launch Claude Code, Mistral Vibe, or OpenCode. The default container image has all three agents pre-installed along with standard development tools. The developer's git configuration and agent config directories are automatically mounted so agents work with their existing credentials and settings.

**Why this priority**: Without a default image that "just works" with the target agents, the developer must build their own image before getting any value from codingbox. This is the core deliverable.

**Independent Test**: Run `codingbox run` with the default image, verify Claude Code, Mistral Vibe, and OpenCode are all callable from the terminal, and that `git config user.name` returns the host's git identity.

**Acceptance Scenarios**:

1. **Given** the default sandbox image is available, **When** the developer runs `codingbox run`, **Then** an interactive terminal session starts with Claude Code (`claude`), Mistral Vibe (`mistral-vibe`), and OpenCode (`opencode`) available on the PATH.
2. **Given** the developer has `~/.gitconfig` on the host, **When** the sandbox starts, **Then** `git config user.name` inside the container returns the same value as on the host.
3. **Given** the developer has `~/.config/claude/` on the host, **When** the sandbox starts, **Then** the Claude Code config directory is available inside the container at the expected location.
4. **Given** the default image, **When** the developer opens `nvim` inside the sandbox, **Then** neovim launches and is usable for editing files.
5. **Given** no `--image` flag or config file, **When** the developer runs `codingbox run`, **Then** the system uses the default sandbox image automatically.

---

### User Story 2 - Auto-Mount Agent Config and Git Directories (Priority: P2)

A developer wants their agent configuration directories and git config to be automatically available inside the sandbox without manually specifying mount flags. When the sandbox starts, codingbox detects common config paths on the host and mounts them into the container at the locations where each agent expects them.

**Why this priority**: Without auto-mounting, the developer must manually add `--mount` flags for every config directory every time, which defeats the zero-arg workflow. This makes the out-of-box experience seamless.

**Independent Test**: Run `codingbox run` without mount flags, verify that `~/.gitconfig`, `~/.config/claude/`, and other agent config directories from the host are accessible inside the container.

**Acceptance Scenarios**:

1. **Given** the host has `~/.gitconfig`, **When** the sandbox starts with default settings, **Then** the file is mounted read-only inside the container at the same absolute path as on the host.
2. **Given** the host has `~/.claude/`, **When** the sandbox starts with default settings, **Then** the directory is mounted read-write inside the container at the same absolute path as on the host.
3. **Given** the host does NOT have `~/.claude/`, **When** the sandbox starts, **Then** no mount is created for that path and no error occurs.
4. **Given** the developer explicitly passes `--mount` flags that conflict with auto-mounts, **When** the sandbox starts, **Then** the explicit mounts take precedence.
5. **Given** the developer wants to disable auto-mounts, **When** they pass a `--no-auto-mounts` flag, **Then** no automatic mounts are added.

---

### User Story 3 - Default Image Configuration (Priority: P3)

A developer wants to configure which image is used as the default across all projects, without specifying it in every local config or central config entry. If no image is specified anywhere, the system falls back to the built-in default sandbox image.

**Why this priority**: Completes the zero-configuration experience. Without this, the developer must always specify an image somewhere (config file, central store, or CLI flag).

**Independent Test**: Run `codingbox run` in a directory with no config, no central config entry, and no flags — verify it uses the default image.

**Acceptance Scenarios**:

1. **Given** no image specified in local config, central config, or CLI flags, **When** the developer runs `codingbox run`, **Then** the system uses the default sandbox image.
2. **Given** a custom default image configured globally (e.g. via `codingbox config set-default --image my-custom:latest`), **When** the developer runs `codingbox run` without an image, **Then** the custom default image is used.
3. **Given** a local config or central config with an image specified, **When** the developer runs `codingbox run`, **Then** the configured image takes precedence over the default.

---

### Edge Cases

- What happens when the default image is not available locally and needs to be pulled?
- What happens when host config directories have restrictive permissions that prevent mounting?
- What happens when the host user's UID/GID differs from the container user, causing permission issues on mounted config files?
- What happens when a `.gitconfig` includes paths to host-specific files (e.g. credential helpers)?
- What happens when auto-mounted config directories are very large (e.g. large Claude cache)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST provide a container image definition that includes Claude Code, Mistral Vibe, OpenCode, neovim, and standard development tools (git, curl, build-essential, Node.js, Python, Go).
- **FR-002**: When no image is specified in any config source or CLI flag, the system MUST use a built-in default image name.
- **FR-003**: The system MUST automatically detect and mount the host's git configuration (`~/.gitconfig` and `~/.config/git/`) read-only into the container at the same absolute path as on the host.
- **FR-004**: The system MUST automatically detect and mount agent config directories from the host into the container at the same absolute path as on the host. Specifically: Claude Code (`~/.claude/`), and any other standard agent config locations.
- **FR-005**: Auto-mounts MUST only occur when the source path exists on the host. Missing paths MUST be silently skipped.
- **FR-006**: The system MUST support a `--no-auto-mounts` flag to disable all automatic mounts.
- **FR-007**: Explicit `--mount` flags and config file mounts MUST take precedence over auto-mounts for the same target path.
- **FR-008**: The system MUST allow configuring a custom default image globally via a setting in the central config store.
- **FR-009**: Configuration precedence for image selection MUST be: CLI flags > local config > central per-directory config > global default image > built-in default image.
- **FR-010**: The container image definition MUST be versioned and publishable to a container registry for easy distribution.

### Key Entities

- **Default Sandbox Image**: A container image bundling development tools and coding agents, used when no other image is specified. Has a built-in name that can be overridden globally.
- **Auto-Mount**: A mount that is automatically added based on detection of host paths. Has a source (host path), target (container path), and access mode. Skipped if source does not exist.
- **Global Default Config**: A section of the central config store that holds system-wide defaults (default image), separate from per-directory entries.

## Assumptions

- Claude Code is installed via `npm install -g @anthropic-ai/claude-code`.
- OpenCode is installed via its published binary or package manager.
- Mistral Vibe is installed via its published installer.
- The container image is based on Ubuntu 22.04 for broad compatibility.
- Agent config directories follow standard XDG/home directory conventions.
- The host user's home directory is used to detect config paths (via `$HOME`).
- Auto-mount for git config includes `~/.gitconfig` (read-only) and `~/.config/git/` (read-only).
- Auto-mount for Claude Code includes `~/.claude/` (read-write, needed for session state).
- All auto-mounts use the same absolute path inside the container as on the host, to make debugging and path references consistent.
- The default image name follows the convention `codingbox/sandbox:latest` (or similar).
- The container user's home directory MUST match the host user's home directory path to ensure `~/` resolves consistently for auto-mounts.
- The container runs as a non-root user to match typical host user permissions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can go from a fresh install to a working sandbox with all three coding agents available by running a single command (`codingbox run`) with zero configuration.
- **SC-002**: `git config user.name` inside the sandbox returns the host user's identity on the first run.
- **SC-003**: All three coding agents (Claude Code, Mistral Vibe, OpenCode) are callable from the sandbox terminal without additional installation steps.
- **SC-004**: Config directories that exist on the host are automatically available inside the sandbox, with no manual mount configuration needed.
- **SC-005**: A developer who doesn't want auto-mounts can disable them with a single flag.
- **SC-006**: The default sandbox image is under 2 GB in size (compressed) to keep initial pull time reasonable.
