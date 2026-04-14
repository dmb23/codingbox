# Implementation Plan: Env-Based Secrets and Central Configuration

**Branch**: `003-env-secrets-central-config` | **Date**: 2026-03-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-env-secrets-central-config/spec.md`

## Summary

Enhance codingbox with two capabilities: (1) env var-based secrets where the developer specifies an env var name and the system auto-generates a placeholder, reading the real value from the host environment; (2) central per-directory configuration stored at `~/.codingbox/directories.yaml` enabling `codingbox run` with zero arguments in any registered directory.

## Technical Context

**Language/Version**: Go 1.22+ (existing)
**Primary Dependencies**: Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite
**Storage**: `~/.codingbox/directories.yaml` (new YAML file for central config)
**Testing**: `go test` + existing unit test patterns
**Target Platform**: macOS (primary), Linux
**Project Type**: CLI tool (enhancement)
**Constraints**: Must be backwards-compatible with existing `placeholder`/`value` secret format

## Constitution Check

*GATE: Must pass before implementation.*

### Principle I: Verify Before Assuming Success (NON-NEGOTIABLE)

| Gate | Status | How |
|------|--------|-----|
| Env secrets verified by execution | PASS | E2E test: set host env, launch sandbox, verify placeholder inside, verify proxy replaces |
| Central config verified by execution | PASS | E2E test: register config, run from directory, verify correct image/settings |
| Backwards compatibility verified | PASS | Existing unit tests + E2E: run with legacy placeholder/value format, confirm no regression |

## Project Structure

### Files Modified

```text
internal/models/config.go       # Add Env field to SecretMapping
internal/config/config.go        # Add central config lookup, env secret resolution
internal/sandbox/sandbox.go      # Pass env secret env vars to container
internal/cli/run.go              # Update config loading flow
internal/cli/init.go             # Add --env-secret flag
```

### Files Added

```text
internal/config/central.go       # DirectoryConfigStore: load/save/get/set/remove/list
internal/config/placeholder.go   # Placeholder generation from env name
internal/cli/config_cmd.go       # codingbox config set/list/remove commands
tests/unit/central_test.go       # Central config store tests
tests/unit/placeholder_test.go   # Placeholder generation tests
tests/unit/env_secret_test.go    # Env secret resolution tests
```

**Structure Decision**: All new code fits within existing package layout. The `internal/config/central.go` module encapsulates the directory config store. No new packages needed.

## Complexity Tracking

> No constitution violations to justify.
