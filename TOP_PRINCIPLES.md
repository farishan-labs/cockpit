# Faris Han Labs — Top Principles

Canonical source: https://github.com/farishan-labs/principles
This file is **derived**. Regenerate with `scripts/render-top.sh`.
Full set (22 principles): see `PRINCIPLES.md`.

> The non-negotiables. Six principles that govern everything built under
> the lab. If a decision conflicts with one of these, the principle wins.
> Load this file first; reach for `PRINCIPLES.md` only when you need
> rationale, examples, or the specialised rules.

## Top principles

1. **P-001 — Never reinvent exact wheels.** If a proven tool already does it, depend on it, aggregate it, or extend it. Build only the glue and the control plane.
2. **P-002 — Zero vendor lock-in.** Open formats, portable code, self-hostable. No dependency on a single giant tech company's proprietary service. Users can fork and run fully offline.
3. **P-003 — Minimal core.** Keep the core tiny (Pi: 4 tools, shortest prompt). Push everything else into extensions. Draw nothing, guess nothing (herdr-core).
4. **P-004 — Programmable extensions, not toggles.** Extensions are code with a manifest and hook points. The agent can author them ("software that builds itself"). Configuration toggles are not extensions.
5. **P-005 — Single static binary.** One file to install. Go, no CGO, no runtime deps.
6. **P-006 — Open and shared.** Apache-2.0. Others self-host and fork freely.

## All principles by group

- **Reuse & research:** P-001 (Never reinvent exact wheels), P-007 (Reuse before build), P-011 (Research before creating a new repo), P-021 (Research gate before feature code)
- **Portability & lock-in:** P-002 (Zero vendor lock-in), P-012 (Canonical files are plain markdown/JSON), P-013 (Adapters are the only harness-aware code), P-022 (Degrade gracefully)
- **Minimalism & scope:** P-003 (Minimal core), P-004 (Programmable extensions, not toggles), P-009 (Small and focused), P-018 (Forager only forages)
- **Distribution:** P-005 (Single static binary), P-006 (Open and shared)
- **Reliability & control:** P-008 (Reliability over novelty), P-010 (Human sets direction, agents execute)
- **Security:** P-015 (Never store secrets in shared memory), P-016 (Scan for secrets before committing)
- **Coordination:** P-017 (Durable changes are announced)
- **Agent capabilities:** P-014 (Skills declare their capabilities)
- **Auth & consent:** P-019 (Keys are manual; never sign up on behalf of the user)
- **CLI:** P-020 (CLIs follow open conventions)

## Optional (secondary)

- Full rationale, examples, applies-to: `PRINCIPLES.md`
- Machine-readable index (ids, titles, tags, tiers, files): `principles.toml`
- Change history: `CHANGELOG.md`
- Validation: `scripts/validate.sh`
