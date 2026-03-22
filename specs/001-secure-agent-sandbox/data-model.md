# Data Model: Secure Agent Sandbox

**Date**: 2026-03-22
**Source**: spec.md key entities + research.md decisions

## Entities

### SandboxSession

Represents a single execution lifecycle of a sandbox environment.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT (ULID) | PK, NOT NULL | Unique session identifier (sortable by time) |
| vm_id | TEXT | NOT NULL | Docker microVM identifier returned by sandboxd |
| vm_socket_path | TEXT | NOT NULL | Path to per-VM Docker daemon socket |
| agent_name | TEXT | NOT NULL | Name of the agent running in this session |
| status | TEXT | NOT NULL, CHECK(status IN ('created','running','stopped','failed')) | Current lifecycle state |
| config_snapshot | JSON | NOT NULL | Frozen copy of SandboxConfig at session creation time |
| created_at | TEXT (ISO8601) | NOT NULL | Session creation timestamp |
| started_at | TEXT (ISO8601) | NULLABLE | When the agent container started |
| stopped_at | TEXT (ISO8601) | NULLABLE | When the session ended |
| error_message | TEXT | NULLABLE | Populated on status='failed' |

**State transitions**: `created` → `running` → `stopped` | `created` → `failed` | `running` → `failed`

**Indexes**: `status`, `created_at`, `agent_name`

---

### SecretMapping

Configuration entry mapping a placeholder to a real secret for proxy injection.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT (UUID) | PK, NOT NULL | Placeholder UUID exposed inside sandbox |
| name | TEXT | NOT NULL, UNIQUE | Human-readable name (e.g., "openai-api-key") |
| target_host | TEXT | NOT NULL | Host to match (e.g., "api.openai.com") |
| header_name | TEXT | NOT NULL | HTTP header to inject (e.g., "Authorization") |
| header_template | TEXT | NOT NULL | Template with `{secret}` placeholder (e.g., "Bearer {secret}") |
| secret_value | TEXT | NOT NULL | Actual secret — stored in host-side config only, never in DB |
| created_at | TEXT (ISO8601) | NOT NULL | When mapping was created |

**Note**: `secret_value` exists only in the host-side YAML config file. It is loaded into memory at runtime by the proxy. It is NEVER written to SQLite, logs, or any sandbox-accessible storage (FR-011).

**Indexes**: `target_host` (for proxy lookup per request)

---

### RequestLogEntry

Record of a single HTTP request/response pair captured by the MITM proxy.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT (ULID) | PK, NOT NULL | Unique log entry identifier |
| session_id | TEXT | NOT NULL, FK→SandboxSession.id | Owning session |
| method | TEXT | NOT NULL | HTTP method (GET, POST, etc.) |
| url | TEXT | NOT NULL | Full request URL |
| host | TEXT | NOT NULL | Extracted hostname (for filtering) |
| request_headers | JSON | NOT NULL | Request headers (secrets redacted to placeholder UUIDs) |
| request_body | BLOB | NULLABLE | Request body (may be large; stored as-is) |
| response_status | INTEGER | NULLABLE | HTTP status code (null if connection failed) |
| response_headers | JSON | NULLABLE | Response headers |
| response_body | BLOB | NULLABLE | Response body |
| latency_ms | INTEGER | NULLABLE | Round-trip time in milliseconds |
| error | TEXT | NULLABLE | Error message if request failed (network error, timeout) |
| secrets_injected | JSON | NULLABLE | Array of secret names injected (for audit trail, not values) |
| timestamp | TEXT (ISO8601) | NOT NULL | When the request was initiated |

**Indexes**: `session_id`, `timestamp`, `host`, `response_status`

**Redaction rule**: Before persisting, all header values matching any `SecretMapping.secret_value` are replaced with the corresponding placeholder UUID.

---

### SandboxConfig

Declarative specification for a sandbox environment. Stored as a YAML file on the host, not in the database. A frozen snapshot is stored in `SandboxSession.config_snapshot` at session creation.

| Field | Type | Description |
|-------|------|-------------|
| name | string | Configuration name |
| agent | string | Agent identifier (e.g., "claude", "codex") |
| workspace_dir | string | Host path to mount as workspace (read-write) |
| mounts | []Mount | Additional directory mounts |
| secrets | []SecretMapping | Secret injection mappings |
| base_image | string | Docker image for agent container (optional, default provided) |
| tools | []string | Pre-installed tools/runtimes (e.g., ["node:20", "python:3.12"]) |
| proxy_port | integer | Host port for MITM proxy (default: auto-assigned) |

**Mount sub-type**:

| Field | Type | Description |
|-------|------|-------------|
| host_path | string | Absolute path on host |
| sandbox_path | string | Path inside sandbox |
| mode | string | "rw" or "ro" |

## Relationships

```
SandboxConfig (YAML file, host-side)
    ├── has many → SecretMapping (in config, loaded at runtime)
    └── has many → Mount

SandboxSession
    ├── snapshots → SandboxConfig (frozen at creation in config_snapshot)
    └── has many → RequestLogEntry (via session_id FK)
```

## Storage Boundaries

| Data | Location | Accessible from sandbox? |
|------|----------|------------------------|
| SandboxConfig (YAML) | Host filesystem | NO |
| SecretMapping.secret_value | Host memory (loaded from YAML) | NO |
| SandboxSession | SQLite on host | NO |
| RequestLogEntry | SQLite on host | NO |
| Placeholder UUIDs | Env vars in sandbox | YES (by design) |
| Workspace files | Synced volume | YES (by design) |
