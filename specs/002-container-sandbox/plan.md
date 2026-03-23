# Implementation Plan: Container Sandbox for Agentic Workloads

**Branch**: `002-container-sandbox` | **Date**: 2026-03-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-container-sandbox/spec.md`

## Summary

Build a CLI tool (`codingbox`) that launches OCI containers as sandboxed environments for coding agents. The sandbox mounts host directories, routes all outbound HTTP/HTTPS traffic through an embedded MITM proxy that logs to SQLite, and transparently injects secrets via placeholder replacement. Defined via YAML config with CLI flag overrides.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: Docker SDK (`github.com/docker/docker/client`), cobra/viper (CLI + config), goproxy (`github.com/elazarl/goproxy`) for MITM proxy
**Storage**: SQLite via `modernc.org/sqlite` (pure Go, no CGO) for traffic logs
**Testing**: `go test` + `testcontainers-go` for integration tests
**Target Platform**: macOS (primary), Linux
**Project Type**: CLI tool
**Performance Goals**: <30s from invocation to interactive session (SC-001)
**Constraints**: Requires Docker daemon; single-container sessions; HTTP/HTTPS only (non-HTTP blocked)
**Scale/Scope**: Single user, single concurrent sandbox session (multi-session is edge case)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Verify Before Assuming Success (NON-NEGOTIABLE)

| Gate | Status | How |
|------|--------|-----|
| CLI commands verified by execution | ✅ PASS | Every implementation task includes running the command and inspecting output |
| Tests run, not just written | ✅ PASS | Integration tests via testcontainers-go execute real Docker containers |
| No compilation-only verification | ✅ PASS | Test plan requires `go test ./...` execution at every checkpoint |

**Post-Phase 1 re-check**: ✅ PASS — All design artifacts (contracts, data model) are verifiable through the CLI and integration tests defined in the test strategy.

## Project Structure

### Documentation (this feature)

```text
specs/002-container-sandbox/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli-commands.md  # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── codingbox/
    └── main.go              # Entry point

internal/
├── cli/
│   ├── root.go              # Root cobra command
│   ├── run.go               # `codingbox run` command
│   ├── logs.go              # `codingbox logs` command
│   ├── init.go              # `codingbox init` command
│   └── ca.go                # `codingbox ca` command
├── config/
│   └── config.go            # YAML config loading + CLI flag merging
├── sandbox/
│   ├── sandbox.go           # Sandbox lifecycle (create, start, stop, cleanup)
│   ├── network.go           # Docker network management (isolated bridge)
│   └── attach.go            # Interactive TTY attachment
├── proxy/
│   ├── proxy.go             # MITM proxy setup + lifecycle
│   ├── logger.go            # Request/response logging to SQLite
│   ├── secrets.go           # Secret placeholder replacement handlers
│   └── certs.go             # CA certificate generation + management
├── store/
│   ├── store.go             # SQLite database init + migrations
│   └── queries.go           # Traffic log insert + query operations
└── models/
    ├── config.go            # SandboxConfig, MountConfig, SecretMapping structs
    ├── sandbox.go           # Sandbox runtime state struct
    └── traffic.go           # TrafficLog struct

tests/
├── integration/
│   ├── sandbox_test.go      # Full sandbox lifecycle test
│   ├── proxy_test.go        # Proxy logging + secret injection test
│   └── cleanup_test.go      # Resource cleanup verification
└── unit/
    ├── config_test.go       # Config loading + merging
    ├── secrets_test.go      # Secret replacement logic
    └── store_test.go        # SQLite operations

go.mod
go.sum
codingbox.yaml.example       # Example configuration file
```

**Structure Decision**: Single project layout. The tool is a standalone CLI binary with no frontend/backend split. All code lives under `cmd/` (entry point) and `internal/` (library code). Tests are separated into integration (requires Docker) and unit (no external dependencies).

## Complexity Tracking

> No constitution violations to justify.
