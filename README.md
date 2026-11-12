# Cockpit

![License](https://img.shields.io/github/license/farishan-labs/cockpit) ![Stars](https://img.shields.io/github/stars/farishan-labs/cockpit)


The first thing installed and opened when operating the **Faris Han Labs**.

A **human+AI control plane**: one place to see the dashboard, control things
(install/enable/disable tools), and drive co-pilots and crews (agents).

Cockpit is an **aggregator, not a replacement**. It reuses, aggregates, and
extends existing tooling — herdr, pi, opencode, nakama (ahmadrosid), tmux, SQLite — and
never reinvents an exact wheel. Zero vendor lock-in.

## Status

M0 — foundation. **Research gate:** landscape documented below before any
M1 code. See [PLAN.md](./PLAN.md) and [SPEC.md](./SPEC.md).

## Principles

Canonical principles live in the [`farishan-labs/principles`](https://github.com/farishan-labs/principles)
repo. Cockpit does not vendor or derive them (P-024: single source, no drift);
reference by id (P-001..) and link to the source. Load the principles repo's
`TOP_PRINCIPLES.md` directly when needed.

## Landscape: inspirations, alternatives, adapters

> Research before code. If a proven tool already does it, we depend on,
> aggregate, or extend it — never rebuild it. This section is the gate.

### Terminal runtimes / multiplexers (we reuse, not rebuild)

| Tool | What it does | Reuse plan |
|------|--------------|------------|
| **tmux** | Ubiquitous persistent terminal multiplexer | **Default backend** for M1 (zero install friction) |
| **herdr-core (`hc`)** | Persistent terminal runtime for agents; multi-machine over SSH; OSC 7501 status; OpenRPC API; Go client exists | **Enhanced adapter** (M3); also candidate to *replace* tmux backend later |
| **Zellij / WezTerm / cmux / Warp** | Modern terminals/multiplexers | Reference only; not a dependency |
| **screen** | Legacy multiplexer | Reference only |

### Agent harnesses / coding agents (we orchestrate, not reimplement)

| Tool | What it does | Reuse plan |
|------|--------------|------------|
| **pi** (Earendil) | Minimal, extensible agent harness ("minimize core + programmable extensions") | **Inspiration** for extension model; optional adapter |
| **opencode** | Coding agent (this machine's daily driver) | **Adapter** (M3): session listing, agent invocation |
| **Claude Code / Codex / Aider / Goose / Cline** | Coding agents | We **launch and steer** them; never reimplement their loop |
| **OpenAI Agents / LangGraph / CrewAI** | Agent frameworks | Reference only |

### Agent cockpit / control-plane products (alternatives — study, don't duplicate)

| Tool | What it does | Relation to cockpit |
|------|--------------|---------------------|
| **SuperBased** | Control plane for AI agents: 27 CLI launchers, terminal grid, cost tracking, tailnet remote; Apache-2.0, local-first | **Closest alternative.** Study its terminal grid + cost tracking; differentiate via tool registry + extensions + Go single binary |
| **herdr** (app + TUI) | Agent terminal workspace; multi-machine; Apache-2.0 | **Closest infrastructure.** Cockpit may be a *client* on herdr-core rather than a parallel product |
| **Open Orchestrator** | Multi-provider cockpit over git worktrees + tmux; MIT, Python | **Closest feature match** (M1–M2). Differentiate via single Go binary + tool registry + extensions |
| **agent-cockpit** | Local control room for Claude Code + Codex; pixel-art UI; Node daemon | Study approval inbox + session chat |
| **AI Agent Session Center** | Real-time dashboard for Claude Code + Codex; live terminals, prompt queues; Electron/browser | Study prompt-queue + auto-resume ideas |
| **AGI Cockpit** | Parallel agent manager; desktop + mobile PWA | Study mobile supervision |
| **Claude Code Agent View** | Built-in multi-agent cockpit | Baseline to beat on UX, not on scope |
| **AgentsRoom** | Native desktop app for parallel agents | Reference for multi-project UX |

### AI agent platforms (reuse if a concrete need lands)

| Tool | What it does | Reuse plan |
|------|--------------|------------|
| **Nakama** (ahmadrosid) | AI agent platform for teams: self-hosted, multi-org, agents with tools, skills, memory, and channels (Telegram/WhatsApp/Discord/Slack) | **Reference/alternative** (M3+); study its multi-org + agent identity + channel model. Not a core dependency. |

### Tooling / libraries we build on (not reinvent)

- **Go** (single static binary, no CGO) · **cobra** (CLI) · **bubbletea** (TUI) · **htmx** (web UI) · **modernc SQLite** (storage) · **TOML** (config/manifests).

### Honest assessment

This is a **crowded space**. Several tools already do most of M1–M2.
Cockpit's defensible angle is narrow: a **lab control plane** that (a) treats
tool registry + programmable extensions as first-class, (b) composes existing
tools instead of reimplementing them, (c) ships as one static Go binary with
zero runtime deps, and (d) is honest about prior art. If that angle does not
hold up against SuperBased / herdr / Open Orchestrator, the right move is to
**contribute to one of them** instead of shipping a parallel product.

## Install (one line)

```sh
curl -fsSL https://farishan-labs.github.io/cockpit/scripts/install.sh | bash
```

Idempotent and safe to re-run: it (re)installs the `cockpit` command to
`~/.local/bin/cockpit`. Then type `cockpit` in a plain bash terminal to check
and open `herdr-app` with a default space at `~/dev/labs` and `opencode` ready
(override the dir with `COCKPIT_DIR=/path/to/dir cockpit`).

## Quick start (dev)

```sh
go build ./...
./cockpit version
./cockpit doctor   # (M1)
./cockpit serve    # (M1)
```

## Layout

```
cmd/cockpit/        # CLI entrypoint (cobra)
internal/version/   # build version
SPEC.md             # specification
PLAN.md             # milestones
```

## License

Apache-2.0 (to be added).
EOF
echo written

## Installation

```bash
go get github.com/farishan-labs/cockpit
```

## Quickstart

```sh
cockpit              # start daemon in background (opens web UI)
cockpit status       # check daemon
cockpit tui          # terminal UI
cockpit open         # open herdr workspace (legacy launcher)
cockpit doctor       # check tmux + agents
cockpit stop         # stop daemon
```

Web UI: http://127.0.0.1:7777

## Config

`~/.config/cockpit/config.toml`:

```toml
port = 7777
default_agent = "opencode"
terminal_backend = "tmux"
db_path = "~/.config/cockpit/cockpit.db"
event_log_path = "~/.config/cockpit/events.jsonl"
open_browser = true
```

## Tray (Ubuntu)

```sh
cockpit tray         # start tray indicator in background
cockpit tray status
cockpit tray stop
```
