# Data Model: Container Sandbox

**Date**: 2026-03-23
**Feature**: 002-container-sandbox

## Entities

### SandboxConfig

The user-provided configuration for a sandbox session. Loaded from YAML config file, with CLI flag overrides applied.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| image | string | yes | OCI image reference (e.g. `codingbox/claude:latest`) |
| workdir | string | no | Host directory to mount as working directory (default: current directory) |
| mounts | []MountConfig | no | Additional directory mounts |
| secrets | []SecretMapping | no | Secret placeholder-to-value mappings |
| proxy_port | int | no | Port for the MITM proxy to listen on (default: auto-assigned) |

### MountConfig

A single additional directory mount configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| source | string | yes | Absolute path on the host |
| target | string | yes | Path inside the container |
| mode | enum(ro, rw) | no | Access mode: `ro` (read-only) or `rw` (read-write). Default: `ro` |

### SecretMapping

A mapping between a placeholder visible inside the sandbox and the real secret value known only to the proxy.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| placeholder | string | yes | The placeholder string used inside the sandbox (e.g. `__GITHUB_TOKEN__`) |
| value | string | yes | The real secret value (never enters the sandbox) |
| replace_in | []enum(headers, body, query) | no | Where to perform replacement. Default: `[headers, body, query]` |

### Sandbox

Runtime state of a running sandbox session. Not persisted to config — exists only in memory during execution.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique session identifier (UUID) |
| container_id | string | Docker container ID |
| network_id | string | Docker network ID (isolated bridge) |
| proxy_addr | string | Address of the MITM proxy (host:port) |
| config | SandboxConfig | The resolved configuration for this session |
| state | enum(creating, running, stopping, stopped) | Current lifecycle state |
| created_at | timestamp | When the sandbox was created |

**State transitions**: `creating → running → stopping → stopped`

- `creating`: Network created, proxy starting, container starting
- `running`: Container attached with interactive TTY, proxy active
- `stopping`: User exited or signal received, cleanup in progress
- `stopped`: All resources cleaned up (container removed, network removed, proxy stopped)

### TrafficLog

A single logged HTTP request-response pair. Persisted to SQLite.

| Field | Type | Description |
|-------|------|-------------|
| id | integer | Auto-incrementing primary key |
| sandbox_id | string | Which sandbox session this belongs to |
| timestamp | datetime | When the request was made |
| method | string | HTTP method (GET, POST, etc.) |
| url | string | Full request URL |
| request_headers | text | JSON-encoded request headers |
| request_body | blob | Raw request body (may be large) |
| response_status | integer | HTTP response status code |
| response_headers | text | JSON-encoded response headers |
| response_body | blob | Raw response body (may be large) |
| secrets_replaced | boolean | Whether secret injection was performed on this request |
| duration_ms | integer | Round-trip time in milliseconds |

**Indexes**:
- `idx_traffic_sandbox_id` on `sandbox_id`
- `idx_traffic_timestamp` on `timestamp`
- `idx_traffic_url` on `url`

### CACertificate

The generated CA certificate used for TLS interception. Persisted to filesystem (`~/.codingbox/ca/`).

| Field | Type | Description |
|-------|------|-------------|
| cert_path | string | Path to CA certificate PEM file |
| key_path | string | Path to CA private key PEM file |
| created_at | timestamp | When the CA was generated |

## Relationships

```text
SandboxConfig 1──* MountConfig      (config has zero or more mounts)
SandboxConfig 1──* SecretMapping     (config has zero or more secrets)
Sandbox       1──1 SandboxConfig     (runtime session has one config)
Sandbox       1──* TrafficLog        (session produces zero or more logs)
CACertificate 1──* Sandbox           (one CA cert shared across sessions)
```
