# Quickstart: Default Sandbox Image

## Zero-Config First Run

```bash
# Just run it — uses the default sandbox image, auto-mounts config dirs
codingbox run
```

Inside the sandbox you have:
- `claude` — Claude Code
- `mistral-vibe` — Mistral Vibe
- `opencode` — OpenCode
- `nvim` — Neovim
- `git` — with your host identity

Your project directory is at `/workspace`. All HTTP traffic is logged.

## Verify Git Config

```bash
# Inside the sandbox
git config user.name    # should match your host identity
git config user.email   # should match your host identity
```

## Verify Agents

```bash
claude --version
opencode --version
mistral-vibe --version
```

## Disable Auto-Mounts

```bash
codingbox run --no-auto-mounts
```

## Set a Custom Default Image

```bash
codingbox config set-default --image my-custom-sandbox:latest
codingbox config show-default
# Output: my-custom-sandbox:latest
```
