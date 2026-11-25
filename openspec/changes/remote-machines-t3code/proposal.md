## Why

Operating Faris Han Labs now spans multiple machines (Mac workstation + "oscar"
Linux box). Today, checking whether a remote machine is reachable and whether
its T3 Code web UI is openable is a manual, ad-hoc sequence (tailscale ping,
SSH auth, `t3 status`, `t3 pair`, curl). Cockpit already aggregates local
tools/adapters and is the natural "first thing opened" control plane, but it
has no concept of **machines** or **remote T3 Code environments**. Adding this
turns the manual checklist we just ran into a one-click dashboard action and
keeps the multi-machine workflow (Mac ↔ oscar) visible and auditable.

## What Changes

- Consume the `machines` product as the single source of truth for machine
  inventory, tunnel config (incl. SSH targets), and reachability. Cockpit does
  NOT maintain a parallel machine registry; all T3-specific state is keyed by
  `machines` ID.
- (Removed) Tailscale adapter — reachability comes from `machines check --json`.
- Add a **T3 Code adapter** that, per machine, reports service version, port,
  LAN/T3-Connect state, and a reachable web URL; can mint a pairing token via
  `t3 pair` (local or `--tailscale`).
- Add a **Remote check** flow: ping → SSH auth probe → T3 status → HTTP 200 on
  the T3 port, with a single aggregated health result.
- Web UI: a "Machines" panel listing machine cards (status dot, T3 web URL,
  "Open", "Pair", "Re-check" actions) plus a detail view (logs, last check,
  pairing token + QR).
- CLI: `cockpit machine add|list|remove|check [name]`, and
  `cockpit t3 pair <machine> [--ttl]`.
- Storage: persist T3-specific state (port, pairing tokens, service status,
  check history) in SQLite keyed by `machines` ID; append check events to the
  JSONL event log. No machine inventory duplication.
- (Resolved) `machines` product is the single source of truth for inventory;
  cockpit consumes `machines list/check --json`.

## Capabilities

### New Capabilities
- `remote-machines`: registry, reachability, and health checks for named
  remote machines.
- `t3-code-remote`: aggregate T3 Code service status and web/pairing access
  for a machine (local or remote over SSH).

### Modified Capabilities
- `adapters`: add `machines` (inventory + reachability) and T3 Code adapters
  to the default set. No Tailscale adapter in cockpit.
- `doctor`: include machine reachability + T3 service in `cockpit doctor`
  output (best-effort, skip if no machines configured).

## Impact

- Affected: `internal/config`, `internal/storage`, `internal/adapters`,
  `internal/daemon` (routes + WS notify), `cmd/cockpit` (new subcommands),
  embedded web UI, `internal/terminal` (SSH exec helper), `SPEC.md`.
- New external (soft) deps: `machines` (provides reachability via its own
  tunnel backends, incl. Tailscale/SSH/custom), `t3`/`ssh` on PATH; adapters
  skip with a warning if absent (mirrors `usermd-check` pattern). Cockpit
  itself does not call `tailscale` directly.
- Non-goals: no reimplementing Tailscale, SSH, or T3 Code; cockpit only
  aggregates and drives them. No cloud hosting; multi-machine stays SSH/tailnet.
- Security: pairing tokens and SSH targets stored with the same sensitivity as
  existing secrets; tokens are short-lived and never logged in full.