# Quickstart: codingbox

## Prerequisites

- Docker Desktop 4.58+ (macOS or Windows) with sandbox/microVM support enabled
- Go 1.22+ (for building from source)

## Install

```bash
# From source
go install github.com/<org>/codingbox@latest

# Or build locally
git clone <repo>
cd codingbox
go build -o codingbox ./cmd/codingbox
```

## First Run

### 1. Create a configuration file

```bash
codingbox config init
```

This creates `codingbox.yml` in the current directory with a starter template.

### 2. Edit the configuration

```yaml
# codingbox.yml
name: my-first-sandbox
agent: claude

workspace: /Users/me/projects/my-app

secrets:
  - name: openai-api-key
    host: api.openai.com
    header: Authorization
    template: "Bearer {secret}"
    value: sk-proj-your-key-here
```

### 3. Launch the sandbox

```bash
codingbox up
```

This will:
1. Create a Docker microVM via the sandboxd API
2. Start the MITM proxy on an auto-assigned port
3. Load the agent base image into the microVM
4. Run the agent container with workspace mounted and proxy configured
5. Stream agent output to your terminal

### 4. View request logs

In another terminal:

```bash
# List active sessions
codingbox ps

# View all requests from the session
codingbox logs <session-id>

# Filter by host
codingbox logs <session-id> --host api.openai.com

# JSON output for scripting
codingbox logs <session-id> --format json
```

### 5. Stop the sandbox

```bash
# Graceful shutdown
codingbox down <session-id>

# Or Ctrl+C in the foreground terminal
```

## Verify Isolation

From inside the sandbox, the agent:
- Can read/write files in the workspace directory
- Cannot access any other host paths
- Sees placeholder UUIDs for secrets in environment variables
- Has all HTTP traffic routed through the MITM proxy
- Cannot bypass the proxy (no direct network access)

## Verify Secret Injection

```bash
# After a session, check that secrets were injected
codingbox logs <session-id> --host api.openai.com --format json

# The log will show:
# - "secrets_injected": ["openai-api-key"]
# - Headers show the placeholder UUID, not the real key
```

## Configuration Reference

See [CLI Contract](contracts/cli-contract.md) for full command and configuration documentation.
