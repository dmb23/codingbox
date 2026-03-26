# codingbox

A sandboxed environment for running coding agents (Claude Code, OpenCode, Mistral Vibe, etc.) with full network visibility and secret isolation.

codingbox launches OCI containers with your project directory mounted, routes all HTTP/HTTPS traffic through a logging proxy, and transparently injects secrets so agents never see your real credentials.

## How it works

```
+-----------------+          +-------------+          +----------+
|   Container     |  HTTP/S  |   codingbox |  HTTP/S  | External |
|   (agent)       | -------> |   proxy     | -------> | APIs     |
|                 |          |             |          |          |
| __API_KEY__ --->| replace  | real-key -->|          |          |
|                 |          |             |          |          |
| <--- __API_KEY__|<- reverse|<-- real-key |          |          |
+-----------------+          +-------------+          +----------+
                                   |
                                   v
                              [ SQLite DB ]
                              traffic logs
```

- Your project directory is bind-mounted read-write at `/workspace`
- All outbound HTTP/HTTPS is intercepted and logged
- Secret placeholders in requests are replaced with real values at the proxy
- Real values in responses are replaced back with placeholders
- Non-HTTP traffic is blocked
- Everything is cleaned up when the session ends

## Prerequisites

- Docker installed and running
- Go 1.22+ (for building from source)

## Install

```bash
go install github.com/mischa/codingbox@latest
```

Or build from source:

```bash
git clone https://github.com/mischa/codingbox.git
cd codingbox
go build -o codingbox ./cmd/codingbox/
```

## Quick start

```bash
# 1. Generate a config file
cd /path/to/your/project
codingbox init --image ubuntu:22.04

# 2. Launch the sandbox
codingbox run

# 3. You're now inside the container at /workspace
#    Work normally. All HTTP traffic is logged.
#    Press Ctrl+D or type 'exit' to leave.

# 4. Review what the agent did
codingbox logs
```

## Configuration

codingbox uses a YAML config file (default: `./codingbox.yaml`). CLI flags override config values.

```yaml
# codingbox.yaml

# Container image (required)
image: "ubuntu:22.04"

# Host directory mounted at /workspace (default: current directory)
workdir: "."

# Additional directory mounts
mounts:
  - source: "/home/user/shared-libs"
    target: "/libs"
    mode: "ro"    # read-only (default)
  - source: "/tmp/output"
    target: "/output"
    mode: "rw"    # read-write

# Secret injection (placeholder -> real value)
secrets:
  - placeholder: "__ANTHROPIC_API_KEY__"
    value: "sk-ant-xxxxxxxxxxxx"
    replace_in: ["headers"]
  - placeholder: "__GITHUB_TOKEN__"
    value: "ghp_xxxxxxxxxxxx"
    replace_in: ["headers", "body"]

# Proxy port (0 = auto-assign)
proxy_port: 0
```

## Commands

### `codingbox run`

Launch an interactive sandbox session.

```bash
# Use config file
codingbox run

# Override image
codingbox run --image ubuntu:24.04

# Specify config path
codingbox run --config /path/to/codingbox.yaml

# Add mounts and secrets via flags
codingbox run --image ubuntu:22.04 \
  --mount /host/path:/container/path:ro \
  --secret "__TOKEN__=real-value:headers"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file (default: `./codingbox.yaml`) |
| `--image` | `-i` | OCI image (overrides config) |
| `--workdir` | `-w` | Working directory to mount (overrides config) |
| `--mount` | `-m` | Additional mount `source:target[:ro\|rw]` (repeatable) |
| `--secret` | `-s` | Secret `placeholder=value[:headers,body,query]` (repeatable) |
| `--proxy-port` | | Proxy port, 0 for auto (overrides config) |

### `codingbox logs`

Query traffic logs from sandbox sessions.

```bash
# Show recent logs
codingbox logs

# Filter by URL
codingbox logs --url api.anthropic.com

# Filter by method and status
codingbox logs --method POST --status 200

# Show request/response bodies
codingbox logs --url api.openai.com --body

# JSON output
codingbox logs --format json --limit 10

# Logs from a specific session
codingbox logs --session abc12345
```

| Flag | Short | Description |
|------|-------|-------------|
| `--session` | | Session ID (default: most recent) |
| `--method` | | Filter by HTTP method |
| `--url` | | Filter by URL substring |
| `--status` | | Filter by response status code |
| `--since` | | Logs since timestamp (RFC3339) |
| `--limit` | `-n` | Max entries (default: 50) |
| `--format` | `-f` | Output format: `table` (default), `json` |
| `--body` | | Include request/response bodies |

### `codingbox init`

Generate a default config file.

```bash
codingbox init --image ubuntu:22.04
codingbox init --force  # overwrite existing
```

### `codingbox ca`

Manage the CA certificate used for HTTPS interception.

```bash
codingbox ca show          # show cert path and fingerprint
codingbox ca regenerate    # generate a new CA cert
```

## Secret injection

Secrets are defined as placeholder-to-value mappings. The proxy handles replacement transparently:

1. **Outbound requests**: placeholders in the configured locations (headers, body, query params) are replaced with real values before forwarding
2. **Inbound responses**: real values are replaced back with placeholders before reaching the container
3. **Inside the container**: only placeholders are ever visible -- the real secret never enters the sandbox

```yaml
secrets:
  - placeholder: "__ANTHROPIC_API_KEY__"
    value: "sk-ant-real-key"
    replace_in: ["headers"]          # only replace in headers

  - placeholder: "__OPENAI_KEY__"
    value: "sk-real-openai-key"
    replace_in: ["headers", "body"]  # replace in headers and body

  - placeholder: "__API_TOKEN__"
    value: "token-123"
    replace_in: ["headers", "body", "query"]  # replace everywhere (default)
```

## Directory mounts

The working directory is always mounted read-write at `/workspace`. Additional mounts can be configured:

```yaml
mounts:
  - source: "/absolute/host/path"
    target: "/container/path"
    mode: "ro"   # read-only (default) -- writes are rejected
  - source: "/another/path"
    target: "/data"
    mode: "rw"   # read-write -- changes visible on host
```

Mounts can also be added via CLI flags:

```bash
codingbox run --mount /host/libs:/libs:ro --mount /tmp/out:/out:rw
```

## Custom sandbox images

Any OCI image works. Build your own with the agent and tools pre-installed:

```dockerfile
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl git nodejs npm
RUN npm install -g @anthropic-ai/claude-code
```

```bash
docker build -t my-agent-sandbox .
codingbox run --image my-agent-sandbox
```

If the image isn't available locally, codingbox pulls it automatically.

## Data storage

- **Traffic logs**: `~/.codingbox/traffic.db` (SQLite)
- **CA certificate**: `~/.codingbox/ca/codingbox-ca.pem`
- **CA private key**: `~/.codingbox/ca/codingbox-ca-key.pem`

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Session ended normally |
| 1 | Configuration error |
| 2 | Docker error |
| 3 | Proxy/TLS error |
