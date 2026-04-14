# Tasks: Default Sandbox Image and Config Defaults

**Input**: Design documents from `/specs/004-default-sandbox-image/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included per the constitution (Verify Before Assuming Success).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Create new files and directory structure

- [x] T001 Create new source files with package declarations: `internal/config/automount.go`, `internal/cli/config_default.go`
- [x] T002 [P] Create `docker/` directory and empty `docker/entrypoint.sh`, `Dockerfile`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared infrastructure needed by multiple user stories

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Add `GlobalDefaults` struct to `internal/config/central.go`: add `Defaults` field with `DefaultImage string` to `DirectoryConfigStore`, update YAML marshal/unmarshal to include top-level `defaults:` key alongside `directories:`
- [x] T004 Define the built-in default image constant in `internal/config/automount.go`: `const DefaultSandboxImage = "codingbox/sandbox:latest"`
- [x] T005 Implement auto-mount registry in `internal/config/automount.go`: define a `var AutoMounts []AutoMountEntry` list with all 7 entries from data-model.md (gitconfig ro, config/git ro, .claude rw, .claude.json rw, .vibe rw, config/opencode rw, local/share/opencode rw); each entry has source pattern (relative to `$HOME`), mode, and description
- [x] T006 Implement `ResolveAutoMounts(home string) []models.MountConfig` in `internal/config/automount.go`: for each auto-mount entry, resolve the source to an absolute path using the given home dir, check if the path exists on the host (`os.Stat`), skip silently if absent, return list of MountConfig for existing paths; target = source (same path)
- [x] T007 [P] Write unit test in `tests/unit/automount_test.go`: create temp dirs mimicking `~/.gitconfig`, `~/.claude/`, etc., call `ResolveAutoMounts` with that temp dir as home, verify only existing paths are returned, verify modes are correct, verify target == source
- [x] T008 Verify foundational changes: run `go build ./cmd/codingbox/` and `go test ./tests/unit/` — all existing + new tests pass

**Checkpoint**: Auto-mount registry works, GlobalDefaults struct in central store

---

## Phase 3: User Story 1 - Default Sandbox Image (Priority: P1) MVP

**Goal**: `codingbox run` works with zero configuration using the default sandbox image with all agents pre-installed

**Independent Test**: Run `codingbox run` with no config, verify it uses `codingbox/sandbox:latest` and agents are available inside

### Implementation for User Story 1

- [x] T009 [US1] Create `Dockerfile` at repository root: Ubuntu 24.04 base, install Node.js 22 LTS (via nodesource), Python 3.12+, Go (latest stable via official tarball), neovim, git, curl, wget, jq, ripgrep, fd-find, build-essential, zsh; install Claude Code (`npm install -g @anthropic-ai/claude-code`), Mistral Vibe (`curl -LsSf https://mistral.ai/vibe/install.sh | bash`), OpenCode (`curl -fsSL https://raw.githubusercontent.com/opencode-ai/opencode/refs/heads/main/install | bash`); set default shell to bash
- [x] T010 [US1] Create `docker/entrypoint.sh`: read `CODINGBOX_UID` and `CODINGBOX_GID` env vars (default 1000), create group and user with matching IDs if they don't exist, set `HOME` to match `CODINGBOX_HOME` env var (default `/home/codingbox`), exec the command as that user via `exec gosu $USERNAME "$@"`; install `gosu` in the Dockerfile
- [x] T011 [US1] Update `internal/config/config.go`: in the image resolution logic, when image is still empty after all config sources are checked, apply fallback: first check `GlobalDefaults.DefaultImage` from central store, then fall back to `DefaultSandboxImage` constant
- [x] T012 [US1] Update `internal/sandbox/sandbox.go`: in `Start()`, pass `CODINGBOX_UID=$(id -u)`, `CODINGBOX_GID=$(id -g)`, and `CODINGBOX_HOME=$HOME` as env vars to the container so the entrypoint can match the host user
- [x] T013 [US1] Build the default image locally: run `docker build -t codingbox/sandbox:latest .` and verify it completes under 2 GB compressed
- [x] T014 [US1] Verify US1 end-to-end: run `codingbox run` with no config or flags (ensure no codingbox.yaml in test dir and no central config), verify it uses `codingbox/sandbox:latest`, verify `claude --version`, `opencode --version`, `nvim --version` are available inside, verify `git config user.name` returns the host user's identity

**Checkpoint**: Zero-config `codingbox run` works with all agents available

---

## Phase 4: User Story 2 - Auto-Mount Config Directories (Priority: P2)

**Goal**: Agent config directories and git config are automatically mounted from host at the same paths

**Independent Test**: Run `codingbox run` without mount flags, verify `~/.gitconfig`, `~/.claude/`, `~/.vibe/` from host are accessible at the same paths inside the container

### Implementation for User Story 2

- [x] T015 [US2] Add `--no-auto-mounts` flag to `codingbox run` in `internal/cli/run.go`
- [x] T016 [US2] Update `internal/sandbox/sandbox.go` `Start()` method: before building the mount list, call `ResolveAutoMounts(os.Getenv("HOME"))` to get auto-mounts; merge with explicit mounts (explicit mounts take precedence for same target path); skip auto-mounts if `--no-auto-mounts` flag is set
- [x] T017 [US2] Pass `--no-auto-mounts` flag value from `internal/cli/run.go` to the sandbox manager: either via the config struct (add `NoAutoMounts bool` to SandboxConfig) or as a parameter to `Start()`
- [x] T018 [US2] Verify US2 end-to-end: run `codingbox run`, inside container check `cat ~/.gitconfig` (should match host), check `ls ~/.claude/` (should exist if it does on host), check `ls ~/.vibe/` (should exist if it does on host); also test `--no-auto-mounts` flag — verify those paths are NOT mounted; also test that a path not present on host is silently skipped

**Checkpoint**: Auto-mounts work for all 7 config paths, `--no-auto-mounts` disables them

---

## Phase 5: User Story 3 - Global Default Image Config (Priority: P3)

**Goal**: Developer can configure a custom default image globally

**Independent Test**: Run `codingbox config set-default --image my-custom:latest`, then run `codingbox run` with no config — verify it uses `my-custom:latest`

### Implementation for User Story 3

- [x] T019 [US3] Implement `codingbox config set-default` command in `internal/cli/config_default.go`: parse `--image` flag, load central store, update `Defaults.DefaultImage`, save store, print confirmation
- [x] T020 [US3] Implement `codingbox config show-default` command in `internal/cli/config_default.go`: load central store, print current default image (or built-in default if not configured)
- [x] T021 [US3] Register `set-default` and `show-default` as subcommands of `config` in `internal/cli/config_default.go`
- [x] T022 [US3] Verify US3 end-to-end: run `codingbox config set-default --image alpine:latest`, run `codingbox config show-default` and verify it shows `alpine:latest`, run `codingbox run` (no config) and verify it uses `alpine:latest` not the built-in default; then reset with `codingbox config set-default --image codingbox/sandbox:latest`

**Checkpoint**: Custom global default image works

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final validation

- [x] T023 [P] Update `README.md`: add "Default Sandbox Image" section documenting what's included, auto-mounts behavior, `--no-auto-mounts` flag, `config set-default`/`show-default` commands
- [x] T024 [P] Update `codingbox.yaml.example`: add comments showing auto-mount behavior and default image
- [x] T025 Run all unit tests: `go test ./tests/unit/` — all existing and new tests pass with zero regressions
- [x] T026 Run full quickstart validation: follow `specs/004-default-sandbox-image/quickstart.md` from scratch, verify each step works

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — create empty files
- **Foundational (Phase 2)**: Depends on Phase 1 — auto-mount registry and global defaults
- **US1 (Phase 3)**: Depends on Phase 2 — needs default image constant and image fallback logic
- **US2 (Phase 4)**: Depends on Phase 2 — needs auto-mount resolution
- **US3 (Phase 5)**: Depends on Phase 2 T003 — needs GlobalDefaults in central store
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2)
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) — independent of US1
- **User Story 3 (P3)**: Can start after Phase 2 T003 — independent of US1/US2

### Parallel Opportunities

- T001 and T002 can run in parallel (different files)
- US1 (Dockerfile) and US2 (auto-mounts) can proceed in parallel after Phase 2
- US3 can proceed in parallel with US1/US2 after T003
- T023 and T024 can run in parallel (different files)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 + Phase 2
2. Complete Phase 3: Build default image, add image fallback
3. **STOP and VALIDATE**: `codingbox run` works with no config, agents available
4. This alone delivers the core value — a working default sandbox

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready
2. US1 → Default image works (MVP)
3. US2 → Auto-mounts work (seamless config)
4. US3 → Custom default image (full flexibility)
5. Polish → Docs updated, quickstart validated

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- The Dockerfile (T009) is the largest single task — it may take time to iterate on image size and agent installations
- Verification tasks (T008, T013, T014, T018, T022, T025, T026) are constitution-mandated
- Agent installation commands may change — verify URLs/methods are current before running
