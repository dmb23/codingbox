# Quickstart: Env-Based Secrets and Central Configuration

## Env-Based Secrets

### Before (legacy placeholder approach)

```yaml
# codingbox.yaml
secrets:
  - placeholder: "__ANTHROPIC_API_KEY__"
    value: "sk-ant-xxxxxxxxxxxx"
    replace_in: ["headers"]
```

### After (env-based approach)

```yaml
# codingbox.yaml
secrets:
  - env: "ANTHROPIC_API_KEY"
    replace_in: ["headers"]
```

The real value is read from `$ANTHROPIC_API_KEY` on the host automatically. Inside the sandbox, the agent sees `ANTHROPIC_API_KEY=__CODINGBOX_ANTHROPIC_API_KEY_a1b2c3d4__` and uses it normally. The proxy handles the rest.

### Verify

```bash
# Set the env var on your host
export ANTHROPIC_API_KEY=sk-ant-real-key

# Launch sandbox
codingbox run

# Inside the sandbox:
echo $ANTHROPIC_API_KEY
# Output: __CODINGBOX_ANTHROPIC_API_KEY_a1b2c3d4__  (placeholder, not real key)

# When the agent uses this in an API call, the proxy replaces it transparently.
```

## Central Per-Directory Configuration

### Register a directory

```bash
cd /path/to/my-project
codingbox config set --image my-agent:latest --env-secret ANTHROPIC_API_KEY
```

### Run from anywhere

```bash
cd /path/to/my-project
codingbox run
# Starts my-agent:latest with ANTHROPIC_API_KEY injected — no flags needed
```

### List registered directories

```bash
codingbox config list
```

### Remove a registration

```bash
cd /path/to/my-project
codingbox config remove
```

## Precedence

1. CLI flags (`--image`, `--env-secret`, etc.) — highest priority
2. Local `codingbox.yaml` in the current directory
3. Central config in `~/.codingbox/directories.yaml` (matches current dir or nearest parent)
4. Error with helpful message
