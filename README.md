# Cockpit

![License](https://img.shields.io/github/license/farishan-labs/cockpit) ![Stars](https://img.shields.io/github/stars/farishan-labs/cockpit)

The first thing installed and opened when operating the **Faris Han Labs**.

A **human+AI control plane**: one place to see the dashboard, control things
(install/enable/disable tools), and drive co-pilots and crews (agents).

Cockpit is an **aggregator, not a replacement**. It reuses, aggregates, and
extends existing tooling — herdr, pi, opencode, nakama (ahmadrosid), tmux, SQLite — and
never reinvents an exact wheel. Zero vendor lock-in.

## Status

M1 complete. M2 tool registry scaffold (manifest + enable/disable/run) done.
Remaining: M2 extension loader/hooks, M3 adapters (herdr/opencode/pi), M4 polish.

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

## Install

From source:

```sh
git clone git@github.com:farishan-labs/cockpit.git
cd cockpit
./scripts/install-local.sh   # builds + installs to ~/.local/bin/cockpit
```

Or with make:

```sh
make install
```

Uninstall:

```sh
cockpit uninstall
# or: ./scripts/uninstall.sh / make uninstall
```

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

## Tools

```sh
cockpit tool list
cockpit tool add <name> <binary> [description]
cockpit tool enable|disable <name>
cockpit tool run <name> [args...]
```

Manifest: `~/.config/cockpit/tools.toml`.

## Tray (Ubuntu)

```sh
cockpit tray         # start tray indicator in background
cockpit tray status
cockpit tray stop
cockpit tray --foreground
```

## Layout

```
cmd/cockpit/          # CLI entrypoint (cobra)
internal/agents/      # agent detection
internal/config/      # config load/save
internal/daemon/      # HTTP+WS daemon, embedded web UI
internal/herdr/       # herdr workspace launcher
internal/storage/     # SQLite + JSONL event log
internal/terminal/    # terminal backend interface + tmux
internal/toolreg/     # tool registry
internal/tui/         # bubbletea TUI
internal/version/     # build version
SPEC.md               # specification
PLAN.md               # milestones
```

## Principles

Canonical principles live in the [`farishan-labs/principles`](https://github.com/farishan-labs/principles)
repo. Cockpit does not vendor or derive them (P-024); reference by id and link.

## License

Apache-2.0
EOF
echo written