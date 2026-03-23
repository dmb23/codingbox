# Quickstart: codingbox

## Prerequisites

- Docker installed and running (`docker info` should succeed)
- Go 1.22+ (for building from source)

## Install

```bash
go install github.com/mischa/codingbox@latest
```

## First Run

### 1. Create a configuration file

```bash
cd /path/to/your/project
codingbox init --image ubuntu:22.04
```

This creates `codingbox.yaml` with default settings.

### 2. Launch a sandbox

```bash
codingbox run
```

This will:
- Start a MITM proxy on an auto-assigned port
- Create an isolated Docker network
- Launch a container from the configured image
- Mount the current directory into the container
- Drop you into an interactive terminal session

### 3. Work inside the sandbox

Inside the container, you have full terminal access. The current directory is mounted read-write. All outbound HTTP/HTTPS traffic is logged.

Press `Ctrl+D` or type `exit` to end the session. All Docker resources are cleaned up automatically.

### 4. Review traffic logs

```bash
codingbox logs
codingbox logs --url api.anthropic.com --body
codingbox logs --format json --limit 10
```

## Using Secret Injection

### 1. Add secrets to your config

```yaml
# codingbox.yaml
image: "codingbox/claude:latest"

secrets:
  - placeholder: "__ANTHROPIC_API_KEY__"
    value: "sk-ant-your-real-key-here"
    replace_in: ["headers"]
```

### 2. Inside the sandbox

The agent sees `__ANTHROPIC_API_KEY__` as the API key. When it makes a request with this placeholder in a header, the proxy transparently replaces it with the real key before forwarding. Responses containing the real key have it replaced back with the placeholder.

The agent never sees `sk-ant-your-real-key-here`.

## Adding Extra Mounts

```yaml
# codingbox.yaml
image: "codingbox/claude:latest"
mounts:
  - source: "/home/user/shared-libs"
    target: "/libs"
    mode: "ro"
  - source: "/tmp/sandbox-output"
    target: "/output"
    mode: "rw"
```

## Verify It Works

After your first session, verify the proxy logged traffic:

```bash
codingbox logs --limit 5
```

You should see a table of HTTP requests made from inside the sandbox. If the table is empty and you made requests, check that Docker is running and the proxy started without errors.
