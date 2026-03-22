# Implementation Plan: Secure Agent Sandbox

**Branch**: `001-secure-agent-sandbox` | **Date**: 2026-03-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-secure-agent-sandbox/spec.md`

## Summary

Build `codingbox`, a CLI tool that creates secure, isolated sandbox environments for coding agents using Docker Desktop's undocumented microVM API. MicroVMs provide kernel-level isolation (separate kernel per sandbox). All outbound HTTP/HTTPS traffic routes through a host-side MITM proxy that injects secrets per-host and logs every request to SQLite for full observability. Secrets never enter the sandbox — agents see only placeholder UUIDs.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: elazarl/goproxy (MITM proxy), docker/docker/client (Docker SDK), spf13/cobra (CLI), modernc.org/sqlite (pure-Go SQLite)
**Storage**: SQLite with JSON columns (host-side, never sandbox-accessible)
**Testing**: Go `testing` + testify assertions + httptest for proxy contract tests
**Target Platform**: macOS, Windows (Docker Desktop 4.58+ with microVM support; no Linux)
**Project Type**: CLI tool
**Performance Goals**: Secret injection <50ms p95 latency overhead (SC-006)
**Constraints**: Single binary distribution, no runtime dependencies beyond Docker Desktop
**Scale/Scope**: Developer tool, single machine, 1-5 concurrent sandbox sessions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Pre-Research | Post-Design | Notes |
|---|-----------|-------------|-------------|-------|
| I | Transparency First | PASS | PASS | All HTTP logged via MITM proxy; secret injection auditable via `secrets_injected` field; config snapshots stored per session |
| II | Observability by Default | PASS | PASS | Structured JSON logs in SQLite; session correlation IDs (ULID); error entries include context; queryable via `codingbox logs` |
| III | Secure Isolation | PASS | PASS | MicroVM kernel-level isolation; proxy on host (secrets never in sandbox); CA cert for HTTPS MITM; HTTP_PROXY forces all traffic through proxy |
| IV | Persistent and Reproducible State | PASS | PASS | Docker volumes in microVM for packages; declarative `codingbox.yml` config; workspace bidirectional sync |
| V | Simplicity | PASS | PASS | Single Go binary; SQLite (no external DB); Docker microVM API (3 endpoints); single project structure |

**Gate: PASSED** — No violations.

### Observability Standards Compliance

- **Log Storage**: SQLite with JSON columns, indexed by session_id, timestamp, host, status_code ✓
- **Secret Redaction**: Proxy redacts secret values to placeholder UUIDs before SQLite write ✓
- **Correlation**: ULID session IDs; all log entries reference session_id FK ✓
- **Retention**: `log_retention_days` configurable in global config ✓

### Development Workflow Compliance

- **Spec before implementation**: This plan + spec.md exist ✓
- **Threat-model review**: Required in PR for isolation boundary and secret handling changes ✓
- **Integration tests for proxy**: Contract tests verify injection + redaction ✓
- **Observability validated first**: Logging infrastructure is Phase 1 of implementation ✓

## Project Structure

### Documentation (this feature)

```text
specs/001-secure-agent-sandbox/
├── plan.md              # This file
├── research.md          # Phase 0 output - technology decisions
├── data-model.md        # Phase 1 output - entities and schema
├── quickstart.md        # Phase 1 output - getting started guide
├── contracts/           # Phase 1 output - CLI interface contract
│   └── cli-contract.md
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── codingbox/
    └── main.go              # Entrypoint

internal/
├── cli/                     # Cobra command definitions
│   ├── root.go
│   ├── up.go
│   ├── down.go
│   ├── ps.go
│   ├── logs.go
│   └── config.go
├── sandbox/                 # MicroVM lifecycle management
│   ├── client.go            # sandboxd.sock HTTP client
│   ├── session.go           # Session state machine
│   └── container.go         # Container operations inside microVM
├── proxy/                   # MITM proxy with secret injection
│   ├── server.go            # Proxy server lifecycle
│   ├── interceptor.go       # Request/response interception hooks
│   ├── secrets.go           # Secret injection logic
│   ├── logger.go            # Request logging to SQLite
│   └── ca.go                # CA certificate generation/management
├── config/                  # Configuration parsing
│   ├── sandbox.go           # codingbox.yml schema
│   └── global.go            # Global config (~/.config/codingbox/)
├── store/                   # SQLite storage layer
│   ├── db.go                # Database initialization, migrations
│   ├── sessions.go          # Session CRUD
│   └── logs.go              # Request log CRUD + queries
└── models/                  # Shared types
    ├── session.go
    ├── secret.go
    ├── log_entry.go
    └── config.go

tests/
├── contract/                # Proxy behavior contracts
│   ├── injection_test.go
│   ├── redaction_test.go
│   └── logging_test.go
├── integration/             # Full sandbox lifecycle tests
│   └── sandbox_test.go
└── unit/                    # Pure logic unit tests
    ├── config_test.go
    ├── store_test.go
    └── session_test.go
```

**Structure Decision**: Single Go project (Option 1 adapted for Go conventions). `cmd/` for binary entrypoint, `internal/` for unexported packages, `tests/` for test categories. No frontend, no web service — pure CLI tool.

## Complexity Tracking

> No constitution violations to justify. Design satisfies all five principles without exceptions.
