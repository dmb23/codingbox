# Tasks: Container Sandbox for Agentic Workloads

**Input**: Design documents from `/specs/002-container-sandbox/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included as verification steps per the constitution (Verify Before Assuming Success).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Project initialization and Go module structure

- [x] T001 Create project directory structure per plan.md: `cmd/codingbox/`, `internal/cli/`, `internal/config/`, `internal/sandbox/`, `internal/proxy/`, `internal/store/`, `internal/models/`, `tests/integration/`, `tests/unit/`
- [x] T002 Initialize Go module (`go mod init`) and add dependencies: `github.com/docker/docker`, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/elazarl/goproxy`, `modernc.org/sqlite`; run `go mod tidy`
- [x] T003 [P] Create `codingbox.yaml.example` at repository root with sample config per contracts/cli-commands.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 [P] Define model structs: SandboxConfig, MountConfig, SecretMapping in `internal/models/config.go` per data-model.md
- [x] T005 [P] Define Sandbox runtime state struct in `internal/models/sandbox.go` per data-model.md (id, container_id, network_id, proxy_addr, state, created_at)
- [x] T006 [P] Define TrafficLog struct in `internal/models/traffic.go` per data-model.md
- [x] T007 Implement config loading in `internal/config/config.go`: load YAML file via viper, validate required fields (image), apply defaults (workdir=`.`, proxy_port=0, mount mode=`ro`, replace_in=`[headers,body,query]`)
- [x] T008 Implement CLI flag override merging in `internal/config/config.go`: parse `--mount` (`source:target[:ro|rw]`) and `--secret` (`placeholder=value[:locations]`) flag formats, merge with config file values
- [x] T009 Create root cobra command in `internal/cli/root.go` and wire to `cmd/codingbox/main.go` entry point; verify the binary builds and `codingbox --help` prints usage

**Checkpoint**: Foundation ready — `go build ./cmd/codingbox/` succeeds, `codingbox --help` prints command structure

---

## Phase 3: User Story 1 - Launch a Coding Agent in a Sandbox (Priority: P1) MVP

**Goal**: Launch an interactive terminal session inside a Docker container with the working directory mounted read-write

**Independent Test**: Run `codingbox run --image ubuntu:22.04`, confirm interactive shell, create a file inside, verify it appears on host

### Implementation for User Story 1

- [x] T010 [US1] Implement Docker network creation in `internal/sandbox/network.go`: create an isolated bridge network (`Internal: true`) with unique name per session, return network ID
- [x] T011 [US1] Implement Docker network cleanup in `internal/sandbox/network.go`: disconnect containers and remove network by ID
- [x] T012 [US1] Implement container creation in `internal/sandbox/sandbox.go`: create container from OCI image with TTY enabled (`Tty: true`, `OpenStdin: true`, `AttachStdin/Stdout/Stderr: true`), mount workdir as read-write bind mount, connect to isolated network
- [x] T013 [US1] Implement interactive TTY attachment in `internal/sandbox/attach.go`: attach to container, set host terminal to raw mode, bidirectional stream copy (stdin→container, container→stdout), restore terminal on exit
- [x] T014 [US1] Implement sandbox lifecycle orchestration in `internal/sandbox/sandbox.go`: `Start()` creates network → creates container → starts container → attaches TTY; `Stop()` detaches → stops container → removes container → removes network
- [x] T015 [US1] Implement signal handling in `internal/sandbox/sandbox.go`: trap SIGINT and SIGTERM, trigger graceful `Stop()` on signal, ensure terminal state is restored
- [x] T016 [US1] Implement `codingbox run` command (basic) in `internal/cli/run.go`: load config, validate image is set, create Sandbox, call `Start()`, wait for session end, call `Stop()`, exit with appropriate code per contracts/cli-commands.md
- [x] T017 [US1] Write unit test in `tests/unit/config_test.go`: verify YAML loading, flag override merging, default values, validation errors for missing image
- [x] T018 [US1] Verify US1 end-to-end: build binary, run `codingbox run --image ubuntu:22.04`, confirm interactive shell starts, create a file in the mounted workdir, exit, verify file exists on host, verify no orphaned containers or networks (`docker ps -a`, `docker network ls`)

**Checkpoint**: User Story 1 fully functional — interactive sandbox with workdir mount and clean cleanup

---

## Phase 4: User Story 2 - Proxy and Log All Outbound Traffic (Priority: P2)

**Goal**: Route all sandbox HTTP/HTTPS traffic through a MITM proxy that logs every request and response to SQLite

**Independent Test**: Launch sandbox, `curl https://httpbin.org/get` from inside, verify request+response appear in `codingbox logs` output

### Implementation for User Story 2

- [x] T019 [P] [US2] Implement CA certificate generation in `internal/proxy/certs.go`: generate self-signed CA cert+key on first run, store in `~/.codingbox/ca/`, load existing CA on subsequent runs, expose CA cert path for container mounting
- [x] T020 [P] [US2] Implement SQLite store initialization in `internal/store/store.go`: open database at `~/.codingbox/traffic.db`, create `traffic_logs` table with schema from data-model.md, set WAL mode + `PRAGMA synchronous = NORMAL`, create indexes
- [x] T021 [US2] Implement traffic log insert and query operations in `internal/store/queries.go`: `InsertLog(TrafficLog)`, `QueryLogs(filters)` with filtering by session, method, URL, status, since, limit
- [x] T022 [US2] Implement MITM proxy setup in `internal/proxy/proxy.go`: create goproxy instance with CA cert, configure HTTPS MITM for all hosts, start HTTP server on configured port (or auto-assign), expose proxy address
- [x] T023 [US2] Implement request/response logging handler in `internal/proxy/logger.go`: goproxy `OnRequest().DoFunc()` captures method, URL, headers, body; `OnResponse().DoFunc()` captures status, headers, body; write TrafficLog to store; measure duration_ms
- [x] T024 [US2] Integrate proxy into sandbox lifecycle in `internal/sandbox/sandbox.go`: `Start()` now starts proxy before container, injects `HTTP_PROXY`/`HTTPS_PROXY` env vars pointing to proxy address, bind-mounts CA cert into container at `/usr/local/share/ca-certificates/codingbox.crt`, sets `SSL_CERT_FILE` and `NODE_EXTRA_CA_CERTS` env vars
- [x] T025 [US2] Implement `codingbox logs` command in `internal/cli/logs.go`: parse filter flags per contracts/cli-commands.md, query store, format output as table (default) or JSON, include `--body` flag for verbose output
- [x] T026 [US2] Implement `codingbox ca show` and `codingbox ca regenerate` subcommands in `internal/cli/ca.go` per contracts/cli-commands.md
- [x] T027 [P] [US2] Write unit test in `tests/unit/store_test.go`: verify SQLite init, log insertion, query filtering by method/URL/status/since/limit
- [x] T028 [US2] Verify US2 end-to-end: build binary, run `codingbox run --image ubuntu:22.04`, inside container run `curl https://httpbin.org/get`, exit, run `codingbox logs`, confirm the request+response are logged with correct URL, status 200, headers, and body; verify `codingbox logs --format json` outputs valid JSON

**Checkpoint**: User Stories 1 AND 2 both work — sandbox with full traffic logging and queryable logs

---

## Phase 5: User Story 3 - Transparent Secret Injection (Priority: P3)

**Goal**: Replace placeholder strings with real secrets in outbound requests and strip secrets from inbound responses, configurable per secret

**Independent Test**: Configure a secret mapping, make a request with the placeholder from inside the sandbox, verify the proxy substituted the real value (visible in logs) while the sandbox only ever saw the placeholder

### Implementation for User Story 3

- [x] T029 [US3] Implement secret replacement logic in `internal/proxy/secrets.go`: given a list of SecretMapping, perform find-and-replace on request headers, body, and/or query parameters based on each secret's `replace_in` config; return modified request and whether any replacement occurred
- [ ] T030 [US3] Implement response reverse-replacement in `internal/proxy/secrets.go`: scan response headers and body for real secret values, replace with corresponding placeholders based on `replace_in` config
- [ ] T031 [US2] Integrate secret handlers into proxy pipeline in `internal/proxy/proxy.go`: add request handler that calls secret replacement before forwarding, add response handler that calls reverse-replacement before returning to container; set `secrets_replaced` flag in TrafficLog
- [ ] T032 [P] [US3] Write unit test in `tests/unit/secrets_test.go`: verify replacement in headers only, body only, query only, all locations, no false positives when placeholder appears in non-configured location, reverse replacement in responses
- [ ] T033 [US3] Verify US3 end-to-end: create config with `secrets: [{placeholder: "__TEST_KEY__", value: "real-secret-123", replace_in: ["headers"]}]`, run sandbox, inside run `curl -H "Authorization: Bearer __TEST_KEY__" https://httpbin.org/headers`, exit, run `codingbox logs --body`, confirm the logged request shows `Authorization: Bearer real-secret-123` (real value in outbound), confirm the container never had access to `real-secret-123`

**Checkpoint**: Secret injection works — agents use placeholders, proxy handles real credentials transparently

---

## Phase 6: User Story 4 - Configure Additional Directory Mounts (Priority: P4)

**Goal**: Mount additional host directories into the container with configurable read-only or read-write access

**Independent Test**: Configure a read-only mount, launch sandbox, verify the directory is readable but not writable inside the container

### Implementation for User Story 4

- [ ] T034 [US4] Implement mount validation in `internal/config/config.go`: verify source paths exist, target paths are absolute, mode is `ro` or `rw`; return clear error for non-existent source directories
- [ ] T035 [US4] Apply additional mounts to container creation in `internal/sandbox/sandbox.go`: iterate MountConfig list, add each as a Docker bind mount with `ReadOnly` set based on mode; append to existing workdir mount
- [ ] T036 [US4] Verify US4 end-to-end: create a temp directory with a test file, configure it as read-only mount in config, run sandbox, verify file is readable inside container (`cat /mnt/test/file.txt`), verify write is rejected (`touch /mnt/test/new.txt` fails), exit; repeat with read-write mount and verify write succeeds and is visible on host

**Checkpoint**: Additional mounts work with correct access control

---

## Phase 7: User Story 5 - Define Sandbox as OCI Image (Priority: P5)

**Goal**: Support any valid OCI image (local or remote) as sandbox definition with clear error handling

**Independent Test**: Build a custom Dockerfile with a specific tool, launch sandbox from that image, verify the tool is available inside

### Implementation for User Story 5

- [ ] T037 [US5] Implement image validation and pull logic in `internal/sandbox/sandbox.go`: check if image exists locally, if not attempt `docker pull`, report progress, return clear error for invalid/missing images with actionable message
- [ ] T038 [US5] Implement `codingbox init` command in `internal/cli/init.go`: generate default `codingbox.yaml` with commented examples per contracts/cli-commands.md, support `--image` pre-fill and `--force` overwrite
- [ ] T039 [US5] Verify US5 end-to-end: create a minimal Dockerfile (`FROM ubuntu:22.04\nRUN apt-get update && apt-get install -y jq`), build it (`docker build -t test-sandbox .`), run `codingbox run --image test-sandbox`, verify `jq --version` works inside; also test with invalid image name and confirm clear error message

**Checkpoint**: Any valid OCI image works as sandbox environment

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T040 [P] Improve error messages across all commands: Docker daemon not running, port already in use, permission denied on mount, image not found — each with actionable guidance per FR-013
- [ ] T041 [P] Harden cleanup in `internal/sandbox/sandbox.go`: ensure cleanup runs even on panic (defer), handle partial state (network created but container failed), add timeout to cleanup operations (5s per SC-006)
- [ ] T042 [P] Add `--proxy-port` CLI flag override to `codingbox run` per contracts/cli-commands.md
- [ ] T043 Run full quickstart.md validation: follow every step in `specs/002-container-sandbox/quickstart.md` from scratch, verify each command works as documented

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - US1 (Phase 3): Can start after Foundational
  - US2 (Phase 4): Depends on US1 (builds on sandbox lifecycle)
  - US3 (Phase 5): Depends on US2 (builds on proxy infrastructure)
  - US4 (Phase 6): Depends on US1 only (mount config is independent of proxy)
  - US5 (Phase 7): Depends on US1 only (image handling is independent of proxy)
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2)
- **User Story 2 (P2)**: Depends on US1 (needs working sandbox lifecycle to add proxy)
- **User Story 3 (P3)**: Depends on US2 (needs proxy pipeline to add secret handlers)
- **User Story 4 (P4)**: Depends on US1 only — can run in parallel with US2/US3
- **User Story 5 (P5)**: Depends on US1 only — can run in parallel with US2/US3

### Within Each User Story

- Models before services
- Services before CLI commands
- Core implementation before integration
- Verification step last (constitution requirement)
- Story complete before moving to next priority

### Parallel Opportunities

- T003, T004, T005, T006 can run in parallel (different files)
- T010, T011 can run in parallel with T012, T013 (network vs container, but same package — sequential recommended)
- T019, T020 can run in parallel (certs vs SQLite — different packages)
- T027, T032 can run in parallel (different test files)
- After US1 complete: US4 and US5 can run in parallel with US2
- T040, T041, T042 can all run in parallel (different concerns)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run `codingbox run --image ubuntu:22.04`, create files, exit, verify cleanup
5. Deploy/demo if ready — basic sandbox works

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → MVP: interactive sandbox
3. Add User Story 2 → Test independently → Traffic logging works
4. Add User Story 3 → Test independently → Secret injection works
5. Add User Story 4 → Test independently → Additional mounts work
6. Add User Story 5 → Test independently → Any OCI image works
7. Polish phase → Production-ready

### Parallel Development Path

After US1 is complete, two tracks can proceed in parallel:

- **Track A**: US2 → US3 (proxy → secrets, sequential dependency)
- **Track B**: US4, US5 (mount config + OCI images, independent of proxy)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verification tasks (T018, T028, T033, T036, T039, T043) are constitution-mandated: run the code, inspect output
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
