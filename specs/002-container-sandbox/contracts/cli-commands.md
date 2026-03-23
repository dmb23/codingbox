# CLI Contract: codingbox

**Date**: 2026-03-23
**Feature**: 002-container-sandbox

## Command Structure

```text
codingbox <command> [flags]
```

## Commands

### `codingbox run`

Launch a sandbox session from a configuration file or CLI flags.

```text
codingbox run [flags]
```

**Flags**:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | `./codingbox.yaml` | Path to configuration file |
| `--image` | `-i` | string | (from config) | OCI image to use (overrides config) |
| `--workdir` | `-w` | string | `.` | Working directory to mount (overrides config) |
| `--mount` | `-m` | string[] | (from config) | Additional mount in `source:target[:ro\|rw]` format (repeatable, appends to config) |
| `--secret` | `-s` | string[] | (from config) | Secret mapping in `placeholder=value[:headers,body,query]` format (repeatable, appends to config) |
| `--proxy-port` | | int | 0 (auto) | Port for MITM proxy (overrides config) |

**Exit codes**:

| Code | Meaning |
|------|---------|
| 0 | Session ended normally |
| 1 | Configuration error (missing image, invalid config file, etc.) |
| 2 | Docker error (daemon not running, image not found, etc.) |
| 3 | Proxy error (port in use, TLS setup failed, etc.) |

**Stdin**: Forwarded to the container's interactive terminal session.
**Stdout**: Container's terminal output.
**Stderr**: Error messages from codingbox itself (not the container).

---

### `codingbox logs`

Query traffic logs from a previous or current sandbox session.

```text
codingbox logs [flags]
```

**Flags**:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--session` | | string | (latest) | Session ID to query (default: most recent) |
| `--method` | | string | (all) | Filter by HTTP method |
| `--url` | | string | (all) | Filter by URL pattern (substring match) |
| `--status` | | int | (all) | Filter by response status code |
| `--since` | | string | (all) | Show logs since timestamp (RFC3339) |
| `--limit` | `-n` | int | 50 | Maximum number of entries |
| `--format` | `-f` | string | `table` | Output format: `table`, `json` |
| `--body` | | bool | false | Include request/response bodies in output |

**Stdout**: Formatted log entries.

---

### `codingbox init`

Generate a default configuration file in the current directory.

```text
codingbox init [flags]
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--image` | string | (empty) | Pre-fill the image field |
| `--force` | bool | false | Overwrite existing config file |

**Output**: Creates `codingbox.yaml` in the current directory.

---

### `codingbox ca`

Manage the CA certificate used for TLS interception.

```text
codingbox ca <subcommand>
```

**Subcommands**:

| Subcommand | Description |
|------------|-------------|
| `codingbox ca show` | Print path and fingerprint of current CA cert |
| `codingbox ca regenerate` | Generate a new CA certificate (invalidates cached host certs) |

---

## Configuration File Format

```yaml
# codingbox.yaml
image: "codingbox/claude:latest"
workdir: "."

mounts:
  - source: "/path/to/shared/libs"
    target: "/libs"
    mode: "ro"
  - source: "/path/to/output"
    target: "/output"
    mode: "rw"

secrets:
  - placeholder: "__GITHUB_TOKEN__"
    value: "ghp_xxxxxxxxxxxx"
    replace_in: ["headers"]
  - placeholder: "__ANTHROPIC_API_KEY__"
    value: "sk-ant-xxxxxxxxxxxx"
    replace_in: ["headers", "body"]

proxy_port: 0  # 0 = auto-assign
```

## Error Output Format

All errors are written to stderr in the format:

```text
Error: <message>
```

For `--format json` on the `logs` command, errors are JSON:

```json
{"error": "<message>"}
```
