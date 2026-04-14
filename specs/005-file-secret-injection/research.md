# Research: File-Based Secret Injection

**Date**: 2026-04-14
**Feature**: 005-file-secret-injection

## R1: Claude Code Auth Token Storage

**Finding**: On Linux (no Keychain), Claude Code stores OAuth credentials at `~/.claude/.credentials.json`.

**File format**:
```json
{
  "claudeAiOauth": {
    "accessToken": "sk-ant-oat01-...",
    "refreshToken": "sk-ant-ort01-...",
    "expiresAt": 1748658860401,
    "scopes": ["user:inference", "user:profile"]
  }
}
```

**Key insight**: The token auto-refreshes. When the `accessToken` expires, Claude Code uses the `refreshToken` to get a new one and writes back to the file. Since our overlay mounts the file read-only, the write-back will fail. This means:
- The `accessToken` works until it expires (~6 hours)
- After expiry, Claude Code can't refresh and auth fails
- For long sessions, the user would need to restart the sandbox

**Alternative approach**: Claude Code also supports `CLAUDE_CODE_OAUTH_TOKEN` env var for headless use. A long-lived OAuth token (1 year validity) can be generated via `claude setup-token` on the host. This would work better with our existing env secret mechanism.

**Decision**: Support both approaches:
1. File secret for `~/.claude/.credentials.json` with `json_key: claudeAiOauth.accessToken` (works for short sessions)
2. Document that for long sessions, `CLAUDE_CODE_OAUTH_TOKEN` via env secret is preferred

## R2: Nested JSON Key Support

**Decision**: Support dot-notation for JSON keys (e.g., `claudeAiOauth.accessToken`).

**Rationale**: Claude Code's credentials file has the token nested under `claudeAiOauth.accessToken`. Top-level-only support would require the user to specify the entire `claudeAiOauth` object, which doesn't work for our string-replacement approach. Dot-notation is simple to implement and covers the real use case.

## R3: Docker Split-Mount Behavior

**Finding**: Docker supports mounting a file on top of a path within an already-mounted directory. When both mounts are specified in the same container creation call, the more specific path (file) takes precedence.

**Example**: If `~/.claude/` is mounted rw as a directory bind mount, and `~/.claude/.credentials.json` is mounted ro as a file bind mount from a temp file, the container sees:
- `~/.claude/` → rw, host directory (sessions, config, etc.)
- `~/.claude/.credentials.json` → ro, temp placeholder file

**Verified**: This works with Docker's bind mount system. The file mount overlays the directory mount for that specific path.

## R4: Temp File Management

**Decision**: Create placeholder temp files in `os.TempDir()/codingbox-<session-id>/`.

**Rationale**: 
- Session-scoped directory avoids conflicts between concurrent sessions
- `os.TempDir()` is platform-appropriate (`/tmp` on Linux/macOS)
- Easy cleanup: remove the entire directory on session end
- Temp dir is created before container start and removed in `Stop()`

## R5: Placeholder File Format

**Decision**: For JSON files with a `json_key`, the placeholder file contains a modified copy of the original JSON with only the target key replaced by the placeholder value. For plain text files, the placeholder file contains just the placeholder string.

**Rationale**: If we replace the entire JSON file with just the placeholder string, Claude Code's JSON parser would fail. We need to keep the file structurally valid — only the secret value changes to the placeholder.
