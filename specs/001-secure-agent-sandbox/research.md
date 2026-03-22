# Research: Secure Agent Sandbox

**Date**: 2026-03-22
**Status**: Complete
**Source**: Rivet blog post (Docker microVM reverse engineering), domain knowledge

## R-001: Sandbox Technology — Docker MicroVM API

**Decision**: Use Docker Desktop's undocumented microVM API via `sandboxd.sock`

**Rationale**: MicroVMs provide kernel-level isolation (separate kernel per sandbox) vs containers' shared-kernel model. This is the strongest isolation boundary available through Docker Desktop without custom VM orchestration. The API is simple (3 endpoints) and returns per-VM Docker daemon sockets, allowing us to run containers inside the VM.

**Alternatives considered**:
- **Standard Docker containers**: Rejected — shared kernel means a container escape gives full host access. Insufficient for running untrusted agent code.
- **Firecracker/Cloud Hypervisor directly**: Rejected — requires custom VM orchestration, image management, and networking. Violates constitution principle V (Simplicity).
- **gVisor**: Rejected — adds syscall filtering overhead, doesn't provide full kernel isolation, complex configuration.

**Key API details** (from Rivet reverse engineering):
- Daemon socket: `~/.docker/sandboxes/sandboxd.sock`
- `POST /vm` — create VM (params: `agent_name`, `workspace_dir`)
- `GET /vm` — list VMs
- `DELETE /vm/{vm_name}` — destroy VM
- Response includes: `vm_id`, `socketPath` (per-VM Docker daemon), `fileSharingDirectories`, `stateDir`, `ca_cert_path`
- Each VM gets its own Docker daemon → run containers inside via `docker --host unix://$VM_SOCK`

**Limitations**:
- macOS and Windows only (Docker Desktop required, no Linux support)
- API is undocumented and may change between Docker Desktop versions
- Agent whitelist exists for `docker sandbox run` but raw API (`POST /vm`) is unrestricted
- Must load container images into VM's Docker daemon via `docker save | docker --host load`

## R-002: Networking & Proxy Architecture

**Decision**: Run our own MITM proxy on the host; override HTTP_PROXY/HTTPS_PROXY inside microVM containers to point to our proxy

**Rationale**: Docker microVMs already route traffic through a proxy at `host.docker.internal:3128`. We replace this with our own proxy running on the host, giving us full control over secret injection, logging, and request modification. The microVM's `host.docker.internal` DNS resolution provides a stable reference to the host.

**Architecture**:
```
[Agent in MicroVM Container]
    → HTTP_PROXY=http://host.docker.internal:{our_port}
    → HTTPS_PROXY=http://host.docker.internal:{our_port}
        → [Our MITM Proxy on Host]
            → Secret injection per host
            → Request/response logging
            → Forward to internet
```

**Alternatives considered**:
- **Proxy inside the microVM**: Rejected — secrets would exist inside the isolation boundary, violating FR-005 and constitution principle III.
- **Chain behind Docker's built-in proxy**: Rejected — adds unnecessary latency, Docker's proxy may change behavior, and we'd need to parse/modify already-proxied traffic.
- **iptables-based transparent proxy**: Rejected — requires root in the VM, complex routing, and doesn't work cleanly with microVM networking model.

**HTTPS interception approach**:
- Generate our own CA certificate at first run
- Install CA cert into microVM containers' trust store (via volume mount or Dockerfile)
- MITM proxy generates per-host certificates signed by our CA on-the-fly
- The microVM API returns a `ca_cert_path` for Docker's own CA — we supplement with ours

**Open question resolved**: MicroVM containers can make arbitrary outbound HTTP/HTTPS requests — the proxy at `host.docker.internal:3128` is a forwarding proxy, not a firewall. By overriding HTTP_PROXY/HTTPS_PROXY env vars when running containers inside the VM, all traffic routes through our proxy instead.

## R-003: Programming Language

**Decision**: Go 1.22+

**Rationale**: Go provides the best combination of: single-binary distribution (critical for CLI tool), mature MITM proxy libraries, native Docker client SDK, excellent concurrency model for handling multiple sandbox sessions, and fast HTTP proxy performance (<50ms injection target).

**Key dependencies**:
- **MITM Proxy**: `github.com/elazarl/goproxy` — mature, widely used, supports HTTPS MITM with dynamic cert generation, request/response modification hooks
- **Docker Client**: `github.com/docker/docker/client` — official SDK, Unix socket support built-in
- **SQLite**: `modernc.org/sqlite` (pure Go, no CGO) or `github.com/mattn/go-sqlite3` (CGO, faster)
- **CLI**: `github.com/spf13/cobra` — standard Go CLI framework
- **Config**: `github.com/spf13/viper` or raw YAML (`gopkg.in/yaml.v3`)
- **Logging**: `log/slog` (stdlib structured logging)

**Alternatives considered**:
- **Python + mitmproxy**: mitmproxy is the most mature MITM tool, but Python distribution is complex (requires runtime), and performance overhead is higher for the proxy layer. Good for scripting/extending but wrong for a CLI tool that needs single-binary distribution.
- **TypeScript/Node.js**: `http-mitm-proxy` exists but is less mature for HTTPS interception. Distribution via npm is acceptable but not as clean as a single binary.
- **Rust**: `hudsucker` crate for MITM, excellent performance, but smaller ecosystem for Docker interaction and higher development complexity. Overkill for this use case.

## R-004: Storage

**Decision**: SQLite with JSON columns

**Rationale**: Constitution requires "queryable backend (database, not flat files)" for logs. SQLite is the simplest database that satisfies this: zero configuration, single-file storage, JSON functions for querying structured fields, adequate performance for a developer tool (not a production server). Aligns with constitution principle V (Simplicity).

**Schema approach**: Single `request_logs` table with indexed columns (session_id, timestamp, url, status_code) plus JSON columns for headers and body. Separate `sessions` table for session metadata.

**Alternatives considered**:
- **PostgreSQL**: Rejected — requires running a separate database server, violates simplicity principle for a developer tool.
- **Flat JSON files**: Rejected — constitution explicitly forbids flat files for log storage. Not queryable.
- **BadgerDB/BoltDB**: Rejected — key-value stores don't provide SQL-level querying needed for the log viewer (filter by host, status code, time range).

## R-005: State Persistence for MicroVMs

**Decision**: Use Docker volumes within the microVM's Docker daemon for persistent state

**Rationale**: Each microVM has its own Docker daemon. Containers running inside can use Docker volumes managed by that daemon. The microVM API's `stateDir` provides the persistence directory on the host side. Combined with `workspace_dir` file sharing, we get: workspace files synced bidirectionally + Docker volumes for installed packages/tools.

**Persistence strategy**:
1. Workspace directory: synced via microVM's built-in file sharing (`workspace_dir` parameter)
2. Agent environment (packages, tools): Docker volume attached to the agent container inside the VM
3. Sandbox configuration: YAML file on host, version-controlled by operator

## R-006: Testing Strategy

**Decision**: Go standard `testing` package + `testify` for assertions + `testcontainers-go` for integration tests

**Rationale**: Standard Go testing for unit tests. Integration tests need real Docker interaction — `testcontainers-go` manages container lifecycle in tests. Contract tests verify proxy behavior (secret injection, logging) using httptest servers.

**Test categories**:
- **Unit**: Proxy request modification, secret redaction, config parsing, log storage queries
- **Contract**: Proxy injects secrets correctly, proxy logs all fields, proxy redacts secrets in logs
- **Integration**: Full sandbox lifecycle (create VM, run container, make HTTP request through proxy, verify log), requires Docker Desktop with microVM support
