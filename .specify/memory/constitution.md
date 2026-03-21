<!--
  Sync Impact Report
  ==================
  Version change: N/A (template) → 1.0.0
  Modified principles: N/A (initial ratification)
  Added sections:
    - Core Principles (5 principles)
    - Observability Standards
    - Development Workflow
    - Governance
  Removed sections: N/A
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ no changes needed
      (Constitution Check section already generic)
    - .specify/templates/spec-template.md ✅ no changes needed
      (FR section already supports observability requirements)
    - .specify/templates/tasks-template.md ✅ no changes needed
      (logging tasks already present in sample phases)
  Follow-up TODOs: None
-->

# Codingbox Constitution

## Core Principles

### I. Transparency First

Every action taken by or within the sandbox MUST be visible and
auditable. No silent side-effects are permitted.

- All HTTP requests leaving the sandbox MUST be intercepted, logged,
  and queryable.
- Secret injection MUST be invisible to sandbox processes but fully
  auditable by the operator (UUID mapping logged, actual secrets
  never written to sandbox-accessible storage).
- Configuration changes (directory mounts, environment variables,
  installed packages) MUST produce a durable, timestamped record.
- Rationale: A sandbox for coding agents is only trustworthy if
  operators can verify exactly what happened inside it.

### II. Observability by Default

Every component MUST emit structured, machine-readable telemetry
from the start — not as a bolt-on.

- All HTTP calls MUST be logged with method, URL, headers (secrets
  redacted), request/response body, status code, and latency.
- Logs MUST use structured formats (JSON) and be stored in a
  queryable backend (database, not flat files).
- Each sandbox session MUST have a unique correlation ID that ties
  all its activity together.
- Error conditions MUST produce actionable log entries (not just
  stack traces) that include context for reproduction.
- Rationale: Observability enables debugging, auditing, and future
  features like the log viewer without retroactive instrumentation.

### III. Secure Isolation

The sandbox boundary MUST be enforced at the infrastructure level,
not by convention or application logic.

- Sandbox processes MUST NOT have direct access to host secrets,
  credentials, or unrestricted network.
- Directory mounts MUST enforce declared access modes (read-write
  vs read-only) at the container/VM level.
- The MITM proxy MUST be the sole network egress path for sandbox
  processes.
- Rationale: Coding agents execute arbitrary code; the isolation
  boundary is the primary security control and MUST be
  infrastructure-enforced.

### IV. Persistent and Reproducible State

Sandbox state changes MUST persist predictably and be
reconstructable.

- Environment modifications (installed packages, tool configs) MUST
  survive sandbox restarts.
- The initial sandbox specification (tools, runtimes, base config)
  MUST be declarative and version-controlled.
- State persistence mechanism MUST be documented and testable (e.g.,
  container layers, volume snapshots).
- Rationale: Agents lose effectiveness if their environment resets
  unexpectedly; reproducibility enables debugging and sharing of
  sandbox configurations.

### V. Simplicity

Start with the minimum viable implementation. Complexity MUST be
justified by a concrete, current requirement.

- Prefer Docker Sandbox over custom VM orchestration unless a
  specific requirement demands otherwise.
- Avoid abstractions until a pattern repeats at least twice.
- Each component (proxy, sandbox, secret manager) SHOULD be
  independently understandable and testable.
- Rationale: Over-engineering a sandbox tool defeats the purpose of
  making agent workflows easier. YAGNI applies.

## Observability Standards

Observability is a cross-cutting concern that applies to every
component. The following standards are mandatory:

- **Log Storage**: All HTTP intercept logs MUST be stored in a
  database with JSON fields, queryable by session, timestamp, URL,
  and status code.
- **Secret Redaction**: Logs MUST replace actual secret values with
  their sandbox-side UUIDs before persistence. Raw secrets MUST
  never appear in any log output.
- **Correlation**: Every sandbox session MUST generate a unique
  session ID. All log entries, proxy records, and state changes
  MUST reference this ID.
- **Retention**: Log retention policy MUST be configurable. Default
  retention MUST be documented.

## Development Workflow

- All features MUST be developed behind a clear specification
  (spec.md) before implementation begins.
- Changes to isolation boundaries or secret handling MUST include
  a threat-model review in the PR description.
- Integration tests for the HTTP proxy MUST verify that secrets
  are correctly injected in outbound requests and redacted in logs.
- Observability infrastructure (logging, storage) MUST be validated
  in the foundational phase before feature work begins.

## Governance

This constitution is the authoritative source of project principles
for Codingbox. It supersedes informal agreements and ad-hoc
decisions.

- **Amendments**: Any change to this constitution MUST be documented
  in a PR with a clear rationale. The Sync Impact Report at the top
  of this file MUST be updated to reflect the change.
- **Versioning**: This constitution follows semantic versioning.
  MAJOR for principle removals/redefinitions, MINOR for additions
  or material expansions, PATCH for wording clarifications.
- **Compliance**: All PRs MUST be reviewable against these
  principles. The plan template's Constitution Check section MUST
  reference the active principles by number.
- **Guidance**: Use `CLAUDE.md` or equivalent agent guidance files
  for runtime development instructions that supplement (but do not
  override) this constitution.

**Version**: 1.0.0 | **Ratified**: 2026-03-21 | **Last Amended**: 2026-03-21
