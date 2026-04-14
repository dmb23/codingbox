# Tasks: Env-Based Secrets and Central Configuration

**Input**: Design documents from `/specs/003-env-secrets-central-config/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included per the constitution (Verify Before Assuming Success).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No new project setup needed — this feature modifies the existing codebase.

- [x] T001 Create new source files with package declarations: `internal/config/central.go`, `internal/config/placeholder.go`, `internal/cli/config_cmd.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared infrastructure needed by multiple user stories

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T002 Add `Env` field to `SecretMapping` struct in `internal/models/config.go`: add `Env string` with yaml/mapstructure tags, keeping existing fields for backwards compatibility
- [x] T003 Implement placeholder generation in `internal/config/placeholder.go`: function `GeneratePlaceholder(envName string) string` that returns `__CODINGBOX_<ENV_NAME>_<sha256[:8]>__`; must be deterministic (same input = same output)
- [x] T004 [P] Write unit test in `tests/unit/placeholder_test.go`: verify deterministic output, uniqueness for different env names, format matches `__CODINGBOX_*__` pattern
- [x] T005 Implement env secret resolution in `internal/config/config.go`: new function `ResolveEnvSecrets(cfg *models.SandboxConfig) error` that for each secret with `Env` set: reads value from host env (or uses explicit `Value`), generates placeholder via `GeneratePlaceholder()`, populates `Placeholder` and `Value` fields; errors if host env var not set and no explicit `Value`
- [x] T006 Update `Validate()` in `internal/config/config.go`: accept secrets with either `Env` set or `Placeholder`+`Value` set (not both, not neither); error on duplicate env var names
- [x] T007 [P] Write unit test in `tests/unit/env_secret_test.go`: verify env resolution reads from host env, uses explicit value override, errors on missing host env, rejects secret with both `env` and `placeholder` set, rejects secret with neither
- [x] T008 Verify foundational changes: run `go build ./cmd/codingbox/` and `go test ./tests/unit/` — all existing + new tests pass

**Checkpoint**: Model updated, placeholder generation works, env secrets resolve correctly

---

## Phase 3: User Story 1 - Environment Variable-Based Secrets (Priority: P1) MVP

**Goal**: Agent uses standard env vars inside the sandbox; proxy transparently replaces placeholders with real secrets

**Independent Test**: Set `ANTHROPIC_API_KEY` on host, configure env secret, launch sandbox, verify env var is placeholder inside, make HTTP request with it, confirm proxy replaced with real value

### Implementation for User Story 1

- [x] T009 [US1] Integrate env secret resolution into `codingbox run` flow in `internal/cli/run.go`: call `config.ResolveEnvSecrets(cfg)` after loading config and before validation; this populates Placeholder/Value fields for env secrets
- [x] T010 [US1] Update `buildEnv()` in `internal/sandbox/sandbox.go`: for each secret with `Env` set, add `ENV_NAME=placeholder_value` to the container environment variables
- [x] T011 [US1] Add `--env-secret` flag to `codingbox run` in `internal/cli/run.go`: parse `ENV_NAME[:headers,body,query]` format, create SecretMapping with `Env` field set, append to config.Secrets before resolution
- [x] T012 [US1] Update `ParseSecretFlag()` in `internal/config/config.go` to handle new `--env-secret` format: input is `ENV_NAME[:locations]` (no `=value`), creates SecretMapping with `Env` set
- [x] T013 [US1] Verify US1 end-to-end: set `TEST_SECRET=real-value-e2e` in host env, run `codingbox run --image ubuntu:22.04 --env-secret TEST_SECRET`, inside container verify `echo $TEST_SECRET` shows placeholder (not `real-value-e2e`), make HTTP request with that value in a header via curl, check `codingbox logs` shows the real value was sent to the destination, confirm `secrets_replaced: true`
- [x] T014 [US1] Verify backwards compatibility: run `codingbox run` with legacy `placeholder`/`value` secret format, confirm it works identically to before (existing tests still pass, E2E works)

**Checkpoint**: Env secrets work end-to-end — agents use env vars naturally, proxy handles replacement

---

## Phase 4: User Story 2 - Central Per-Directory Configuration (Priority: P2)

**Goal**: `codingbox run` with no arguments works in any registered directory

**Independent Test**: Register config for a directory, cd into it, run `codingbox run` with no args, confirm correct image and settings

### Implementation for User Story 2

- [x] T015 [US2] Implement `DirectoryConfigStore` in `internal/config/central.go`: struct with `Load(path)`, `Save()`, `Get(dir)`, `FindNearest(dir)`, `Set(dir, cfg)`, `Remove(dir)`, `List()` methods; store is a YAML file at `~/.codingbox/directories.yaml`; `FindNearest` walks up from the given dir to root looking for a matching entry
- [x] T016 [US2] Implement canonical path resolution in `internal/config/central.go`: function `CanonicalDir(dir string) (string, error)` that resolves to absolute path with symlinks resolved via `filepath.EvalSymlinks`
- [x] T017 [P] [US2] Write unit test in `tests/unit/central_test.go`: verify Set/Get/Remove/List operations, FindNearest walks up parents, canonical path resolution, Save/Load round-trip to temp YAML file
- [x] T018 [US2] Update config loading in `internal/config/config.go`: modify `Load()` to accept an optional fallback — when no explicit config path and no local `codingbox.yaml`, call `DirectoryConfigStore.FindNearest(cwd)` to look up central config; return helpful error if nothing found
- [x] T019 [US2] Update error message in `Validate()` in `internal/config/config.go`: when image is missing, suggest both `codingbox init` and `codingbox config set --image <image>`
- [x] T020 [US2] Verify US2 end-to-end: create a temp directory, run `codingbox config set --image ubuntu:22.04 --dir <tempdir>` (requires T021-T023 from US3), then run `codingbox run` from that directory with no args, confirm sandbox starts with ubuntu:22.04; also test from a subdirectory to verify parent walking; also test precedence: place a local `codingbox.yaml` with a different image and confirm it takes priority

**Checkpoint**: Zero-arg `codingbox run` works from any registered directory

---

## Phase 5: User Story 3 - Config Management CLI (Priority: P3)

**Goal**: Register, list, update, and remove central configs via CLI

**Independent Test**: Run `codingbox config set`, `config list`, `config set` (update), `config remove`, verify each operation

### Implementation for User Story 3

- [x] T021 [US3] Implement `codingbox config set` command in `internal/cli/config_cmd.go`: parse `--dir`, `--image`, `--mount`, `--env-secret`, `--secret`, `--proxy-port` flags; resolve dir to canonical path; create/update entry in DirectoryConfigStore; print confirmation
- [x] T022 [US3] Implement `codingbox config list` command in `internal/cli/config_cmd.go`: load store, print table of directory → image, secrets count, mounts count
- [x] T023 [US3] Implement `codingbox config remove` command in `internal/cli/config_cmd.go`: parse `--dir` flag (default cwd), resolve to canonical path, remove entry, print confirmation; error if entry not found
- [x] T024 [US3] Register `config` command group with `set`, `list`, `remove` subcommands on the root command in `internal/cli/config_cmd.go`
- [x] T025 [US3] Verify US3 end-to-end: run `codingbox config set --image ubuntu:22.04` from a temp dir, run `codingbox config list` and verify entry appears, run `codingbox config set --image alpine:latest` to update, verify update in list, run `codingbox config remove`, verify entry gone
- [x] T026 [US2] Verify US2 end-to-end (full integration): now that config CLI exists, repeat T020 verification — register config, run sandbox from that dir, verify it works

**Checkpoint**: Full config management workflow works

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T027 [P] Add `--env-secret` flag to `codingbox init` in `internal/cli/init.go`: pre-fill env secret entries in generated config file
- [x] T028 [P] Update README.md: add env-based secrets section showing new `env:` config format and `--env-secret` flag; add central config section showing `codingbox config set/list/remove` workflow
- [x] T029 Run all unit tests: `go test ./tests/unit/` — all existing and new tests pass with zero regressions
- [x] T030 Run full quickstart validation: follow `specs/003-env-secrets-central-config/quickstart.md` from scratch, verify each step works

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — create empty files
- **Foundational (Phase 2)**: Depends on Phase 1 — model changes and placeholder logic
- **US1 (Phase 3)**: Depends on Phase 2 — needs env secret resolution
- **US2 (Phase 4)**: Depends on Phase 2 — needs central config store
- **US3 (Phase 5)**: Depends on Phase 4 T015-T016 — needs store implementation to build CLI
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — independent of US2/US3
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) — needs US3 for full E2E verification
- **User Story 3 (P3)**: Depends on US2 T015-T016 (store implementation) — CLI wraps the store

### Parallel Opportunities

- T003 and T004 can run in parallel (implementation + test in different files)
- T005 and T007 can run in parallel (implementation + test in different files)
- US1 (Phase 3) and US2 T015-T017 can run in parallel after Phase 2
- T027 and T028 can run in parallel (different files)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (model + placeholder + env resolution)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: env secrets work end-to-end, backwards compatible
5. This alone delivers significant value — agents use env vars naturally

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready
2. US1 → Env-based secrets work (MVP)
3. US2 + US3 → Central config works (full feature)
4. Polish → README updated, quickstart validated

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Verification tasks (T008, T013, T014, T020, T025, T026, T029, T030) are constitution-mandated
- This feature modifies existing files — be careful not to break existing functionality
- Run existing tests after every phase to catch regressions early
