# Operator Profile Integration — Cockpit ↔ user-md

> Status: ideation (2026-11-12). Goal: make `cockpit` (control plane) and
> `user-md` (operator profile template + validator) point at the **same
> agentic workflow system**, instead of being two unrelated Faris Han Labs
> repos.

## Problem

Today the two repos are siblings with no edge between them:

- `cockpit` launches agent CLIs (claude/codex/pi/opencode) in tmux sessions,
  tracks status, owns a tool registry and an extension hook system.
- `user-md` defines and validates `USER.md` — the portable operator profile
  (tastes, identity, constraints) that those agents are supposed to read.

Nothing in cockpit reads, validates, or injects `USER.md`. An agent launched
by cockpit has no guarantee it ever sees the operator profile. The two products
therefore do **not** participate in one workflow system; they just happen to
share an org and a language.

## Spec engine (OpenSpec)

OpenSpec (`openspec` CLI) is the spec management system for cockpit and the
canonical **spec engine** slot in the agentic workflow system (shared with
`user-md` and `farishan/agent`).

- Cockpit repo: `openspec/` (config, changes, specs), `.opencode/`, `.pi/`
  installed by `openspec init`.
- Change `operator-profile-integration` is the first OpenSpec change; it
  defines the `operator-profile` capability.
- Other repos adopt the same tooling; the workflow manifest references the
  active spec engine (`openspec`) so agents know where specs live.

## Unifying concept: Operator Context

Introduce one first-class artifact — the **Operator Context** — that both repos
(and the personal core `farishan/agent`) consume and produce:

```
Operator Context (single source of truth)
├── USER.md          (tastes, identity, constraints)        ← user-md owns
├── machine profile  (paths, env, OS)                      ← profiles/machines
├── AGENTS.md ptr    (project/global instructions)         ← agent repo
├── memory seed      (OptMem durable facts)                ← OptMem
└── active goals     (side-hustle goals DB)                ← session-gate
```

Both `cockpit` and `user-md` become **consumers/producers of this artifact**
rather than independent products. The OpenClaw split is preserved: `SOUL.md`
(agent persona) never feeds `USER.md` (operator profile).

## Integration designs (ranked)

### A. Cockpit consumes USER.md (read-only injection) — SMALL, HIGH VALUE
- Add `user_md_path` to `~/.config/cockpit/config.toml` (default
  `~/.config/cockpit/USER.md`, or symlink to `~/dev/user-md/USER.md`).
- On agent launch, cockpit exports `USER_MD_PATH` (and, if the agent's cwd
  lacks a `USER.md`, symlinks/copies the configured file there) so the harness
  picks it up.
- `cockpit doctor` runs `usermd-check <path>` and reports pass/fail + cap
  usage.
- Effort: small. Value: high — every launched agent gets operator context.

### B. user-md ships as a cockpit extension — MEDIUM
- `user-md` provides an extension manifest + a `session:start` hook that
  injects the profile into agent prompts.
- `usermd-check` runs as a `pre:tool` / launch pre-flight hook.
- Depends on cockpit M2 extension loader (still pending).
- Effort: medium. Value: high (programmable, agent-authorable).

### C. Cockpit tool registry auto-registers FHL CLIs — SMALL
- `cockpit tool sync` scans PATH/known install dirs for `usermd-check`,
  `memory`, `sweep`, etc. and registers them.
- `user-md` ships a `cockpit-tool.toml` snippet; cockpit reads manifests from
  a local dir or registry URL.
- Effort: small. Value: medium (discoverability of usermd-check).

### D. Shared event log / telemetry — MEDIUM
- Cockpit's JSONL event log records `operator_profile_version` +
  `user_md_hash` per session.
- `user-md-ops` demand signals can be derived from cockpit session counts.
- Effort: medium. Value: medium (analytics, demand probe automation).

### E. Unified workflow manifest (the "same system" literal) — MEDIUM/LARGE
- A single manifest (e.g. `~/.config/fhl/workflow.toml`, or a section of the
  personal core `farishan/agent`) that **both** cockpit and user-md reference:
  paths to USER.md, AGENTS.md, machine profile, memory, principles, agent set,
  work vs personal mode.
- `cockpit` reads it at startup; `usermd-check` validates it; OptMem seed
  derives from it.
- Effort: medium-large. Value: very high — one manifest, many consumers.

### F. Cockpit renders USER.md in the UI — SMALL
- Web UI "Operator" panel: current USER.md (read-only), validation status,
  cap usage, last audit date, link to `usermd-check` output.
- Effort: small. Value: medium (visibility).

### G. Memory bridge — MEDIUM
- Export `USER.md` from OptMem memories (and import preferences back).
- `cockpit` exposes "export operator profile" using OptMem + the user-md
  template.
- Effort: medium. Value: medium (reduces duplication).

## Recommended minimal viable connection (MVP)

Land these in cockpit M2/M3, in order:

1. **Config**: add `user_md_path` to cockpit config (design A-lite).
2. **Doctor**: `cockpit doctor` runs `usermd-check` on that path.
3. **Launch injection**: export `USER_MD_PATH` + ensure file in agent cwd.
4. **UI**: Operator panel (design F).
5. Later, when M2 extension loader lands: extension B + tool sync C.

## E-lite: shared manifest sketch

```toml
# ~/.config/fhl/workflow.toml  (or section in farishan/agent)
[operator]
user_md_path   = "~/.config/cockpit/USER.md"
machine_profile = "~/.config/cockpit/machine.toml"   # or profiles/machines

[agents]
default = "opencode"
set     = ["claude", "codex", "pi", "opencode", "aider"]

[memory]
optmem_dir = "~/.optmem/memory"

[spec]
engine = "openspec"          # spec management system (cockpit, user-md, agent)
root   = "openspec/"
```

`cockpit` reads `[operator]` + `[agents]` + `[mode]` at startup;
`usermd-check --workflow ~/.config/fhl/workflow.toml` validates that the
referenced `USER.md` exists, passes cap/secrets/contradiction checks, and that
`user_md_path` is consistent with the machine profile.

## Keep the splits (do not violate)

- `SOUL.md` (agent persona) and `USER.md` (operator profile) never feed each
  other (OpenClaw rule, already in user-md README).
- Infra rules stay in plugins/AGENTS.md, not in USER.md (user-md judgement
  memory #695).
- Machine facts stay in the machine profile, not USER.md.
- Cockpit must not become a second memory system; it points at OptMem.

## Phased plan

- **Phase 0 (now)**: this doc + config key + doctor check. ~1 day.
- **Phase 1**: launch injection + Operator UI panel. ~2-3 days.
- **Phase 2**: shared workflow manifest (E-lite) + usermd-check `--workflow`
  flag. ~1 week.
- **Phase 3**: extension B + tool sync C + memory bridge G. Depends on
  cockpit M2 extension loader. ~2 weeks.

## Success criteria

- An agent launched by `cockpit` has `USER_MD_PATH` set and the file present
  in its cwd; `usermd-check` passes.
- `cockpit doctor` fails if the operator profile is missing/invalid.
- One manifest (`workflow.toml`) is the single place that changes when the
  operator or machine changes; both tools read it.
- No duplication of tastes between USER.md, AGENTS.md, and OptMem.
