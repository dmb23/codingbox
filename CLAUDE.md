# codingbox Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-27

## Active Technologies
- Go 1.22+ (existing) + Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite (003-env-secrets-central-config)
- `~/.codingbox/directories.yaml` (new YAML file for central config) (003-env-secrets-central-config)
- Go 1.22+ (existing codebase) + Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite (004-default-sandbox-image)

- Go 1.22+ + Docker SDK (`github.com/docker/docker/client`), cobra/viper (CLI + config), goproxy (`github.com/elazarl/goproxy`) for MITM proxy (002-container-sandbox)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.22+

## Code Style

Go 1.22+: Follow standard conventions

## Recent Changes
- 004-default-sandbox-image: Added Go 1.22+ (existing codebase) + Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite
- 003-env-secrets-central-config: Added Go 1.22+ (existing) + Existing — cobra/viper, Docker SDK, goproxy, modernc.org/sqlite

- 002-container-sandbox: Added Go 1.22+ + Docker SDK (`github.com/docker/docker/client`), cobra/viper (CLI + config), goproxy (`github.com/elazarl/goproxy`) for MITM proxy

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
