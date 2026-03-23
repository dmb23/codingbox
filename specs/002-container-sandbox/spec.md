# Feature Specification: Container Sandbox for Agentic Workloads

**Feature Branch**: `002-container-sandbox`
**Created**: 2026-03-23
**Status**: Draft
**Input**: User description: "I want to build an environment to securely run agentic workloads with low interactive oversight. My goal is to have a sandboxed environment in which I can interactively run different coding agents (e.g. Claude, OpenCode, Mistral Vibe). The sandbox should allow write access to the local directory, and the possibility to configure additional directories that are included with read or write access. All outbound traffic of the sandbox should go through a mitm server that logs all calls. In addition the server should be able to transparently inject secrets into the calls: secrets are only known outside of the sandbox, and a placeholder is created inside the sandbox. I would like to be able to define the sandbox as OCI images."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Launch a Coding Agent in a Sandbox (Priority: P1)

A developer wants to run a coding agent (e.g. Claude Code) inside a sandboxed container to work on a project in their current directory. They invoke a CLI command, specifying the sandbox image and the working directory. The sandbox starts, mounts the working directory with write access, and drops the user into an interactive terminal session inside the container where the agent is available.

**Why this priority**: This is the core value proposition — without a working sandbox that mounts the local directory and provides an interactive session, nothing else matters.

**Independent Test**: Can be fully tested by launching a sandbox with a test image, verifying the working directory is mounted read-write, and confirming the user gets an interactive terminal session.

**Acceptance Scenarios**:

1. **Given** a valid OCI image and a local project directory, **When** the user launches the sandbox, **Then** an interactive terminal session starts inside the container with the project directory mounted read-write.
2. **Given** a running sandbox session, **When** the agent creates or modifies files in the mounted directory, **Then** those changes are visible on the host filesystem.
3. **Given** no sandbox image specified, **When** the user launches the sandbox, **Then** a clear error message is shown explaining that an image is required.

---

### User Story 2 - Proxy and Log All Outbound Traffic (Priority: P2)

A developer wants visibility into all network calls their coding agent makes. When the sandbox is running, all outbound HTTP/HTTPS traffic is routed through a MITM proxy that logs every request and response. The developer can review these logs to understand what the agent communicated externally.

**Why this priority**: Network visibility is the primary security feature — it lets the developer trust the agent's behavior without watching every action in real time.

**Independent Test**: Can be tested by launching a sandbox, making an outbound HTTP request from inside, and verifying the request and response appear in the proxy logs.

**Acceptance Scenarios**:

1. **Given** a running sandbox, **When** a process inside makes an outbound HTTPS request, **Then** the request URL, headers, and body are logged by the proxy.
2. **Given** a running sandbox, **When** a process inside makes an outbound HTTPS request, **Then** the response status, headers, and body are logged by the proxy.
3. **Given** a running sandbox, **When** a process attempts to bypass the proxy (e.g. direct IP connection on port 443), **Then** the connection is blocked or still routed through the proxy.

---

### User Story 3 - Transparent Secret Injection (Priority: P3)

A developer wants their coding agent to be able to make authenticated API calls without the agent having access to the actual secrets. The developer defines secret mappings (placeholder → real value) outside the sandbox. Inside the sandbox, the agent uses placeholders in its requests. The MITM proxy transparently replaces placeholders with real secrets in outbound requests and strips secrets from inbound responses.

**Why this priority**: This enables a powerful trust model — agents can function fully without ever seeing credentials — but it depends on the proxy infrastructure from US2.

**Independent Test**: Can be tested by defining a secret mapping, making an API call from inside the sandbox using the placeholder, and verifying the proxy substituted the real secret in the outbound request while the sandbox never saw the real value.

**Acceptance Scenarios**:

1. **Given** a secret mapping `PLACEHOLDER_API_KEY → real-api-key-value` and a running sandbox, **When** a process inside sends a request with `PLACEHOLDER_API_KEY` in a header, **Then** the proxy replaces it with `real-api-key-value` before forwarding to the destination.
2. **Given** a secret mapping and a running sandbox, **When** a response from an external service contains the real secret value, **Then** the proxy replaces it with the placeholder before passing the response into the sandbox.
3. **Given** a running sandbox, **When** a process inside tries to read environment variables or files, **Then** only placeholders are visible, never real secret values.

---

### User Story 4 - Configure Additional Directory Mounts (Priority: P4)

A developer wants to give the sandbox access to additional directories beyond the working directory — for example, a shared library folder (read-only) or an output directory (read-write). They specify these mounts in the sandbox configuration.

**Why this priority**: Extends the core sandbox capability (US1) to support more complex project layouts, but the sandbox is useful without it.

**Independent Test**: Can be tested by launching a sandbox with additional mount configurations and verifying the directories are accessible with the correct permissions inside the container.

**Acceptance Scenarios**:

1. **Given** an additional directory configured as read-only, **When** the sandbox starts, **Then** the directory is accessible inside the container and write attempts are rejected.
2. **Given** an additional directory configured as read-write, **When** the sandbox starts, **Then** the directory is accessible and writable inside the container, and changes are visible on the host.
3. **Given** an additional mount pointing to a non-existent host directory, **When** the sandbox starts, **Then** a clear error message is shown.

---

### User Story 5 - Define Sandbox as OCI Image (Priority: P5)

A developer wants to create custom sandbox environments tailored to specific agents or workflows. They define the environment as an OCI-compliant image (e.g. via a Containerfile/Dockerfile) that includes the agent, its dependencies, and any required tooling. The sandbox tool uses this image to launch containers.

**Why this priority**: OCI images are the standard for portable, reproducible container definitions. This is important for sharing and versioning sandbox environments, but the tool can work with pre-built images initially.

**Independent Test**: Can be tested by building a custom OCI image with a specific tool installed, launching a sandbox from that image, and verifying the tool is available inside.

**Acceptance Scenarios**:

1. **Given** a valid OCI image reference (local or remote registry), **When** the user specifies it for the sandbox, **Then** the sandbox launches using that image.
2. **Given** an OCI image that includes a specific coding agent, **When** the sandbox starts, **Then** the agent is available and runnable inside the container.
3. **Given** an invalid or non-existent image reference, **When** the user attempts to launch the sandbox, **Then** a clear error message explains the image could not be found or pulled.

---

### Edge Cases

- What happens when the container runtime is not installed or not running?
- What happens when the specified port for the MITM proxy is already in use?
- What happens when the sandbox process crashes or is killed — are mounts cleanly unmounted?
- What happens when a process inside the sandbox attempts non-HTTP traffic (e.g. raw TCP, gRPC)? → Blocked; connection refused.
- What happens when a secret placeholder string appears in non-secret context (e.g. in source code or documentation)?
- What happens when the host filesystem runs out of disk space while the sandbox is writing?
- What happens when multiple sandbox instances run concurrently?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST launch an OCI container from a user-specified image and provide an interactive terminal session.
- **FR-002**: System MUST mount the current working directory into the container with read-write access.
- **FR-003**: System MUST support configuring additional host directories as mounts with either read-only or read-write access.
- **FR-004**: System MUST route all outbound HTTP and HTTPS traffic from the container through a MITM proxy.
- **FR-005**: System MUST log all proxied requests and responses including URL, headers, and body to a persistent database.
- **FR-006**: System MUST support defining secret mappings as placeholder-to-real-value pairs outside the sandbox.
- **FR-007**: System MUST transparently replace secret placeholders with real values in outbound requests, in the HTTP message locations configured for each secret (headers, body, query parameters, or any combination).
- **FR-008**: System MUST transparently replace real secret values with placeholders in inbound responses, in the same configured locations as the corresponding outbound replacement.
- **FR-009**: System MUST ensure real secret values are never accessible inside the sandbox (not in environment variables, files, or any other channel).
- **FR-010**: System MUST accept OCI-compliant images (local or from registries) as sandbox definitions.
- **FR-011**: System MUST route all outbound HTTP/HTTPS traffic through the proxy and block all other outbound traffic (deny by default).
- **FR-012**: System MUST clean up all resources (containers, proxy processes, network configuration) when the sandbox session ends.
- **FR-013**: System MUST provide clear error messages when prerequisites are missing (Docker daemon, images, ports).
- **FR-014**: System MUST support a configuration file as the primary method for defining sandbox parameters (image, mounts, secret mappings).
- **FR-015**: System MUST support CLI flags that override individual configuration file values.

### Key Entities

- **Sandbox**: A running container instance with its associated mounts, network configuration, and proxy. Has a lifecycle (created → running → stopped) and a reference to its source image.
- **Sandbox Image**: An OCI-compliant container image that defines the environment (installed tools, agents, dependencies). Referenced by name/tag or registry URL.
- **Sandbox Configuration**: The set of parameters for a sandbox session: image reference, working directory, additional mounts (path + access mode), and secret mappings. Defined primarily via a configuration file, with individual values overridable by CLI flags.
- **MITM Proxy**: A process that intercepts, logs, and optionally modifies HTTP/HTTPS traffic between the sandbox and external services.
- **Secret Mapping**: A placeholder string, a real secret value, and a set of replacement locations (headers, body, query parameters). The placeholder is visible inside the sandbox; the real value is known only to the proxy. Replacement locations are configurable per secret.
- **Traffic Log**: A record of a proxied request-response pair, including timestamp, URL, method, headers, body, and response details. Persisted in a database for querying and audit.

## Clarifications

### Session 2026-03-23

- Q: Which container runtime to target? → A: Docker only.
- Q: How does the user provide sandbox configuration? → A: Config file as primary, with CLI flag overrides.
- Q: How does the user access traffic logs? → A: Stored in a database.
- Q: What happens to non-HTTP/HTTPS traffic? → A: Blocked (deny by default).
- Q: Where in HTTP messages are secret placeholders replaced? → A: Configurable per secret (headers, body, query params, or any combination).

## Assumptions

- The host machine has Docker installed and the Docker daemon is running.
- The user has sufficient filesystem permissions to mount the specified directories.
- Only HTTP/HTTPS outbound traffic is permitted. All other protocols are blocked by default.
- Secret placeholders are unique strings unlikely to appear in normal content (e.g. `__SECRET_GITHUB_TOKEN__`).
- The MITM proxy handles TLS termination by injecting a CA certificate into the container's trust store.
- A single sandbox session runs one container; multi-container orchestration is out of scope.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can go from invoking the sandbox command to an interactive agent session in under 30 seconds (excluding image pull time).
- **SC-002**: 100% of outbound HTTP/HTTPS requests from the sandbox are captured in the traffic log.
- **SC-003**: Secret placeholders are replaced in outbound requests with zero instances of real secrets leaking into the sandbox environment.
- **SC-004**: The sandbox can be launched with any valid OCI image without requiring image modifications by the user.
- **SC-005**: Additional directory mounts respect their configured access mode (read-only directories reject writes 100% of the time).
- **SC-006**: All sandbox resources are cleaned up within 5 seconds of session termination, leaving no orphaned containers or processes.
- **SC-007**: A new user can configure and launch their first sandbox session within 5 minutes using only the documentation.
