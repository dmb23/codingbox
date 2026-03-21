# Codingbox

>![INFO] **Idea**
> I want to create a sandbox environment for coding agents.

## Goal:

- **MUST** create a secure sandbox in a MicroVM
- **MUST** specify the initial content of the sandbox
  - claude code, open code, vibe
  - uv, ruff, ...
  - basic dev setup
- **MUST** HTTP Server transparantly intercepting all calls
  - inject known authorization wihtout the secret being available in the Sandbox
  - log all calls
- **MUST** possible to configure access to directories
  - live rw access to current project dir
  - ro access to some config dirs
- **MUST** persistent state
  - if the environment is changed (new packages), this should persist

## Implementation Ideas
  - probably `docker sandbox` is easiest
  - find out how to route `docker sandbox` to an MITM server running in a normal container
  - secret config: define env vars, create a UUID for those in the sandbox, replace them in HTTP calls to defined hosts
  - log all calls in a db, json field

## Future
- viewer for logs
