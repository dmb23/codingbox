# Implementation Plan: Default Sandbox Image and Config Defaults

**Branch**: `004-default-sandbox-image` | **Date**: 2026-03-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-default-sandbox-image/spec.md`

## Summary

Create a default container image with Claude Code, Mistral Vibe, OpenCode, and development tools. Add auto-mounting of host config directories (git, agent configs) at the same paths. Add global default image config so `codingbox run` works with zero arguments.

## Technical Context

**Language/Version**: Go 1.22+ (existing codebase)
**Primary Dependencies**: Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite
**New artifacts**: Dockerfile for default sandbox image, entrypoint.sh for UID/GID matching
**Testing**: `go test` + Docker-based E2E verification
**Target Platform**: macOS (primary), Linux
**Project Type**: CLI tool enhancement + container image
**Constraints**: Image must stay under 2 GB compressed; auto-mounts must not error on missing paths

## Constitution Check

*GATE: Must pass before implementation.*

### Principle I: Verify Before Assuming Success (NON-NEGOTIABLE)

| Gate | Status | How |
|------|--------|-----|
| Default image verified | PASS | Build image, run sandbox, verify all 3 agents callable |
| Auto-mounts verified | PASS | Run sandbox, verify git config and agent dirs mounted at correct paths |
| Default image fallback verified | PASS | Run `codingbox run` with no config, verify it uses default image |
| UID/GID matching verified | PASS | Create file in mounted dir from container, verify host ownership matches |

## Project Structure

### Files Added

```text
Dockerfile                           # Default sandbox image definition
docker/entrypoint.sh                 # UID/GID matching entrypoint
internal/config/automount.go         # Auto-mount detection and generation
internal/cli/config_default.go       # config set-default / show-default commands
tests/unit/automount_test.go         # Auto-mount unit tests
```

### Files Modified

```text
internal/config/central.go           # Add GlobalDefaults to store
internal/config/config.go            # Add default image fallback
internal/sandbox/sandbox.go          # Add auto-mounts + UID/GID passing
internal/cli/run.go                  # Add --no-auto-mounts flag
README.md                            # Document default image, auto-mounts
```

**Structure Decision**: Dockerfile at repo root. Entrypoint script in `docker/` directory. Auto-mount logic in `internal/config/automount.go` to keep it separate from core config loading.

## Complexity Tracking

> No constitution violations to justify.
