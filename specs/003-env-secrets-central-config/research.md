# Research: Env-Based Secrets and Central Configuration

**Date**: 2026-03-26
**Feature**: 003-env-secrets-central-config

## R1: No New Tech Stack Decisions

This feature builds entirely on the existing codingbox codebase (Go 1.22+, cobra/viper, Docker SDK, goproxy, SQLite). No new dependencies are required.

## R2: Placeholder Generation Strategy

**Decision**: Use `__CODINGBOX_<ENV_NAME>_<8-char-hex>__` format where the hex is derived from a hash of the env var name.

**Rationale**:
- Deterministic: same env name always produces the same placeholder (enables config caching and debugging)
- Unlikely to collide with real content due to `__CODINGBOX_` prefix
- Human-readable in logs (you can see which env var a placeholder belongs to)
- 8 hex chars from SHA-256 of the env name gives sufficient uniqueness

**Alternatives considered**:
- Random UUID per session: not deterministic, hard to debug
- Simple `__ENV_NAME__`: could collide with user-defined placeholders from legacy format

## R3: Central Config Storage Format

**Decision**: Single YAML file at `~/.codingbox/directories.yaml`

**Rationale**:
- Human-readable and editable as a fallback
- Viper already supports YAML natively
- Single file avoids managing a directory of config fragments
- Per-directory lookup is a simple map keyed by canonical path

**Structure**:
```yaml
directories:
  /Users/dev/project-a:
    image: "my-agent:latest"
    secrets:
      - env: "ANTHROPIC_API_KEY"
    mounts:
      - source: "/shared/libs"
        target: "/libs"
        mode: "ro"
  /Users/dev/project-b:
    image: "ubuntu:22.04"
```

**Alternatives considered**:
- SQLite: overkill for a handful of directory entries
- One file per directory (e.g., `~/.codingbox/dirs/<hash>.yaml`): harder to browse and manage
- TOML: less familiar to Docker/container users than YAML

## R4: Config Precedence Implementation

**Decision**: Layered lookup in `config.Load()`:
1. If `--config` flag provided → use that file exclusively
2. Else if `./codingbox.yaml` exists → use it
3. Else look up current directory (and parents) in `~/.codingbox/directories.yaml`
4. If nothing found → return error with helpful message

**Rationale**: Matches user expectation (local overrides global), aligns with FR-010, and the existing `Load()` function already handles steps 1-2 — step 3 is a natural extension.
