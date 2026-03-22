# codingbox

Secure, isolated sandbox environments for coding agents. Runs agents inside Docker Desktop microVMs with kernel-level isolation, transparent secret injection via a MITM proxy, and full HTTP request observability logged to SQLite.

## Features

- **MicroVM isolation** — each sandbox runs in its own microVM with a separate kernel (not just a container)
- **Transparent secret injection** — secrets are injected into outbound HTTP requests by host. The agent never sees real credentials, only placeholder UUIDs
- **Full request observability** — every HTTP request/response is logged to SQLite with method, URL, headers, status, latency, and secrets audit trail
- **Secret redaction** — real secret values are replaced with placeholders before being written to the database
- **Declarative config** — define your sandbox with a single `codingbox.yml` file
- **Persistent state** — installed packages and tool configs survive sandbox restarts

## Prerequisites

- **Docker Desktop 4.58+** (macOS or Windows) with sandbox/microVM support enabled
- **Go 1.22+** (for building from source)

## Install

```bash
# Build from source
git clone <repo-url>
cd codingbox
go build -o codingbox ./cmd/codingbox

# Or install directly
go install ./cmd/codingbox
```

## Quick Start

### 1. Generate a config file

```bash
codingbox config init
```

### 2. Edit `codingbox.yml`

```yaml
name: my-project
agent: claude
workspace: /Users/me/projects/my-app

secrets:
  - name: openai-api-key
    host: api.openai.com
    header: Authorization
    template: "Bearer {secret}"
    value: sk-proj-your-key-here

  - name: github-token
    host: api.github.com
    header: Authorization
    template: "token {secret}"
    value: ghp_your-token-here
```

### 3. Launch the sandbox

```bash
codingbox up
```

This will:
1. Create a Docker microVM
2. Start the MITM proxy (auto-assigned port)
3. Load the agent base image into the microVM
4. Run the agent container with workspace mounted and proxy configured
5. Stream agent output to your terminal

Press `Ctrl+C` for graceful shutdown, or run in the background with `--detach`.

### 4. View request logs

```bash
# List active sessions
codingbox ps

# View all requests from a session
codingbox logs <session-id>

# Filter by host
codingbox logs <session-id> --host api.openai.com

# Filter by status code
codingbox logs <session-id> --status 200

# JSON output
codingbox logs <session-id> --format json
```

### 5. Stop the sandbox

```bash
codingbox down <session-id>
```

## Commands

| Command | Description |
|---------|-------------|
| `codingbox up` | Start a sandbox session |
| `codingbox down <id>` | Stop a running session |
| `codingbox ps` | List sessions |
| `codingbox logs <id>` | Query request logs |
| `codingbox config init` | Generate starter config |
| `codingbox config validate` | Validate a config file |

### `codingbox up`

```
codingbox up [--config <path>] [--name <name>] [--detach]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `./codingbox.yml` | Path to config file |
| `--name` | auto | Session name |
| `--detach` | false | Run in background, print session ID |

### `codingbox down`

```
codingbox down <session-id | session-name> [--force]
```

### `codingbox ps`

```
codingbox ps [--all] [--format table|json]
```

### `codingbox logs`

```
codingbox logs <session-id> [--host <host>] [--status <code>] [--since <ts>] [--until <ts>] [--format table|json] [--limit <n>]
```

## Configuration

### Sandbox config (`codingbox.yml`)

```yaml
name: my-sandbox
agent: claude
workspace: /absolute/path/to/project

mounts:
  - host: /path/on/host
    sandbox: /path/in/sandbox
    mode: ro            # ro (default) or rw

secrets:
  - name: openai-api-key
    host: api.openai.com
    header: Authorization
    template: "Bearer {secret}"
    value: sk-proj-...

base_image: ubuntu:22.04   # optional
tools:                      # optional
  - node:20
  - python:3.12

proxy:
  port: 0                  # 0 = auto-assign
```

### Global config (`~/.config/codingbox/config.yml`)

```yaml
db_path: ~/.local/share/codingbox/codingbox.db
log_retention_days: 30
ca_cert_path: ~/.local/share/codingbox/ca.pem
ca_key_path: ~/.local/share/codingbox/ca-key.pem
```

## How It Works

```
[Agent in MicroVM Container]
    -> HTTP_PROXY=http://host.docker.internal:{port}
    -> HTTPS_PROXY=http://host.docker.internal:{port}
        -> [MITM Proxy on Host]
            -> Secret injection per host
            -> Request/response logging to SQLite
            -> Forward to internet
```

1. **Isolation**: The agent runs inside a Docker Desktop microVM — a lightweight VM with its own kernel. Host paths outside explicit mounts are inaccessible.
2. **Secret injection**: The proxy matches outbound requests by hostname and injects the configured header (e.g., `Authorization: Bearer <real-key>`). Inside the sandbox, the agent only sees environment variables like `OPENAI_API_KEY_PLACEHOLDER=<uuid>`.
3. **Observability**: Every request/response is logged to SQLite with full metadata. Secret values are redacted to placeholder UUIDs before persistence — real secrets never touch the database.

## Development

```bash
# Run tests
go test ./...

# Build
go build -o codingbox ./cmd/codingbox

# Run with verbose logging
codingbox --verbose up
```

## Project Structure

```
cmd/codingbox/          CLI entrypoint
internal/
  cli/                  Cobra command definitions
  config/               YAML config parsing
  models/               Shared types
  proxy/                MITM proxy, secret injection, logging
  sandbox/              MicroVM client, container management
  store/                SQLite storage layer
tests/
  contract/             Proxy behavior contracts
  unit/                 Pure logic tests
```
