# Tasks: Secure Agent Sandbox

**Input**: Design documents from `/specs/001-secure-agent-sandbox/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-contract.md

**Tests**: Constitution development workflow mandates integration tests for the HTTP proxy (secret injection + redaction). Test tasks are included for constitutionally required coverage.

**Organization**: Tasks grouped by user story. US2 depends on US1 (proxy infrastructure). US3 depends on US1+US2 (proxy + injection for redaction).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Go module, install dependencies, create project skeleton and shared types

- [X] T001 Initialize Go module and install dependencies: cobra, goproxy, modernc.org/sqlite, testify, gopkg.in/yaml.v3 in go.mod
- [X] T002 Create project directory structure per plan.md: cmd/codingbox/, internal/{cli,sandbox,proxy,config,store,models}/, tests/{contract,integration,unit}/
- [X] T003 [P] Define shared model types: SandboxSession, SecretMapping, RequestLogEntry, SandboxConfig, Mount in internal/models/session.go, internal/models/secret.go, internal/models/log_entry.go, internal/models/config.go
- [X] T004 [P] Implement sandbox config parsing (codingbox.yml schema) with YAML deserialization and validation in internal/config/sandbox.go
- [X] T005 [P] Implement global config loading (~/.config/codingbox/config.yml) with defaults for db_path, log_retention_days, ca_cert_path, ca_key_path in internal/config/global.go
- [X] T006 [P] Implement CA certificate generation and loading: generate self-signed CA keypair on first run, load existing CA, expose cert/key paths in internal/proxy/ca.go
- [X] T007 Initialize SQLite database with schema migrations: sessions table, request_logs table, indexes per data-model.md in internal/store/db.go
- [X] T008 Implement CLI root command with global flags (--config, --verbose) and version info in cmd/codingbox/main.go and internal/cli/root.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: MicroVM API client and basic proxy server — the two infrastructure pieces ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T009 Implement sandboxd Unix socket HTTP client: POST /vm (create), GET /vm (list), DELETE /vm/{name} (destroy), parse JSON responses (vm_id, socketPath, stateDir, ca_cert_path) in internal/sandbox/client.go
- [X] T010 Implement MITM proxy server lifecycle: start goproxy on configurable port, configure HTTPS MITM with CA from T006, graceful shutdown, return assigned port in internal/proxy/server.go
- [X] T011 Implement basic request/response interceptor skeleton: OnRequest and OnResponse hooks on goproxy, pass-through forwarding (no injection or logging yet), attach session context in internal/proxy/interceptor.go
- [X] T012 Implement session CRUD in SQLite store: Create (with config snapshot), Get, List (with status filter), UpdateStatus (state machine transitions: created→running→stopped, created→failed, running→failed) in internal/store/sessions.go

**Checkpoint**: MicroVM client can create/destroy VMs. Proxy starts and forwards HTTPS traffic. Sessions tracked in SQLite.

---

## Phase 3: User Story 1 — Launch an Isolated Sandbox (Priority: P1) MVP

**Goal**: Developer can launch a sandbox where a coding agent runs isolated from the host, with only explicitly mounted directories accessible, and state persists across restarts

**Independent Test**: Launch a sandbox, verify host paths outside mounts are inaccessible, verify mounted directories work with correct access modes (rw/ro), verify packages persist across restart

### Implementation for User Story 1

- [X] T013 [US1] Implement session orchestrator: coordinate VM creation (via client.go), proxy startup, container launch, session state tracking, and teardown sequence in internal/sandbox/session.go
- [X] T014 [US1] Implement container operations inside microVM: load base image into VM's Docker daemon (docker save/load via VM socket), run agent container with workspace mount, HTTP_PROXY/HTTPS_PROXY env vars pointing to host.docker.internal:{proxy_port}, CA cert volume mount, Docker volume for persistent state in internal/sandbox/container.go
- [X] T015 [US1] Implement `codingbox up` command: parse config, validate, start session orchestrator, stream container output in foreground mode, print session ID in detach mode, handle Ctrl+C for graceful shutdown in internal/cli/up.go
- [X] T016 [US1] Implement `codingbox down` command: look up session by ID or name, trigger graceful shutdown (stop container, destroy VM, stop proxy), update session status, support --force flag in internal/cli/down.go
- [X] T017 [P] [US1] Implement `codingbox ps` command: query sessions from store, display table (session ID, name, status, agent, created_at) or JSON output, support --all flag for stopped sessions in internal/cli/ps.go
- [X] T018 [P] [US1] Implement `codingbox config init` command: generate starter codingbox.yml template, error if file exists, support --output flag in internal/cli/config.go
- [X] T019 [US1] Implement `codingbox config validate` command: load and validate config, check mount paths exist, validate secret mappings have required fields, print errors to stderr in internal/cli/config.go
- [X] T020 [US1] Implement state persistence across restarts: use named Docker volume inside microVM for agent environment (packages, tool configs), re-attach volume on session restart, verify volume survives VM destroy/recreate cycle in internal/sandbox/container.go

**Checkpoint**: `codingbox up` launches an isolated microVM sandbox. `codingbox down` stops it. `codingbox ps` lists sessions. Mounts enforce rw/ro. State persists across restarts. Agent cannot access unmounted host paths.

---

## Phase 4: User Story 2 — Transparent Secret Injection (Priority: P2)

**Goal**: Secrets configured in codingbox.yml are automatically injected into outbound HTTP requests by the proxy. The agent never sees real secret values — only placeholder UUIDs.

**Independent Test**: Configure a secret for api.openai.com, make a request from the sandbox to that host, verify the Authorization header contains the real secret in the upstream request while env vars inside the sandbox show only the placeholder UUID.

**Dependencies**: Requires US1 proxy infrastructure (Phase 2 T010-T011)

### Tests for User Story 2 (constitutionally required)

- [X] T021 [P] [US2] Contract test: proxy injects correct header for matching host, no injection for non-matching host, template expansion works (e.g., "Bearer {secret}") in tests/contract/injection_test.go
- [X] T022 [P] [US2] Contract test: placeholder UUIDs are set as env vars in container, real secret values are not present in container environment in tests/contract/injection_test.go

### Implementation for User Story 2

- [X] T023 [US2] Implement secret injection logic: match request host against SecretMapping.target_host, expand header_template with secret_value, set header on outbound request, track which secrets were injected per request, log warning for misconfigured/missing mappings in internal/proxy/secrets.go
- [X] T024 [US2] Wire secret injection into proxy interceptor: load SecretMapping list from config at proxy startup, call injection logic in OnRequest hook, attach secrets_injected metadata to request context for downstream logging in internal/proxy/interceptor.go
- [X] T025 [US2] Inject placeholder UUIDs as environment variables in agent container: for each SecretMapping, set env var {NAME}_PLACEHOLDER={uuid} inside the container so the agent can reference secrets by placeholder without seeing real values in internal/sandbox/container.go

**Checkpoint**: Secrets are injected into outbound requests per host. Agent sees only placeholder UUIDs. Non-matching hosts pass through unmodified. Missing mappings produce warnings, not failures.

---

## Phase 5: User Story 3 — Full Request Observability (Priority: P3)

**Goal**: Every HTTP request/response through the proxy is logged to SQLite with full metadata. Secrets are redacted in logs. Operators can query logs by session, host, status, and time range.

**Independent Test**: Run a sandbox session with several HTTP requests (including to hosts with secret injection), then query logs via `codingbox logs` and verify all requests appear with complete metadata and redacted secrets.

**Dependencies**: Requires US1 proxy + US2 secret injection (for redaction)

### Tests for User Story 3 (constitutionally required)

- [X] T026 [P] [US3] Contract test: proxy logs all required fields (method, URL, headers, status, latency, timestamp, session_id) for every request in tests/contract/logging_test.go
- [X] T027 [P] [US3] Contract test: secret values in logged headers are replaced with placeholder UUIDs before persistence, raw secrets never appear in SQLite in tests/contract/redaction_test.go

### Implementation for User Story 3

- [X] T028 [US3] Implement request log CRUD and query methods: Insert log entry, query by session_id with filters (host, status, since/until, limit), return results sorted by timestamp in internal/store/logs.go
- [X] T029 [US3] Implement proxy request logger: capture method, URL, host, request headers, request body, response status, response headers, response body, latency (start/end timing), error details for failed requests, write to store via logs.go in internal/proxy/logger.go
- [X] T030 [US3] Implement secret redaction in logger: before persisting, scan all header values against loaded SecretMapping.secret_value entries, replace matches with corresponding placeholder UUID, populate secrets_injected array with names of injected secrets in internal/proxy/logger.go
- [X] T031 [US3] Wire logger into proxy interceptor: call logger in OnResponse hook (or OnError for failed requests), pass session_id from context, ensure logging does not block proxy forwarding (async write or buffered channel) in internal/proxy/interceptor.go
- [X] T032 [US3] Implement `codingbox logs` command: accept session-id positional arg, support --host, --status, --since, --until, --format (table/json), --limit flags, query store, render table output (timestamp, method, URL, status, latency, secrets) or JSON output in internal/cli/logs.go

**Checkpoint**: All HTTP requests are logged with full metadata. Secrets are redacted. `codingbox logs` provides queryable access with filters. Failed requests include error context.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, hardening, and cross-story improvements

- [X] T033 Handle network connectivity loss: proxy logs connection failures with error context, agent receives clear network error (not silent hang) in internal/proxy/logger.go and internal/proxy/interceptor.go
- [X] T034 Handle concurrent sandbox sessions: verify each session gets independent proxy instance on unique port, session IDs are independent ULIDs, logs are correlated by session_id with no cross-contamination in internal/sandbox/session.go
- [X] T035 [P] Handle disk space exhaustion: detect low disk space on stateDir, surface clear error to operator with cleanup guidance via stderr in internal/sandbox/session.go
- [X] T036 [P] Implement log retention: delete request_logs older than log_retention_days on session creation or via periodic cleanup, configurable in global config in internal/store/logs.go
- [X] T037 [P] Add structured application logging via log/slog: consistent JSON log output for codingbox's own operations (not proxy request logs), log levels configurable via --verbose flag in internal/cli/root.go
- [X] T038 Validate quickstart.md end-to-end: walk through the quickstart flow (config init → edit config → up → make requests → logs → down), verify all commands work as documented

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001-T008) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational (T009-T012)
- **US2 (Phase 4)**: Depends on US1 proxy being operational (T010-T011, T013-T014)
- **US3 (Phase 5)**: Depends on US2 secret injection (T023-T024, for redaction logic)
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational — no other story dependencies
- **US2 (P2)**: Depends on US1's proxy and container setup (T010-T011, T014) — extends the interceptor with injection
- **US3 (P3)**: Depends on US2's injection (T023-T024) — needs secret values to implement redaction

### Within Each User Story

- Models/types before services
- Services before CLI commands
- Core logic before integration wiring
- Tests can be written in parallel with or before implementation

### Parallel Opportunities

**Phase 1** (after T001-T002):
- T003, T004, T005, T006 can all run in parallel (different files, no deps)

**Phase 2**:
- T009 and T010 can run in parallel (sandbox client vs proxy server)

**Phase 3** (US1):
- T017 and T018 can run in parallel (ps command vs config init)

**Phase 4** (US2):
- T021 and T022 can run in parallel (test files)

**Phase 5** (US3):
- T026 and T027 can run in parallel (test files)

---

## Parallel Example: Phase 1 Setup

```bash
# After T001 (go mod init) and T002 (directory structure):
# Launch these 4 tasks in parallel:
Task: "Define shared model types in internal/models/*.go"
Task: "Implement sandbox config parsing in internal/config/sandbox.go"
Task: "Implement global config loading in internal/config/global.go"
Task: "Implement CA certificate generation in internal/proxy/ca.go"
```

## Parallel Example: User Story 2

```bash
# Launch contract tests in parallel before implementation:
Task: "Contract test: injection logic in tests/contract/injection_test.go"
Task: "Contract test: placeholder UUIDs in tests/contract/injection_test.go"
# Then implement sequentially: T023 → T024 → T025
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T008)
2. Complete Phase 2: Foundational (T009-T012)
3. Complete Phase 3: User Story 1 (T013-T020)
4. **STOP and VALIDATE**: Launch a sandbox, verify isolation, test mounts, test persistence
5. Deliver MVP — developers can run agents in isolated sandboxes

### Incremental Delivery

1. Setup + Foundational → Infrastructure ready
2. Add US1 → Isolated sandbox works → **MVP**
3. Add US2 → Secrets injected transparently → agents can call authenticated APIs
4. Add US3 → Full audit trail → operators have observability
5. Polish → Edge cases handled, production-ready

### Single-Developer Strategy

Work sequentially through phases. Each phase builds on the previous:
1. Phase 1 + 2: Foundation (~T001-T012)
2. Phase 3: US1 MVP (~T013-T020) → validate
3. Phase 4: US2 secrets (~T021-T025) → validate
4. Phase 5: US3 observability (~T026-T032) → validate
5. Phase 6: Polish (~T033-T038)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [USn] label maps task to specific user story
- Constitution requires proxy integration tests (injection + redaction) — included in US2/US3
- Secret values MUST never appear in SQLite, logs, or sandbox-accessible storage (FR-011)
- All file paths are relative to repository root (/workspace/)
- Commit after each task or logical group
