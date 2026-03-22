# CLI Contract: codingbox

**Date**: 2026-03-22
**Binary**: `codingbox`
**Config file**: `codingbox.yml` (or `~/.config/codingbox/config.yml`)

## Commands

### `codingbox up`

Start a sandbox session from configuration.

```
codingbox up [--config <path>] [--name <session-name>] [--detach]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `./codingbox.yml` | Path to sandbox configuration file |
| `--name` | auto-generated | Human-readable session name |
| `--detach` | false | Run in background, print session ID |

**Exit codes**: 0 = session ended cleanly, 1 = session failed, 2 = config error

**stdout** (detach mode): Session ID on single line
**stdout** (foreground mode): Agent container output streamed

---

### `codingbox down`

Stop a running sandbox session.

```
codingbox down <session-id | session-name> [--force]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | false | Kill immediately without graceful shutdown |

**Exit codes**: 0 = stopped, 1 = not found or already stopped

---

### `codingbox ps`

List sandbox sessions.

```
codingbox ps [--all] [--format <table|json>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | false | Include stopped sessions |
| `--format` | `table` | Output format |

**stdout** (table):
```
SESSION ID          NAME        STATUS    AGENT    CREATED
01JFXYZ...          my-project  running   claude   2026-03-22T10:00:00Z
```

**stdout** (json): Array of session objects

---

### `codingbox logs`

Query request logs for a session.

```
codingbox logs <session-id> [--host <hostname>] [--status <code>] [--since <timestamp>] [--until <timestamp>] [--format <table|json>] [--limit <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | (all) | Filter by target hostname |
| `--status` | (all) | Filter by HTTP status code |
| `--since` | (none) | Only entries after this timestamp |
| `--until` | (none) | Only entries before this timestamp |
| `--format` | `table` | Output format |
| `--limit` | 100 | Maximum entries to return |

**stdout** (table):
```
TIMESTAMP             METHOD  URL                          STATUS  LATENCY  SECRETS
2026-03-22T10:01:00Z  POST    https://api.openai.com/...   200     145ms    [openai-key]
2026-03-22T10:01:02Z  GET     https://registry.npmjs.org/  200     89ms     []
```

**stdout** (json): Array of RequestLogEntry objects (secrets redacted)

---

### `codingbox config validate`

Validate a sandbox configuration file.

```
codingbox config validate [--config <path>]
```

**Exit codes**: 0 = valid, 1 = invalid (errors printed to stderr)

---

### `codingbox config init`

Generate a starter configuration file.

```
codingbox config init [--output <path>]
```

**Exit codes**: 0 = written, 1 = file already exists

## Configuration File Format

```yaml
# codingbox.yml
name: my-project-sandbox
agent: claude

workspace: /Users/me/projects/my-app

mounts:
  - host: /Users/me/.ssh/config
    sandbox: /home/agent/.ssh/config
    mode: ro

secrets:
  - name: openai-api-key
    host: api.openai.com
    header: Authorization
    template: "Bearer {secret}"
    value: sk-proj-abc123...

  - name: github-token
    host: api.github.com
    header: Authorization
    template: "token {secret}"
    value: ghp_abc123...

base_image: codingbox/agent-base:latest

tools:
  - node:20
  - python:3.12
  - ripgrep

proxy:
  port: 0  # 0 = auto-assign
```

## Global Configuration

```yaml
# ~/.config/codingbox/config.yml
db_path: ~/.local/share/codingbox/codingbox.db
log_retention_days: 30
ca_cert_path: ~/.local/share/codingbox/ca.pem
ca_key_path: ~/.local/share/codingbox/ca-key.pem
```

## Error Output Convention

All errors go to stderr. Format:
```
codingbox: error: <message>
codingbox: warning: <message>
```
