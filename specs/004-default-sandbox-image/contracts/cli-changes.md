# CLI Contract Changes: Default Sandbox Image and Config Defaults

**Date**: 2026-03-27
**Feature**: 004-default-sandbox-image

## Changes to `codingbox run`

**New flag**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--no-auto-mounts` | bool | false | Disable automatic config directory mounts |

**Changed behavior**: When no image is specified via CLI, local config, or central config, the system uses the global default image (configurable, defaults to `codingbox/sandbox:latest`).

## Changes to `codingbox config`

**New subcommand**: `codingbox config set-default`

```text
codingbox config set-default [flags]
```

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--image` | `-i` | string | Set the global default image |

**New subcommand**: `codingbox config show-default`

```text
codingbox config show-default
```

Prints the current global default image.

## Dockerfile

New file at repository root: `Dockerfile`

Defines the default sandbox image with:
- Ubuntu 24.04 base
- Node.js 22 LTS, Python 3.12+, Go
- neovim, git, curl, jq, ripgrep, fd-find, build-essential, zsh
- Claude Code, Mistral Vibe, OpenCode
- Entrypoint script that matches host UID/GID

## Central Config Store Changes

`~/.codingbox/directories.yaml` gains a `defaults` section:

```yaml
defaults:
  default_image: "codingbox/sandbox:latest"

directories:
  /Users/dev/project-a:
    image: "custom:latest"
```
