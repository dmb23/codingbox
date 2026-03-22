# Feature Specification: Secure Agent Sandbox

**Feature Branch**: `001-secure-agent-sandbox`
**Created**: 2026-03-21
**Status**: Draft
**Input**: User description: "Build a secure sandbox for running coding agents isolated from my machine. I want to safely separate the agents from my filesystem, inject secrets so that secrets are not present inside of the sandbox without limiting the agent and have full observability over all requests that were made"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Launch an Isolated Sandbox (Priority: P1)

As a developer, I want to launch a sandbox environment where a coding agent runs completely isolated from my host machine, so that the agent cannot access or modify files outside of explicitly shared directories.

**Why this priority**: Without isolation, the sandbox has no value. This is the foundational capability that everything else builds on.

**Independent Test**: Can be fully tested by launching a sandbox, attempting to access host paths not explicitly mounted, and verifying access is denied. Delivers the core value of safe agent execution.

**Acceptance Scenarios**:

1. **Given** a sandbox configuration specifying a project directory as read-write, **When** the sandbox is launched, **Then** the agent can read and write files only within that mounted directory.
2. **Given** a running sandbox, **When** the agent attempts to access a host path that was not explicitly mounted, **Then** the access is denied and the agent receives a clear error.
3. **Given** a sandbox configuration specifying a config directory as read-only, **When** the agent attempts to write to that directory, **Then** the write is denied and existing files remain readable.
4. **Given** a previously launched sandbox with installed packages, **When** the sandbox is restarted, **Then** the installed packages and environment changes persist.

---

### User Story 2 - Transparent Secret Injection (Priority: P2)

As a developer, I want to configure secrets (API keys, tokens) that are automatically injected into outbound HTTP requests from the sandbox, so that the agent can use authenticated services without ever seeing or accessing the actual secret values.

**Why this priority**: Agents need to call authenticated APIs (LLM providers, Git hosts, package registries) to be useful. Without secret injection, agents either cannot function or secrets must be exposed inside the sandbox.

**Independent Test**: Can be tested by configuring a secret mapping, having the agent make an HTTP request to a target host, and verifying the secret is injected in the outbound request while the agent only sees a placeholder.

**Acceptance Scenarios**:

1. **Given** a secret configured for a specific host (e.g., `api.openai.com`), **When** the agent makes an HTTP request to that host, **Then** the proxy intercepts the request and injects the real secret into the appropriate header before forwarding.
2. **Given** a configured secret, **When** the agent inspects its own environment variables, **Then** the agent sees only a placeholder UUID, never the actual secret value.
3. **Given** a secret configured for host A, **When** the agent makes a request to host B (not configured), **Then** no secret injection occurs and the request is forwarded as-is.
4. **Given** a misconfigured or missing secret mapping, **When** the agent makes a request to the target host, **Then** the system logs a warning and the request proceeds without injection (no silent failure).

---

### User Story 3 - Full Request Observability (Priority: P3)

As a developer, I want to view a complete log of every HTTP request the agent made during a sandbox session, so that I can audit agent behavior, debug failures, and understand what external services were contacted.

**Why this priority**: Observability is critical for trust and debugging, but it builds on top of the proxy infrastructure established in US1 and US2. A sandbox with isolation and secret injection is already useful before the log viewer exists.

**Independent Test**: Can be tested by running a sandbox session, having the agent make several HTTP requests, and then querying the log to verify all requests are recorded with full details.

**Acceptance Scenarios**:

1. **Given** a completed sandbox session, **When** I query the request log for that session, **Then** I see every HTTP request with method, URL, status code, latency, and timestamps.
2. **Given** a session where secrets were injected, **When** I view the request log, **Then** secret values are redacted and replaced with their placeholder UUIDs.
3. **Given** multiple sandbox sessions, **When** I query logs, **Then** I can filter by session ID, time range, target host, or status code.
4. **Given** a request that failed (network error, timeout, 5xx response), **When** I view the log entry, **Then** the error details are recorded with enough context to reproduce and debug the issue.

---

### Edge Cases

- What happens when the sandbox loses network connectivity mid-session? The proxy MUST log the connection failure and the agent MUST receive a clear network error (not a silent hang).
- What happens when a secret placeholder UUID is used by the agent in a request body (not just headers)? The system MUST only inject secrets into configured header positions, not perform arbitrary string replacement in request bodies.
- What happens when the sandbox runs out of disk space due to persistent state accumulation? The system MUST surface a clear error to the operator with guidance on cleanup.
- What happens when two sandbox sessions run concurrently? Each session MUST have an independent correlation ID and logs MUST NOT intermingle.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an isolated execution environment where coding agents run without direct access to the host filesystem, network, or processes beyond explicitly configured mounts.
- **FR-002**: System MUST allow operators to declare directory mounts with explicit access modes (read-write or read-only) that are enforced at the infrastructure level.
- **FR-003**: System MUST route all outbound HTTP/HTTPS traffic from the sandbox through a transparent intercepting proxy.
- **FR-004**: System MUST support secret configuration that maps placeholder identifiers to actual secret values, associated with specific target hosts and header positions.
- **FR-005**: System MUST inject configured secrets into outbound requests at the proxy level, ensuring the actual secret value never enters the sandbox environment.
- **FR-006**: System MUST log every HTTP request and response passing through the proxy, including method, URL, headers (with secrets redacted), request/response body, status code, and latency.
- **FR-007**: System MUST store logs in a structured, queryable format with JSON fields, searchable by session ID, timestamp, URL, and status code.
- **FR-008**: System MUST assign a unique session ID to each sandbox session and correlate all log entries to that session.
- **FR-009**: System MUST persist environment changes (installed packages, tool configurations) across sandbox restarts.
- **FR-010**: System MUST provide a declarative configuration format for specifying the initial sandbox contents (pre-installed tools, runtimes, base development setup).
- **FR-011**: System MUST never write actual secret values to any log, database field, or file accessible within or from the sandbox.

### Key Entities

- **Sandbox Session**: A single execution of the sandbox environment; identified by a unique session ID; has a lifecycle (created, running, stopped); associated with a configuration and a set of log entries.
- **Secret Mapping**: A configuration entry that associates a placeholder UUID with an actual secret value and a target host/header; used by the proxy for injection; the actual value is never exposed inside the sandbox.
- **Request Log Entry**: A record of a single HTTP request/response pair captured by the proxy; contains method, URL, headers, body, status, latency, timestamp, and session ID; secrets are redacted before storage.
- **Sandbox Configuration**: A declarative specification of a sandbox environment including base image/tools, directory mounts with access modes, secret mappings, and network policies.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent running inside the sandbox cannot access any host path that was not explicitly mounted — verified by attempting access to 5 distinct unmounted paths with 100% denial rate.
- **SC-002**: Secret values configured for injection are never visible inside the sandbox — verified by searching all environment variables, mounted files, and process memory within the sandbox for the secret value.
- **SC-003**: 100% of outbound HTTP requests from the sandbox are captured in the request log with complete metadata (method, URL, status, latency, timestamp, session ID).
- **SC-004**: Operators can find any specific request from a session within 10 seconds using session ID and basic filters (host, status code, time range).
- **SC-005**: Sandbox environment changes (installed packages) persist across at least 3 consecutive restart cycles without data loss.
- **SC-006**: Secret injection adds less than 50ms of additional latency to proxied requests (p95).
- **SC-007**: Concurrent sandbox sessions produce fully independent, non-interleaved log streams — verified by running 2 simultaneous sessions and confirming zero cross-contamination of session IDs in logs.
