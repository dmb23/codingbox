# Sandboxed Coding Environment for Coding Agents

## Current Status

| Phase | Item | Status |
|---|---|---|
| 1 | Docker Desktop 4.60 + Sandboxes v0.11.0 | Done |
| 1 | MicroVM sandbox created (`codingbox`) | Done |
| 1 | Filesystem isolation (workspace-only mount) | Done |
| 2 | Network deny-by-default policy | Done |
| 2 | Initial whitelist (49 domain rules) | Done |
| 2 | Network logging (JSON) | Done |
| 2 | `./sandbox` launcher script | Done |
| 2 | `./sandbox watch` for blocked request monitoring | Done |
| 2 | `./sandbox allow-host` for dynamic approval | Done |
| 1 | Git config (user, aliases, LFS) synced to sandbox | Done |
| 1 | Git credential injection (Docker `gh-token` helper) | Done (built-in) |
| 3 | Non-git secret injection (API keys via proxy) | TODO |
| 3 | Dual-mode profiles (agent vs developer) | TODO |
| 4 | Empirical whitelist refinement from logs | TODO |
| 4 | CI/headless profile (static whitelist, GET-only) | TODO |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│  SANDBOX (isolation boundary)                           │
│                                                         │
│  [Coding Agent]                                         │
│    - No secrets in env or files                         │
│    - Filesystem: explicit rw/ro mounts only             │
│    - HTTP_PROXY=unix:///run/proxy.sock                  │
│    - git credential.helper → proxy-helper               │
│                                                         │
│    └──→ Unix Domain Socket (bind-mounted from host)     │
└─────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│  HOST PROXY LAYER                                       │
│                                                         │
│  [mitmproxy / custom proxy]                             │
│    - Domain allowlist/denylist enforcement               │
│    - JSONL audit log of all requests                    │
│    - Credential injection per destination               │
│    - Interactive domain approval (new domain → prompt)  │
│    │                                                    │
│    └──→ OS Keychain / Vault / dotfiles                  │
└─────────────────────────────────────────────────────────┘
             │
             ▼
        [External: GitHub, npm, PyPI, APIs, ...]
```

---

## Hard Requirements

### 1. Filesystem Isolation

**Mechanism:** Mount-based isolation. The sandbox sees only explicitly declared paths.

| Mount Type | Examples | Implementation |
|---|---|---|
| **Read-write** | Project directory, `/tmp`, `~/.claude` | Bind-mount or virtiofs share |
| **Read-only** | `~/.gitconfig`, `~/.config/git/ignore`, toolchain dirs | Read-only bind-mount |
| **Denied** | `~/.ssh`, `~/.aws`, `~/.config/gh`, `~/.env`, rest of `$HOME` | Not mounted at all — invisible to sandbox |

On macOS with Docker/microVM, this is configured via volume mounts. With Anthropic's `sandbox-runtime` (which uses `sandbox-exec` on macOS), this is configured in `~/.srt-settings.json`:

```json
{
  "filesystem": {
    "denyRead": ["~/.ssh", "~/.aws", "~/.config/gh"],
    "allowWrite": ["./project", "/tmp", "~/.claude"],
    "denyWrite": [".env", "*.pem", "*.key"]
  }
}
```

### 2. Network Request Logging

**Mechanism:** All traffic routes through a proxy that logs every request in JSONL.

```jsonl
{"ts":"2026-02-13T10:00:01Z","method":"GET","url":"https://registry.npmjs.org/express","status":200,"bytes":15234}
{"ts":"2026-02-13T10:00:02Z","method":"POST","url":"https://api.anthropic.com/v1/messages","status":200,"bytes":8421}
{"ts":"2026-02-13T10:00:03Z","method":"GET","url":"https://evil.example.com/exfil","status":403,"blocked":true}
```

Implementation: mitmproxy addon that writes structured logs (see proxy section below). This is a byproduct of the proxy architecture — every request is already intercepted, so logging is trivial.

---

## Debated Point 3: Container vs VM

### Containers (Docker/Podman with hardening)

**Pros:**
- Sub-second startup
- Minimal memory overhead
- Rich ecosystem (devcontainers, Compose, pre-built images)
- On macOS, Docker Desktop already runs a Linux VM, so containers get a partial VM boundary "for free" — a container escape lands in the Docker VM, not on macOS directly

**Cons:**
- Shared kernel within the Docker VM — container-to-container escapes are real
- 3+ runC CVEs in 2025 alone (CVE-2025-31133, CVE-2025-52565, CVE-2025-52881)
- CVE-2025-9074: any container could access Docker's internal API and mount the host filesystem — bypassed Enhanced Container Isolation entirely
- Hardening (seccomp, AppArmor, `--cap-drop=ALL`, `--security-opt=no-new-privileges`) reduces but does not eliminate the attack surface
- The structural problem: the Linux syscall surface (~400 syscalls) is the attack boundary, and it produces CVEs every year

### MicroVM (Docker Sandboxes / Firecracker / Kata)

**Pros:**
- Hardware-enforced isolation (KVM/Apple Virtualization.framework) — each sandbox gets its own kernel
- ~125ms startup (Firecracker), subsecond (Docker Sandboxes on macOS)
- <5 MiB memory overhead per microVM
- Attack surface is the hypervisor (~50k lines of Rust for Firecracker), not the Linux kernel (~28M lines of C)
- Docker Sandboxes (Docker Desktop 4.58+) provides this on macOS natively, purpose-built for coding agents
- Apple's Containerization framework (2025) validates this model — each container in its own VM

**Cons:**
- Requires Docker Desktop 4.58+ (Docker Sandboxes) or Linux with KVM (Firecracker/Kata)
- Slightly more complex setup than plain `docker run`
- No GPU passthrough in Firecracker (irrelevant for coding agents)
- Docker Sandboxes not available on Linux

### Decision: MicroVM via Docker Sandboxes

**Rationale:** On macOS, Docker Sandboxes is purpose-built for exactly this use case. Each sandbox runs in its own microVM using Apple Virtualization.framework with a dedicated Docker daemon. Startup is subsecond. The security boundary is hardware-enforced. The alternative — hardened containers — still shares a kernel, and the 2025 CVE track record shows this is not a theoretical risk.

If later needed on Linux, Kata Containers (with Firecracker backend) provides the equivalent: VM-isolated containers via standard OCI/CRI APIs, ~150-300ms startup.

The "few seconds of startup time is acceptable" framing is actually generous — microVMs start in <1 second, comparable to containers.

---

## Debated Point 4: Network Whitelisting

### Option A: No whitelisting (log only)

**Pros:**
- Zero friction — agent can access anything
- No maintenance of domain lists
- Logging still provides auditability

**Cons:**
- Prompt injection can direct the agent to download/execute malicious payloads
- Agent can exfiltrate code, secrets, or context to attacker-controlled servers
- Logging detects abuse after the fact, not during

### Option B: Static whitelist (deny by default)

**Pros:**
- Strong protection against prompt injection exfiltration
- Predictable, auditable configuration

**Cons:**
- Maintaining the list is painful — every new dependency host, documentation site, or API requires a config change
- Agent workflow breaks when a legitimate domain isn't whitelisted
- Can create false sense of security (allowed domains like `github.com` still host arbitrary user content)

### Option C: Whitelist with interactive approval

**Pros:**
- Baseline protection from static whitelist
- New domains approved in real-time during the session
- Developer stays in control without pre-enumerating every domain
- Approvals can be scoped to session, or persisted to config

**Cons:**
- Requires developer attention — interrupts flow for approval prompts
- A sophisticated prompt injection could craft a plausible-looking domain request
- Implementation complexity: proxy must hold the connection while awaiting approval

### Generating the initial whitelist

Start with this curated set (based on Docker Sandboxes, OpenAI Codex, and Anthropic sandbox-runtime defaults):

| Category | Domains |
|---|---|
| **npm** | `registry.npmjs.org`, `*.npmjs.org` |
| **PyPI** | `pypi.org`, `files.pythonhosted.org` |
| **Cargo** | `crates.io`, `index.crates.io`, `static.crates.io` |
| **Go** | `proxy.golang.org`, `sum.golang.org` |
| **GitHub** | `github.com`, `*.github.com`, `*.githubusercontent.com` |
| **AI APIs** | `api.anthropic.com`, `api.openai.com` |
| **Linux packages** | `*.ubuntu.com`, `*.debian.org`, `*.alpinelinux.org` |
| **Container registries** | `*.docker.io`, `*.docker.com`, `ghcr.io` |

Refinement approach: run the agent for a few real tasks with `policy=allow` and full logging. Analyze the JSONL logs to extract all unique domains the agent actually contacted. This empirical list becomes your baseline whitelist.

### Dynamic extension mechanism

When the proxy intercepts a request to an unlisted domain:
1. Hold the TCP connection (configurable timeout, e.g., 30s)
2. Notify the developer via CLI prompt / terminal notification
3. Show: domain, full URL, HTTP method, requesting process
4. Developer responds: **allow (session)**, **allow (permanent)**, or **deny**
5. Decision is cached for the session duration or persisted to config

This is exactly how Anthropic's sandbox-runtime works for Claude Code.

### Decision: Static whitelist + interactive approval

**Rationale:** A pure static whitelist is too brittle for real coding work where dependencies pull from unexpected CDNs and documentation lives on varied domains. Interactive approval provides the safety net without requiring exhaustive pre-enumeration. The initial whitelist covers 90%+ of legitimate traffic, and interactive approval handles the long tail.

For CI/headless runs (no human to approve), use the static whitelist only, with the option to restrict HTTP methods to GET/HEAD/OPTIONS (the OpenAI Codex approach) to limit exfiltration even to whitelisted domains.

---

## Debated Point 5: Secret Injection

### Option A: Secrets in environment variables (status quo)

**Pros:**
- Simple, universal — every tool understands `$GITHUB_TOKEN`
- Zero friction for manual work

**Cons:**
- Agent can read all env vars via `env`, `printenv`, `/proc/self/environ`
- Agent can exfiltrate secrets in bash commands, error messages, or generated code
- Prompt injection attacks specifically target secret extraction

### Option B: Proxy-mediated transparent injection

**Pros:**
- Agent never sees or knows about secrets
- Destination-bound: a GitHub token can only be used against `*.github.com`
- Protocol-aware: git proxy can validate target branch, not just host
- Prompt injection cannot extract what doesn't exist in the sandbox

**Cons:**
- Complexity: requires a proxy that understands multiple protocols (HTTP, git, SSH)
- Debugging auth failures is harder — the agent sees unauthenticated requests
- Not all tools respect `HTTP_PROXY` — some need wrapper scripts or credential helpers

### Option C: Template/placeholder substitution

**Pros:**
- Agent knows placeholders exist, can use them intentionally
- Flexible — agent controls header/body placement
- Easier to debug: agent sees `__GITHUB_TOKEN__` in its own requests

**Cons:**
- Agent could try sending placeholders to unauthorized destinations (mitigated by destination binding)
- Prompt injection can target known placeholder names
- Agent awareness of secret existence is itself a risk

### How manual work functions with proxy-mediated injection

This is the crux of the concern, and there are practical solutions:

**Git operations:** Configure `git credential.helper` to point to a proxy-aware helper binary. When you run `git push` inside the sandbox, the credential helper calls the host-side proxy over the Unix socket, the proxy injects the real token, git authenticates transparently. This works identically for the agent and for you.

**CLI tools (curl, gh, npm):** Set `HTTP_PROXY` / `HTTPS_PROXY` pointing to the proxy socket. The proxy injects credentials based on destination. `curl https://api.github.com/user` works without you ever typing a token. For tools that don't respect the proxy env var, thin wrapper scripts delegate to the proxy's credential API.

**Running tests:** Tests that call external APIs route through the same proxy — credentials are injected transparently. Tests against local services use `localhost` which bypasses the proxy (or is configured to passthrough).

**Debugging auth failures:**
- Proxy audit logs show: destination, injected credential ID (not value), upstream response status
- A 401 from upstream immediately tells you the stored credential is wrong/expired
- `GIT_TRACE=1 git push` shows the credential helper interaction
- Breakglass mode: temporarily expose a credential to your shell session with audit logging and TTL

**Dual-mode operation:** The proxy supports two profiles:
- **Agent mode:** Strict allowlist, credential injection, full logging
- **Developer mode:** Broader access, credential injection still active (so you don't need secrets in env either), interactive approval for new domains, proxy in verbose logging mode

Switching is a single command or config flag. Both modes inject credentials the same way — the difference is network policy strictness.

### Decision: Proxy-mediated transparent injection

**Rationale:** The entire purpose of sandboxing is to limit damage from a compromised or prompt-injected agent. If secrets are in environment variables, all other sandboxing is undermined — the agent can exfiltrate them in a single `curl` to a whitelisted domain with user content (e.g., creating a GitHub gist). Transparent proxy injection eliminates this class of attack entirely.

The manual workflow concern is addressed by the fact that you also benefit from the proxy — you never need to manage tokens in dotfiles or env vars. The git credential helper and `HTTP_PROXY` patterns work identically for human and agent use. The only friction point is debugging auth failures, which the proxy audit log and breakglass mode handle.

Template/placeholder is rejected because agent awareness of secret existence is itself an attack surface. If the agent knows `__GITHUB_TOKEN__` exists, prompt injection can target it.

---

## Implementation Plan

### Phase 1: Isolation Foundation

1. **Install Docker Desktop 4.58+** and enable Docker Sandboxes
2. **Configure sandbox filesystem mounts:**
   - RW: project directory, `~/.claude`
   - RO: `~/.gitconfig`, `~/.config/git/ignore`, toolchain configs
   - Denied: everything else (default in microVM isolation)
3. **Verify isolation:** from inside sandbox, confirm `~/.ssh`, `~/.aws`, `~/.config/gh` are inaccessible

### Phase 2: Network Proxy & Logging

4. **Deploy mitmproxy** as the sandbox network proxy (mitmproxy for flexibility; Squid later if performance matters)
5. **Implement whitelist addon** with the initial domain list above
6. **Implement JSONL audit logging addon** — every request logged with timestamp, method, URL, status, bytes
7. **Implement interactive approval** — unknown domains trigger a CLI prompt, decision cached per session
8. **Configure the sandbox** to route all traffic through the proxy (`--policy deny --allow-host ...` or iptables/network namespace)

### Phase 3: Secret Injection

9. **Configure proxy credential injection rules** — map destination domains to secrets stored in macOS Keychain
10. **Create a git credential helper** that delegates to the proxy
11. **Create wrapper scripts** for tools that don't respect `HTTP_PROXY` (if any)
12. **Set up dual-mode profiles** (agent mode vs developer mode)
13. **Test the full flow:** agent pushes to GitHub, installs npm packages, calls APIs — all without secrets in the sandbox

### Phase 4: Hardening & Operational Maturity

14. **Run real agent tasks** in log-only mode to build empirical whitelist
15. **Refine whitelist** from logs, switch to deny-by-default
16. **Add CI/headless profile** with static whitelist + GET-only restriction
17. **Document breakglass procedures** for auth debugging

---

## Tool Choices Summary

| Component | Tool | Why |
|---|---|---|
| Isolation | Docker Sandboxes (macOS) | MicroVM per sandbox, purpose-built for agents, subsecond startup |
| Proxy | mitmproxy | Python scripting for custom addons (whitelist, logging, approval, credential injection) |
| Secret storage | macOS Keychain (via `security` CLI) | Already available, no additional infrastructure |
| Audit log | JSONL files via mitmproxy addon | Simple, greppable, ingestible by any log tool |
| Git credentials | Custom credential helper → proxy | Standard git protocol, transparent to user and agent |
| Fallback (Linux) | Kata Containers + Firecracker | Equivalent microVM isolation via Kubernetes RuntimeClass |

---

## Key References

- [Docker Sandboxes Architecture](https://docs.docker.com/ai/sandboxes/architecture/)
- [Docker Sandboxes Network Policies](https://docs.docker.com/ai/sandboxes/network-policies/)
- [Anthropic sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime)
- [Anthropic Engineering: Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing)
- [OpenAI Codex Cloud Internet Access](https://developers.openai.com/codex/cloud/internet-access/)
- [Fly.io Tokenizer](https://github.com/superfly/tokenizer) — secret-injecting proxy
- [CyberArk Secretless Broker](https://github.com/cyberark/secretless-broker)
- [Envoy Credential Injector Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/credential_injector_filter)
- [Apple Containerization Framework (WWDC 2025)](https://developer.apple.com/documentation/containerization)
- [Firecracker MicroVM](https://github.com/firecracker-microvm/firecracker)
- [Kata Containers](https://katacontainers.io/)
- [mitmproxy](https://mitmproxy.org/)
