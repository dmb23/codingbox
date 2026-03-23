# Research: Container Sandbox for Agentic Workloads

**Date**: 2026-03-23
**Feature**: 002-container-sandbox

## R1: Programming Language

**Decision**: Go 1.22+

**Rationale**:
- Docker Engine SDK is native Go (`github.com/docker/docker/client`)
- Excellent CLI tooling ecosystem (cobra, viper)
- Single static binary distribution — no runtime dependencies
- Strong concurrency primitives for proxy + container management
- Cross-compilation to macOS and Linux without CGO (when using pure Go dependencies)

**Alternatives considered**:
- **Rust**: Excellent performance but Docker SDK ecosystem is less mature; higher development cost
- **Python**: mitmproxy is Python-native, but subprocess management overhead and distribution complexity (requires Python runtime)
- **TypeScript/Node**: Good Docker libraries exist but weaker for systems-level proxy work

## R2: MITM Proxy Library

**Decision**: goproxy (`github.com/elazarl/goproxy`)

**Rationale**:
- Embeddable Go library — compiles into single binary with the CLI
- 10+ years mature, used by Stripe, Google, Grafana, Kubernetes
- Simple handler API for request/response modification (`OnRequest().Do()`, `OnResponse().Do()`)
- Full TLS interception with certificate caching (resolves 92% CPU bottleneck from RSA key generation)
- Excellent performance (native Go, no Python overhead)

**Alternatives considered**:
- **mitmproxy (Python)**: Most feature-rich but requires subprocess management, Python runtime, custom addon for SQLite logging, and has performance overhead
- **go-mitmproxy (lqqyt2423)**: Good Go port but no transparent proxy mode, smaller community
- **AdGuard gomitmproxy**: Corporate-backed but smaller ecosystem, no transparent proxy
- **sslsplit (C)**: Excellent transparent mode but cannot do HTTP-level request/response body modification needed for secret injection

**Transparent proxy workaround**: goproxy does not support transparent proxy mode. Instead:
1. Create an isolated Docker network (`Internal: true`) — containers cannot reach external networks
2. Run the proxy process on the host, exposed to the isolated network
3. Set `HTTP_PROXY` and `HTTPS_PROXY` environment variables in the container
4. Docker network isolation + iptables rules ensure no bypass is possible

## R3: Database for Traffic Logs

**Decision**: SQLite via `modernc.org/sqlite`

**Rationale**:
- Pure Go (no CGO) — enables clean cross-compilation to macOS/Linux
- ~75% performance of CGO-based mattn/go-sqlite3, which is acceptable for logging workloads
- WAL mode supports concurrent reads (user queries) while proxy writes logs
- Embedded — no external database server to manage
- Standard `database/sql` interface
- Mature and actively maintained (v1.36+, SQLite 3.49.0)

**Configuration**: WAL mode + `PRAGMA synchronous = NORMAL` + single writer connection

**Alternatives considered**:
- **mattn/go-sqlite3**: Fastest but requires CGO, complicates cross-compilation (Grafana moved away from it for this reason)
- **zombiezen/go-sqlite**: ~6x faster than modernc but non-standard API; consider if write perf becomes bottleneck
- **Badger**: ~375x faster writes but no SQL query support — users would want to query logs by URL, status code, time range
- **bbolt**: Simple and stable but no SQL, limited query capability

## R4: CLI Framework

**Decision**: cobra (`github.com/spf13/cobra`) + viper (`github.com/spf13/viper`)

**Rationale**:
- Industry standard for Go CLIs (used by Docker, Kubernetes, Hugo)
- Cobra provides subcommand structure, flag parsing, help generation
- Viper integrates with cobra for config file support (YAML) with CLI flag overrides — matches FR-014/FR-015
- Well-documented, mature

**Alternatives considered**:
- **urfave/cli**: Good but less config file integration
- **kong**: Clean API but smaller ecosystem

## R5: Configuration Format

**Decision**: YAML

**Rationale**:
- Standard in container tooling (docker-compose, Kubernetes)
- Human-readable and writable
- Viper has native YAML support
- Familiar to the target audience (developers working with Docker)

## R6: Container Networking Strategy

**Decision**: Isolated Docker bridge network with proxy as gateway

**Approach**:
1. Create a Docker bridge network with `Internal: true` (no external access for containers)
2. The proxy runs on the host and listens on a port accessible from the Docker network
3. Container is started with `HTTP_PROXY`/`HTTPS_PROXY` env vars pointing to the proxy
4. The `Internal: true` flag ensures containers cannot bypass the proxy — no masquerade/NAT rules are created
5. DNS resolution is handled by Docker's embedded DNS within the network

**Why not transparent proxy**: None of the mature Go proxy libraries support transparent mode. Using explicit proxy configuration via env vars is reliable, deterministic, and doesn't require iptables manipulation inside the container.

## R7: TLS Certificate Management

**Decision**: Generate a CA certificate on first run, inject into container trust store

**Approach**:
1. On first run, generate a self-signed CA certificate and key, stored in a config directory (e.g. `~/.codingbox/ca/`)
2. goproxy uses this CA to generate per-host certificates on-the-fly (with caching)
3. The CA certificate is bind-mounted into the container and added to the system trust store via an entrypoint script or environment variable (`SSL_CERT_FILE`, `NODE_EXTRA_CA_CERTS`, etc.)
4. This avoids requiring users to modify their OCI images

## R8: Testing Strategy

**Decision**: `go test` + `testcontainers-go` for integration tests

**Rationale**:
- `go test` is the standard Go testing framework
- `testcontainers-go` provides programmatic Docker container management for integration tests
- Enables testing the full sandbox lifecycle (create network → start proxy → start container → verify traffic → cleanup) in automated tests
- Aligns with constitution: verification through actual execution

**Alternatives considered**:
- **Manual Docker testing only**: Does not scale, not automatable
- **Mock Docker client**: Violates constitution (compilation ≠ correctness)
