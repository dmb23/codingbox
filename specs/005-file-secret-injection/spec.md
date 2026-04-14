# Feature Specification: File-Based Secret Injection

**Feature Branch**: `005-file-secret-injection`
**Created**: 2026-03-28
**Status**: Draft
**Input**: File-based secret injection via split-mount: read secret values from host files, create placeholder files inside the container, and replace placeholders in HTTP traffic via the proxy. Enables OAuth token isolation for agents like Claude Code without exposing real tokens inside the sandbox.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Isolate OAuth Tokens from the Sandbox (Priority: P1)

A developer uses Claude Code inside the sandbox. Claude Code stores its OAuth token in a file inside `~/.claude/`. The developer wants the token to never be visible inside the container, while still allowing Claude Code to function normally. The system reads the real token from the host file, places a placeholder in the corresponding file inside the container, and the proxy transparently replaces the placeholder with the real token in outbound HTTP requests.

**Why this priority**: This is the core use case — OAuth tokens are the most sensitive credential type for agentic workflows. Without this, the developer must choose between exposing real tokens inside the sandbox or not using OAuth-authenticated agents at all.

**Independent Test**: Configure a file secret for a test file, launch the sandbox, verify the file inside the container contains a placeholder (not the real value), make an HTTP request that includes the file's content as a header, and confirm the proxy replaced the placeholder with the real value.

**Acceptance Scenarios**:

1. **Given** a file secret configured as `file: ~/.claude/credentials.json`, `json_key: oauth_token`, **When** the sandbox starts, **Then** the directory `~/.claude/` is mounted read-write as usual, but the specific file `credentials.json` is overlaid with a read-only file containing the placeholder instead of the real token.
2. **Given** a running sandbox with a file secret, **When** an agent reads the token file and sends the value in an HTTP Authorization header, **Then** the proxy replaces the placeholder with the real token value before forwarding.
3. **Given** a running sandbox with a file secret, **When** a response contains the real token value, **Then** the proxy replaces it with the placeholder before the response reaches the container.
4. **Given** the overlay file is read-only, **When** the agent or any process tries to write to the token file, **Then** the write is rejected (read-only), preserving the placeholder. The host file is never modified.
5. **Given** a file secret for a file that does not exist on the host, **When** the sandbox starts, **Then** a clear error indicates the file was not found.

---

### User Story 2 - File Secret with Plain Text Files (Priority: P2)

A developer has a plain text token file (not JSON) that contains a single secret value, such as a bearer token or API key stored in a file. The system reads the entire file content as the secret value.

**Why this priority**: Not all agents store tokens as JSON. Some use plain text files (e.g., a `.token` file with just the raw token string). Supporting plain text files makes the feature broadly applicable.

**Independent Test**: Create a plain text file with a token, configure it as a file secret without `json_key`, verify the entire file content is treated as the secret and replaced in the container with a placeholder.

**Acceptance Scenarios**:

1. **Given** a file secret configured as `file: ~/.myagent/.token` (no `json_key`), **When** the sandbox starts, **Then** the entire file content is used as the secret value, and the file inside the container contains the placeholder.
2. **Given** the plain text file has trailing whitespace or newlines, **When** the secret is resolved, **Then** the value is trimmed before use.

---

### User Story 3 - File Secret via CLI Flag (Priority: P3)

A developer wants to specify file secrets via a CLI flag without editing a config file, for quick one-off use.

**Why this priority**: Completes the UX parity with env secrets, which already support a `--env-secret` CLI flag.

**Independent Test**: Run `codingbox run --file-secret ~/.claude/credentials.json:oauth_token`, verify the file secret is applied.

**Acceptance Scenarios**:

1. **Given** the developer runs `codingbox run --file-secret "~/.claude/credentials.json:oauth_token:headers"`, **When** the sandbox starts, **Then** the file secret is configured with the specified file, JSON key, and replacement locations.
2. **Given** the developer runs `codingbox run --file-secret "~/.myagent/.token"` (no key, no locations), **When** the sandbox starts, **Then** the entire file content is used as the secret with default replacement in all locations.

---

### Edge Cases

- What happens when the JSON key does not exist in the file? Error with clear message.
- What happens when the file is binary (not text)? Undefined behavior — document that only text files are supported.
- What happens when the token value is very long (e.g., JWT with 1000+ chars)? Should still work — placeholder replacement is string-based.
- What happens when two file secrets reference the same file but different JSON keys? Both should work independently.
- What happens when the file is inside a directory that isn't auto-mounted? The user must ensure the parent directory is mounted (via auto-mount or explicit mount).
- What happens when the auto-mount for `~/.claude/` is disabled via `--no-auto-mounts` but a file secret references `~/.claude/credentials.json`? Error: the parent directory must be mounted for the file overlay to work.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a secret type `file` that reads a secret value from a file on the host filesystem.
- **FR-002**: When `json_key` is specified, the system MUST parse the file as JSON and extract the value at the given key.
- **FR-003**: When `json_key` is not specified, the system MUST use the entire file content (trimmed) as the secret value.
- **FR-004**: The system MUST generate a unique placeholder for each file secret (same deterministic format as env secrets).
- **FR-005**: The system MUST create a temporary file containing the placeholder value and mount it read-only into the container at the same path as the original host file, overlaying the auto-mounted or explicitly-mounted parent directory.
- **FR-006**: The host file MUST never be modified. The overlay mount ensures the container sees the placeholder while the host retains the real value.
- **FR-007**: The proxy MUST replace the placeholder with the real value in outbound requests at the configured locations (headers, body, query).
- **FR-008**: The proxy MUST replace the real value with the placeholder in inbound responses at the configured locations.
- **FR-009**: The system MUST verify that the parent directory of the file secret is mounted (via auto-mount or explicit mount) before applying the file overlay. If not mounted, the system MUST report a clear error.
- **FR-010**: The system MUST support a `--file-secret` CLI flag for specifying file secrets on the command line.
- **FR-011**: The system MUST clean up temporary placeholder files when the sandbox session ends.
- **FR-012**: File secrets MUST be configurable in `codingbox.yaml` and `~/.codingbox/directories.yaml` alongside env secrets.

### Key Entities

- **File Secret**: A secret whose value is read from a file on the host. Defined by a file path, an optional JSON key, and replacement locations. The system generates a placeholder, creates a temp file with the placeholder, and overlays it into the container at the original path.
- **Placeholder File**: A temporary read-only file containing the placeholder value, mounted into the container on top of the real file path. Cleaned up when the sandbox exits.

## Assumptions

- Only text files are supported for file secrets (not binary).
- JSON extraction supports top-level keys only (not nested paths like `a.b.c`). Nested key support can be added later.
- The placeholder file is created in a temporary directory managed by codingbox (e.g., `/tmp/codingbox-<session>/`).
- File secrets produce the same type of SecretMapping (with Placeholder and Value) as env secrets — the proxy replacement logic is shared.
- The `replace_in` default for file secrets is `[headers, body, query]` (same as env secrets).
- Claude Code's auth token file path and format will need to be researched during planning.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can use Claude Code with OAuth authentication inside the sandbox without the real OAuth token ever being readable inside the container.
- **SC-002**: The real token file on the host is never modified by the sandbox — verified by comparing file content before and after a sandbox session.
- **SC-003**: The proxy correctly replaces the placeholder with the real token in 100% of outbound requests that contain it.
- **SC-004**: File secrets work alongside env secrets with zero conflicts — both types can be configured simultaneously.
- **SC-005**: Temporary placeholder files are cleaned up within 5 seconds of sandbox termination.
- **SC-006**: A developer can configure a file secret for any text file in a single line of YAML config or a single CLI flag.
