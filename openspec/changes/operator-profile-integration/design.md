## Context

See proposal.md (Why/What). Cockpit already has a config loader (`internal/config`),
a doctor command, an agent launcher, and an embedded htmx web UI. `usermd-check`
is an optional external binary (soft dependency). Operator profile integration is
additive; no breaking changes to existing commands.

## Goals / Non-Goals

**Goals:**
- Make `USER.md` a first-class input to cockpit's agent launch path.
- Validate it with the existing `usermd-check` binary (reuse, don't reinvent).
- Surface status in doctor and web UI.

**Non-Goals:**
- No new LLM calls or agent loop changes.
- No multi-user auth.
- No memory system; OptMem remains the memory layer.

## Decisions

1. **Config key placement.** Add `user_md_path` to the existing `config.toml`
   (BurntSushi/toml) rather than a new file. *Alt: separate operator.toml —
   rejected to keep config surface small (P-003).*

2. **Validation via subprocess.** Call `usermd-check` as an external binary
   (exec.LookPath) instead of linking a Go library. *Alt: import user-md/checker
   as a module — rejected; keeps cockpit's dependency graph minimal and avoids
   coupling release cycles (P-001 reuse, not vendoring).*

3. **Injection mechanism.** Export `USER_MD_PATH` env var AND symlink `USER.md`
   into the session cwd if missing. Env covers harnesses that read the variable;
   symlink covers harnesses that look for `USER.md` in cwd. *Alt: copy file —
   rejected; symlink keeps single source of truth.*

4. **Doctor behavior when validator absent.** Warn, don't fail. Keeps cockpit
   usable on machines without `usermd-check` installed (P-008 reliability).

5. **UI panel.** Read profile + run validator server-side, render via htmx
   (consistent with existing web UI). No client-side JS beyond htmx.

## Risks / Trade-offs

- [usermd-check not installed] → doctor warns; launch injection still sets env
  and symlinks (file presence check skipped).
- [profile too large / invalid] → doctor fails; launch still proceeds with
  warning banner in UI.
- [symlink target deleted] → launch-time check re-creates symlink; log event.

## Migration Plan

1. Add config key (backward compatible; default path).
2. Implement doctor check.
3. Implement launch injection.
4. Add UI panel.
5. (Later) shared workflow.toml manifest (separate change).

Rollback: remove config key usage; all changes are additive.

## Open Questions

None blocking. Future: shared workflow.toml schema (tracked in separate change).