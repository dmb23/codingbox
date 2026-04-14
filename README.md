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

## Default sandbox image

The default image (`codingbox/sandbox:latest`) includes:
- **Agents**: Claude Code, Mistral Vibe, OpenCode
- **Editor**: neovim
- **Languages**: Node.js 22, Python 3.12+ (via uv), Go 1.24
- **Tools**: git, curl, jq, ripgrep, fd-find, build-essential, zsh, ruff, ty

### Auto-mounted config directories

codingbox automatically mounts these host paths into the container (at the same absolute path) if they exist:

| Host Path | Mode | Purpose |
|-----------|------|---------|
| `~/.gitconfig` | read-only | Git identity |
| `~/.config/git/` | read-only | Git config dir |
| `~/.claude/` | read-write | Claude Code config + sessions |
| `~/.claude.json` | read-write | Claude Code global settings |
| `~/.vibe/` | read-write | Mistral Vibe config |
| `~/.config/opencode/` | read-write | OpenCode settings |
| `~/.local/share/opencode/` | read-write | OpenCode data |

Missing paths are silently skipped. Disable with `--no-auto-mounts`.

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

codingbox loads config from (in order of precedence):
1. `--config` flag (explicit path)
2. `./codingbox.yaml` (local config file)
3. `~/.codingbox/directories.yaml` (central per-directory config)

CLI flags override all config sources.

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

# Secrets: reads value from host environment automatically
secrets:
  - env: "ANTHROPIC_API_KEY"
    replace_in: ["headers"]
  - env: "GITHUB_TOKEN"
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
  --env-secret ANTHROPIC_API_KEY
```

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file (default: `./codingbox.yaml`) |
| `--image` | `-i` | OCI image (overrides config) |
| `--workdir` | `-w` | Working directory to mount (overrides config) |
| `--mount` | `-m` | Additional mount `source:target[:ro\|rw]` (repeatable) |
| `--env-secret` | `-e` | Env secret `ENV_NAME[:headers,body,query]` (repeatable) |
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

### `codingbox config`

Manage central per-directory configurations stored at `~/.codingbox/directories.yaml`.

```bash
# Register a config for the current directory
codingbox config set --image my-agent:latest --env-secret ANTHROPIC_API_KEY

# List all registered directories
codingbox config list

# Update an existing entry
codingbox config set --image new-image:latest

# Remove a registration
codingbox config remove

# Register for a specific directory
codingbox config set --dir /path/to/project --image ubuntu:22.04
```

After registering, `codingbox run` works with zero arguments from that directory (or any subdirectory).

```bash
# Set a custom global default image
codingbox config set-default --image my-custom:latest

# Show current default image
codingbox config show-default
```

### `codingbox ca`

Manage the CA certificate used for HTTPS interception.

```bash
codingbox ca show          # show cert path and fingerprint
codingbox ca regenerate    # generate a new CA cert
```

## Secret injection

Secrets are injected so agents never see real credentials:

1. **Outbound requests**: placeholders are replaced with real values before forwarding
2. **Inbound responses**: real values are replaced back with placeholders
3. **Inside the container**: only placeholders are visible

### Env-based secrets (recommended)

Specify an environment variable name. The real value is read from the host environment automatically -- no secrets in config files.

```yaml
secrets:
  - env: "ANTHROPIC_API_KEY"
    replace_in: ["headers"]

  - env: "GITHUB_TOKEN"
    replace_in: ["headers", "body"]
```

Inside the sandbox, `$ANTHROPIC_API_KEY` is set to an auto-generated placeholder like `__CODINGBOX_ANTHROPIC_API_KEY_a1b2c3d4__`. When the agent uses this in an HTTP request, the proxy replaces it with the real key.

You can also pass env secrets via CLI:

```bash
codingbox run --env-secret ANTHROPIC_API_KEY --env-secret GITHUB_TOKEN:headers
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
- **Central config**: `~/.codingbox/directories.yaml` (per-directory configs)
- **CA certificate**: `~/.codingbox/ca/codingbox-ca.pem`
- **CA private key**: `~/.codingbox/ca/codingbox-ca-key.pem`

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Session ended normally |
| 1 | Configuration error |
| 2 | Docker error |
| 3 | Proxy/TLS error |
