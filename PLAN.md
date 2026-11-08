# Cockpit — Plan

Status: Draft. Owner: Faris Han (+ agents). Repo: farishan-labs/cockpit.

## Guiding principles

- Never reinvent exact wheels: reuse/aggregate/extend (herdr, pi, opencode,
  nakama, tmux, ...).
- Zero vendor lock-in: open formats, portable, self-hostable.
- Minimal core; everything else is extensions.
- Single static binary (Go, no CGO).

## Milestones

### M0 — Foundation (done-ish)

- [x] Repo created (farishan-labs/cockpit, public).
- [x] Go module scaffolded (go 1.24, cobra).
- [x] SPEC.md, PRINCIPLES.md, PLAN.md, README.md.
- [ ] LICENSE (Apache-2.0).
- [ ] CI: gofmt, go vet, go test, build (GitHub Actions).
- [ ] goreleaser config for single-binary releases.

### M1 — Minimal viable cockpit

Goal: `cockpit serve` runs a daemon with a web dashboard; you can see and
launch agents in tmux sessions.

Tasks:
1. `cockpit doctor`: detect tmux, agent CLIs on PATH, sqlite availability.
2. Core daemon: HTTP+WS server on localhost (configurable port), graceful
   shutdown, PID file.
3. Storage: SQLite (modernc) schema for sessions/agents/tools; JSONL event log.
4. Terminal backend interface + tmux implementation:
   - list sessions, create session, send keys, capture pane, kill.
5. Agent registry: detect known agents (claude, codex, pi, opencode, aider)
   on PATH; store metadata.
6. Launch agent: create tmux session, run agent CLI, record event.
7. Status: process liveness + OSC 7501 parsing where possible.
8. Web UI (embedded): dashboard listing sessions/agents/status; launch form.
   htmx + minimal CSS. Open browser on serve.
9. TUI client: `cockpit tui` attaches to daemon, lists sessions, launches.
   bubbletea.
10. Config: TOML at ~/.config/cockpit/config.toml (port, default agent,
    terminal backend, machines).
11. Docs: README quickstart, config reference.

### M2 — Tool registry & extensions

- Tool manifest (TOML) + `cockpit tool list|enable|disable|run`.
- Extension loader: dir of extensions with manifest + hooks.
- Hook points v1 (session start/end, pre/post tool).
- `cockpit tool install` from a git URL (vendored, no registry service).

### M3 — Aggregation of external tools

- Adapter: herdr-core (optional) for richer status + remote machines.
- Adapter: opencode (session listing, agent invocation).
- Adapter: pi (extension/skill discovery).
- Adapter: nakama (if/when a concrete use case lands).
- Unified dashboard across adapters.

### M4 — Polish & sharing

- goreleaser releases (linux/mac/windows, amd64/arm64).
- Homebrew tap / install script.
- Docs site (embedded or separate).
- Example extensions + tools.
- Community health files in farishan-labs org.

## Task ordering & dependencies

- M1.2 (daemon) blocks M1.8 (web) and M1.9 (tui).
- M1.3 (storage) blocks M1.5/M1.6.
- M1.4 (terminal backend) blocks M1.6.
- M2 depends on M1 complete.
- M3 depends on M2 (extension hooks) for adapter integration.

## Definition of done (M1)

- `go build` produces a single static binary.
- `cockpit doctor` reports tmux + at least one agent.
- `cockpit serve` starts, opens web UI showing sessions.
- Launching an agent from web UI creates a tmux session running it.
- `cockpit tui` lists and launches sessions.
- `go test ./...` passes; `gofmt` clean.

## Risks

- OSC 7501 status parsing complexity → fall back to liveness first.
- tmux version differences → pin to common subset of commands.
- Embedded web UI asset pipeline → keep it vanilla (no build step) for M1.
- Scope creep → hold the line on non-goals.

## Open decisions

- Default terminal backend for M1: tmux (decided).
- Extension language: start with executable scripts (any language), revisit
  Go/WASM later.
- Whether to depend on herdr-core in M1: no (M3).
