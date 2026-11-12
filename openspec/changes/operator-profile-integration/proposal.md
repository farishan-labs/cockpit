## Why

Cockpit launches agent CLIs but never feeds them the operator profile (`USER.md`), so
launched agents may ignore who the operator is. `user-md` defines/validates that
profile but has no runtime consumer. The two repos need one shared Operator Context.

## What Changes

- Add `user_md_path` to cockpit config (`~/.config/cockpit/config.toml`).
- `cockpit doctor` runs `usermd-check` on the configured path; reports pass/fail + cap usage.
- On agent launch, cockpit exports `USER_MD_PATH` and ensures the file is present in the
  session's working directory.
- Web UI gains an "Operator" panel (profile text, validation status, last audit date).
- (Later) Read a shared `~/.config/fhl/workflow.toml` manifest (paths, agent set, mode).

## Capabilities

### New Capabilities
- `operator-profile`: cockpit consumes, validates, and injects the operator profile
  (`USER.md`) into launched agent sessions.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- Affected: `internal/config`, `internal/daemon`, `cmd/cockpit` (doctor, launch paths),
  embedded web UI, `docs/operator-profile-integration.md`.
- New external dependency (soft): `usermd-check` binary on PATH (optional; doctor skips
  with warning if absent).
- SPEC.md sections: "Architecture → Tool registry" (operator profile subsection added),
  "MVP (M1)" (doctor + launch injection).

## Non-goals

- Cockpit does not become a memory system; it points at OptMem.
- No change to `SOUL.md`/`USER.md` split (OpenClaw rule preserved).
- No multi-user auth (single operator + agents).