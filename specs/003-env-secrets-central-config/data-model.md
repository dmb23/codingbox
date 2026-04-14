# Data Model: Env-Based Secrets and Central Configuration

**Date**: 2026-03-26
**Feature**: 003-env-secrets-central-config

## Changes to Existing Entities

### SecretMapping (modified)

New `Env` field added. When `Env` is set, the system reads the real value from the host environment (or from `Value` as override) and auto-generates the `Placeholder`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| placeholder | string | conditional | Manual placeholder (legacy format). Required if `env` is not set. |
| value | string | conditional | Real secret value. Required if `env` is not set. Optional override when `env` is set. |
| env | string | conditional | Environment variable name. When set, value is read from host env, placeholder is auto-generated. |
| replace_in | []string | no | Where to perform replacement. Default: `[headers, body, query]` |

**Validation rules**:
- Exactly one of `env` or `placeholder` MUST be set (not both, not neither).
- If `env` is set and `value` is empty, the host env var MUST exist.
- If `placeholder` is set (legacy), `value` MUST be set.

### SandboxConfig (unchanged structurally)

The `Secrets` field now accepts both legacy and env-based secret formats. No structural change to SandboxConfig itself.

## New Entities

### DirectoryConfigStore

Manages the central per-directory configuration at `~/.codingbox/directories.yaml`.

| Field | Type | Description |
|-------|------|-------------|
| directories | map[string]SandboxConfig | Map of canonical directory path to sandbox config |

**Operations**:
- `Get(dir string) (*SandboxConfig, bool)` — exact match lookup
- `FindNearest(dir string) (*SandboxConfig, string, bool)` — walk up parents
- `Set(dir string, cfg SandboxConfig)` — create or update entry
- `Remove(dir string) bool` — delete entry
- `List() map[string]SandboxConfig` — return all entries
- `Save()` / `Load()` — persist to / read from YAML file

**File location**: `~/.codingbox/directories.yaml`

## Placeholder Generation

For env-based secrets, the placeholder is derived deterministically:

```
placeholder = "__CODINGBOX_" + ENV_NAME + "_" + sha256(ENV_NAME)[:8] + "__"
```

Example: `ANTHROPIC_API_KEY` → `__CODINGBOX_ANTHROPIC_API_KEY_a1b2c3d4__`

## Relationships

```text
SandboxConfig 1──* SecretMapping     (config has zero or more secrets)
SecretMapping     has either:
  - env + auto-placeholder + optional value override  (new format)
  - placeholder + value                                (legacy format)

DirectoryConfigStore 1──* SandboxConfig  (store maps dirs to configs)
```
