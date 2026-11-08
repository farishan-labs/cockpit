# Cockpit — Specification

## Purpose

The first thing installed and opened when operating the Faris Han Labs. A
**human+AI control plane**: one place to see the dashboard, control things
(install/enable/disable tools), and drive co-pilots and crews (agents).

Cockpit is an **aggregator, not a replacement**. It reuses, aggregates, and
extends existing tooling (herdr, pi, opencode, nakama, tmux, ...). It never
reinvents an exact wheel.

## Prior art & adapters (research gate)

Cockpit is built **after** surveying the landscape; see README "Landscape".
No reimplementation of an existing wheel. Concrete dependencies/adapters:

- **tmux** — default terminal backend (M1).
- **herdr-core (`hc`)** — enhanced terminal backend + multi-machine + OSC 7501
  status + OpenRPC API (Go client on pkg.go.dev). Candidate to *replace* the
  tmux backend if it proves strictly better; until then, tmux is the
  zero-friction default.
- **pi** — inspiration for minimal core + programmable extensions; optional
  adapter for skill/extension discovery.
- **opencode** — adapter for session listing / agent invocation.
- **nakama** — adapter only if real-time multi-agent coordination need
  emerges (M3+); not a core dependency.
- **SuperBased / Open Orchestrator / agent-cockpit / AI Agent Session Center**
  — studied as alternatives; cockpit differentiates via tool registry +
  programmable extensions + single static Go binary + composition principle.
  If the angle does not hold, pivot to contributing to one of them.

## Principles (non-negotiable)

Canonical source: [`farishan-labs/principles`](https://github.com/farishan-labs/principles)
(pinned submodule `./principles`). Cockpit's `PRINCIPLES.md` is derived from
it. Do not duplicate or edit principles here; reference by id (P-001..).

1. **Never reinvent exact wheels.** (P-001)
2. **Zero vendor lock-in.** (P-002)
3. **Minimal core.** (P-003)
4. **Programmable extensions.** (P-004)
5. **Single static binary.** (P-005)
6. **Open and shared.** (P-006)
7. **Reuse before build.** (P-007)
8. **Reliability over novelty.** (P-008)
9. **Small and focused.** (P-009)
10. **Human sets direction, agents execute.** (P-010)

## Architecture

Composition of a minimal core plus adapters and clients:

```
                ┌──────────────────────────────┐
                │        Clients               │
                │   TUI (bubbletea)            │
                │   Web UI (embedded, htmx)    │
                │   CLI (cockpit ...)          │
                └──────────────┬───────────────┘
                               │  OpenRPC/HTTP+WS
                ┌──────────────┴───────────────┐
                │        cockpit core (Go)     │
                │  state · registry · router   │
                │  agent orchestration         │
                │  terminal backend interface  │
                └──────┬───────────┬───────────┘
                       │           │
        ┌──────────────┴┐   ┌─────┴──────────────┐
        │  Adapters      │   │  Storage           │
        │  herdr-core    │   │  SQLite (modernc)  │
        │  tmux          │   │  JSONL event log   │
        │  pi / opencode │   │  (Pi-style replay) │
        │  nakama        │   └────────────────────┘
        │  ...           │
        └────────────────┘
```

### Core (minimal)

- Owns state, registry, routing, agent lifecycle, event log.
- Speaks an open protocol (OpenRPC-style, like herdr-core's `hc schema`).
- Draws nothing, guesses nothing: status comes from programs (OSC 7501) or
  process liveness, not screen scraping.

### Terminal backend (interface)

- Default: `tmux` (ubiquitous, persists across client quit, multi-machine
  via SSH for free).
- Optional: `herdr-core` (`hc`) for richer agent status (OSC 7501) and
  remote machine management.
- Swappable; cockpit never reimplements a terminal multiplexer.

### Agent orchestration

- Cockpit does NOT implement LLM calls. It launches agent CLIs (claude,
  codex, pi, opencode, aider, ...) inside terminal sessions and steers them.
- Tracks status via OSC 7501 where supported, else process liveness.
- Co-pilots (single agents) and crews (agent teams) are both just sessions
  with a role/label.

### Tool registry

- A manifest (TOML) of tools: name, binary, version, enabled, source.
- `cockpit tool install|enable|disable|list|run`.
- Tools are external binaries/scripts; cockpit only registers and runs them.

### Extensions

- Directory of extensions: each has a manifest + executable + hook points.
- Hook points mirror Pi's lifecycle (pre/post tool, session events, ...).
- The agent can author extensions (Pi's "software that builds itself").

### Storage

- SQLite (modernc, pure Go) for structured state.
- Append-only JSONL event log (Pi-style) for replayable sessions.

### Clients

- TUI: bubbletea (standard Go TUI). Keyboard-driven.
- Web UI: served by core on localhost, embedded via `embed`. htmx for
  interactivity. Opens in the default browser.
- CLI: `cockpit` for scripting and agent use.

## MVP (M1)

1. `cockpit version`, `cockpit doctor` (check tmux, agents, sqlite).
2. `cockpit serve` starts the core daemon (HTTP+WS on localhost) and opens
   the web UI; TUI attaches via `cockpit tui`.
3. Dashboard: list terminal sessions + agents + their status.
4. Launch an agent (claude/pi/opencode) in a tmux session from the UI.
5. Tool registry: list + enable/disable (manifest-based, no install yet).
6. Append-only event log for launched sessions.

## Non-goals (v1)

- Reimplementing a terminal multiplexer.
- Reimplementing an LLM agent loop.
- Cloud/SaaS hosting (local-first; sharing via git/fork).
- Multi-user auth (single user + agents).

## Open questions

- Default agent set and detection (PATH scan).
- Extension hook API shape (borrow Pi's event names).
- Whether to depend on herdr-core or stay tmux-only for M1.
- GUI: embedded web vs native (Tauri). M1 = embedded web.
