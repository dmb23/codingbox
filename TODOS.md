> claude --resume 980add33-88f2-47bd-b38d-8029edd45baf

- WIP: state persistence. Now ~/.local is persisted
- [ ] dir-specific config. If I do not specify differently, I want to start codingbox in a directory, and automatically apply config and naming for that directory, stored in a central location.
- [ ] base image: create a reasonable base image that provides curl, python toolchain (uv, ruff, ty), go toolchain, LSPs, claude code, open code, mistral vibe
  - possibly include plugins like superpowers / speckit / playground
- [ ] debug web access
- [ ] figure out config for secrets
  - probably: claude just needs to login (check if this is persisted)
  - mistral should work
