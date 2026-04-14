# CLI Contract Changes: Env-Based Secrets and Central Configuration

**Date**: 2026-03-26
**Feature**: 003-env-secrets-central-config

## Changes to Existing Commands

### `codingbox run` (modified)

**New behavior**: When no `--config` flag and no local `codingbox.yaml`, looks up central config for the current directory.

**New flag**:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--env-secret` | `-e` | string[] | Env secret in `ENV_NAME[:headers,body,query]` format (repeatable). Reads value from host env. |

**Error messages**:
- No config found: `Error: no configuration found for <dir>. Run 'codingbox init' or 'codingbox config set --image <image>' to set up.`
- Host env var missing: `Error: secret env var "ANTHROPIC_API_KEY" is not set on the host. Set it or provide an explicit value in the config.`

---

### `codingbox init` (modified)

**New flag**:

| Flag | Type | Description |
|------|------|-------------|
| `--env-secret` | string[] | Pre-fill an env secret entry (repeatable) |

---

## New Command: `codingbox config`

### `codingbox config set`

Register or update a central configuration entry for a directory.

```text
codingbox config set [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` (cwd) | Target directory |
| `--image` | `-i` | string | | OCI image |
| `--mount` | `-m` | string[] | | Mount `source:target[:ro\|rw]` (repeatable) |
| `--env-secret` | `-e` | string[] | | Env secret `ENV_NAME[:locations]` (repeatable) |
| `--secret` | `-s` | string[] | | Legacy secret `placeholder=value[:locations]` (repeatable) |
| `--proxy-port` | | int | 0 | Proxy port |

**Stdout**: Confirmation message with directory path and key settings.

---

### `codingbox config list`

List all registered directory configurations.

```text
codingbox config list
```

**Stdout**: Table of directory → image, secrets count, mounts count.

---

### `codingbox config remove`

Remove the central config entry for a directory.

```text
codingbox config remove [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` (cwd) | Target directory |

**Stdout**: Confirmation message.

---

## Configuration File Format Changes

### New env secret format in `codingbox.yaml`

```yaml
secrets:
  # New: env-based secret (reads from host env)
  - env: "ANTHROPIC_API_KEY"
    replace_in: ["headers"]

  # New: env-based with explicit value override
  - env: "GITHUB_TOKEN"
    value: "ghp_override_value"
    replace_in: ["headers", "body"]

  # Legacy: still supported
  - placeholder: "__CUSTOM_TOKEN__"
    value: "manual-secret"
    replace_in: ["headers"]
```

### Central config file (`~/.codingbox/directories.yaml`)

```yaml
directories:
  /Users/dev/project-a:
    image: "my-agent:latest"
    secrets:
      - env: "ANTHROPIC_API_KEY"
        replace_in: ["headers"]
    mounts:
      - source: "/shared/libs"
        target: "/libs"
        mode: "ro"
  /Users/dev/project-b:
    image: "ubuntu:22.04"
    proxy_port: 8080
```
