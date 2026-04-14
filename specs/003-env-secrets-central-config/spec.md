# Feature Specification: Env-Based Secrets and Central Configuration

**Feature Branch**: `003-env-secrets-central-config`
**Created**: 2026-03-26
**Status**: Draft
**Input**: User description: "I want to change two things: 1. The secret configuration will be more helpful if I can specify an environment variable (e.g. ANTHROPIC_API_KEY) that then is set inside the sandbox with a placeholder value, and the placeholder value is replaced in the requests. 2. I would like to be able to run the command codingbox run without any additional args in different directories. Without any other specification it should use default configuration specific for this directory. This configuration should be stored in a central location (~/.codingbox)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Environment Variable-Based Secrets (Priority: P1)

A developer wants their coding agent to use API keys via environment variables, just like it would outside the sandbox. The developer specifies an environment variable name (e.g. `ANTHROPIC_API_KEY`). The system reads the real value from the host's environment automatically — no secrets in config files. Inside the sandbox, the environment variable is set to an auto-generated placeholder value. When the agent reads the env var and includes it in an HTTP request, the proxy transparently replaces the placeholder with the real secret. An optional `value` field can override the host env var if needed.

**Why this priority**: This is the higher-value change. The current placeholder-based approach requires the agent to know about the placeholder convention. With env var-based secrets, the agent uses standard environment variables naturally, making codingbox compatible with any agent without special configuration.

**Independent Test**: Can be tested by configuring an env var secret, launching a sandbox, verifying the env var is set to a placeholder inside, making an HTTP request using that env var value, and confirming the proxy replaced it with the real value.

**Acceptance Scenarios**:

1. **Given** a secret configured as `env: ANTHROPIC_API_KEY` (no `value` field) and the host has `ANTHROPIC_API_KEY=sk-ant-real-key` in its environment, **When** the sandbox starts, **Then** the environment variable `ANTHROPIC_API_KEY` is set inside the container to an auto-generated placeholder value (not the real key), and the real value is read from the host env.
2. **Given** a secret configured as `env: ANTHROPIC_API_KEY`, `value: override-key` (explicit value), **When** the sandbox starts, **Then** `override-key` is used as the real secret instead of reading from the host environment.
3. **Given** a secret configured as `env: ANTHROPIC_API_KEY` with no `value` field and the host does NOT have `ANTHROPIC_API_KEY` set, **When** the sandbox starts, **Then** a clear error message indicates the host env var is not set.
4. **Given** a running sandbox with an env secret, **When** a process reads `$ANTHROPIC_API_KEY` and sends it in an HTTP Authorization header, **Then** the proxy replaces the placeholder with the real value before forwarding the request.
5. **Given** a running sandbox with an env secret, **When** a response contains the real secret value, **Then** the proxy replaces it with the placeholder before the response reaches the container.
6. **Given** a secret configured with both `env` and `replace_in` fields, **When** the proxy processes requests, **Then** replacement only occurs in the specified locations (headers, body, query, or any combination).
7. **Given** the existing `placeholder`/`value` secret format (without `env`), **When** the sandbox starts, **Then** it continues to work as before — backwards-compatible.

---

### User Story 2 - Central Per-Directory Configuration (Priority: P2)

A developer works on multiple projects across different directories and wants to run `codingbox run` in any project directory without specifying flags or having a config file in each project. A central configuration store at `~/.codingbox/` holds per-directory defaults (image, secrets, mounts). When the developer runs `codingbox run` in a directory, codingbox looks up the configuration for that directory path.

**Why this priority**: This improves daily ergonomics significantly. Currently, the developer must either place a `codingbox.yaml` in every project or pass flags every time. Central configuration enables a zero-arg workflow across all projects.

**Independent Test**: Can be tested by registering a configuration for a directory path, changing to that directory, running `codingbox run` with no arguments, and confirming the correct image and settings are applied.

**Acceptance Scenarios**:

1. **Given** a central configuration entry for `/Users/dev/project-a` with image `my-agent:latest`, **When** the developer runs `codingbox run` from `/Users/dev/project-a`, **Then** the sandbox starts with `my-agent:latest`.
2. **Given** no local `codingbox.yaml` and no central config for the current directory, **When** the developer runs `codingbox run`, **Then** a clear error message explains that no configuration was found and suggests using `codingbox init` or registering a central config.
3. **Given** both a local `codingbox.yaml` and a central config for the same directory, **When** the developer runs `codingbox run`, **Then** the local `codingbox.yaml` takes precedence over the central config.
4. **Given** a central config for `/Users/dev/project-a`, **When** the developer runs `codingbox run` from `/Users/dev/project-a/src/`, **Then** the config for the nearest matching parent directory (`/Users/dev/project-a`) is used.
5. **Given** a central config entry, **When** the developer passes CLI flags (e.g. `--image`), **Then** the CLI flags override the central config values.

---

### User Story 3 - Register and Manage Central Configurations (Priority: P3)

A developer wants to register, list, update, and remove central configurations for directories without manually editing files. A CLI subcommand provides these operations.

**Why this priority**: Without a management command, users would have to manually edit the central config file, which is error-prone. This completes the ergonomics of US2.

**Independent Test**: Can be tested by registering a config, listing it, updating a field, and removing it, verifying each operation via `codingbox config list`.

**Acceptance Scenarios**:

1. **Given** no existing config for a directory, **When** the developer runs `codingbox config set --image ubuntu:22.04` from that directory, **Then** a central config entry is created for the current directory with the specified image.
2. **Given** an existing config entry, **When** the developer runs `codingbox config list`, **Then** all registered directory configurations are displayed with their key settings.
3. **Given** an existing config entry for a directory, **When** the developer runs `codingbox config remove` from that directory, **Then** the entry is removed and confirmed.
4. **Given** an existing config entry, **When** the developer runs `codingbox config set --image new-image:latest` from the same directory, **Then** the image field is updated and other fields are preserved.

---

### Edge Cases

- What happens when the real secret value appears in the auto-generated placeholder? (Should not happen if placeholders are sufficiently random/unique.)
- What happens when multiple secrets reference the same environment variable name?
- What happens when the central config directory path contains symlinks — does it resolve to the canonical path?
- What happens when a secret env var name conflicts with a system-critical env var (e.g. `PATH`, `HOME`)?
- What happens when the user moves a project directory after registering a central config?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a secret format where the user specifies an environment variable name (`env`). The real value is read from the host's environment by default; an optional `value` field overrides it. The system auto-generates a unique placeholder. If no `value` is provided and the host env var is not set, the system MUST report a clear error.
- **FR-002**: System MUST set the specified environment variable inside the container to the auto-generated placeholder value.
- **FR-003**: System MUST replace the auto-generated placeholder with the real value in outbound requests at the configured locations.
- **FR-004**: System MUST replace the real value with the auto-generated placeholder in inbound responses at the configured locations.
- **FR-005**: System MUST continue to support the existing `placeholder`/`value` secret format for backwards compatibility.
- **FR-006**: System MUST generate placeholders that are unique, deterministic per secret name, and unlikely to appear in normal content.
- **FR-007**: System MUST store per-directory configuration entries in a central location under the user's home directory (`~/.codingbox/`).
- **FR-008**: System MUST look up configuration for the current working directory when `codingbox run` is invoked without a `--config` flag and no local `codingbox.yaml` exists.
- **FR-009**: System MUST walk up the directory tree from the current directory to find the nearest matching central config entry.
- **FR-010**: Configuration precedence MUST be: CLI flags > local `codingbox.yaml` > central per-directory config > error.
- **FR-011**: System MUST provide a `codingbox config set` command to register or update a central config entry for a directory.
- **FR-012**: System MUST provide a `codingbox config list` command to display all registered directory configurations.
- **FR-013**: System MUST provide a `codingbox config remove` command to delete a central config entry for a directory.
- **FR-014**: System MUST resolve directory paths to their canonical (absolute, symlink-resolved) form before storing or looking up central config entries.

### Key Entities

- **Env Secret**: A secret defined by an environment variable name and optional replacement locations. The real value is read from the host's environment by default (or from an explicit `value` override). The placeholder is auto-generated. The env var is set inside the sandbox to the placeholder.
- **Central Config Store**: A persistent store of per-directory sandbox configurations, located at `~/.codingbox/`. Keyed by canonical directory path.
- **Directory Config Entry**: A sandbox configuration (image, mounts, secrets, proxy_port) associated with a specific directory path in the central store.

## Clarifications

### Session 2026-03-26

- Q: Where does the real secret value come from for env-based secrets? → A: Read from host environment by default; optional `value` field overrides.

## Assumptions

- Auto-generated placeholders use the format `__CODINGBOX_<ENV_NAME>_<SHORT_HASH>__` to be unique and recognizable.
- The central config store uses a single YAML file (`~/.codingbox/directories.yaml`) for simplicity and human readability.
- Symlinks in directory paths are resolved before lookup to avoid duplicate entries.
- The `codingbox config set` command operates on the current working directory by default, with an optional `--dir` flag to specify a different directory.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can configure a secret using only an environment variable name and value, with no manual placeholder creation.
- **SC-002**: An agent running inside the sandbox can use `$ENV_VAR_NAME` in HTTP requests and the proxy correctly replaces the placeholder, with zero instances of the real secret visible inside the container.
- **SC-003**: A developer can run `codingbox run` with no arguments in any registered directory and the correct sandbox configuration is applied.
- **SC-004**: The existing `placeholder`/`value` secret format continues to work identically — zero regressions.
- **SC-005**: Configuration lookup from current directory to central store completes in under 100 milliseconds.
- **SC-006**: A developer can register a configuration for a new directory in a single command under 10 seconds.
