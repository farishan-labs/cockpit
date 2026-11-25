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

- Introduce a **Machine registry** in cockpit: named machines with transport
  (tailscale host / SSH target), labels (work/personal), and stored metadata.
- Add a **Tailscale adapter** (`tailscale status/ping`) reporting node
  reachability, IP, OS, and online/offline.
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
- Storage: persist machine registry + last-check results in SQLite; append
  check events to the JSONL event log.
- (Optional, later) Consume a shared machine list from `machines` product /
  `workflow.toml` instead of a cockpit-only list.

## Capabilities

### New Capabilities
- `remote-machines`: registry, reachability, and health checks for named
  remote machines.
- `t3-code-remote`: aggregate T3 Code service status and web/pairing access
  for a machine (local or remote over SSH).

### Modified Capabilities
- `adapters`: add Tailscale and T3 Code adapters to the default set.
- `doctor`: include machine reachability + T3 service in `cockpit doctor`
  output (best-effort, skip if no machines configured).

## Impact

- Affected: `internal/config`, `internal/storage`, `internal/adapters`,
  `internal/daemon` (routes + WS notify), `cmd/cockpit` (new subcommands),
  embedded web UI, `internal/terminal` (SSH exec helper), `SPEC.md`.
- New external (soft) deps: `tailscale` and `t3`/`ssh` on PATH; adapters skip
  with a warning if absent (mirrors `usermd-check` pattern).
- Non-goals: no reimplementing Tailscale, SSH, or T3 Code; cockpit only
  aggregates and drives them. No cloud hosting; multi-machine stays SSH/tailnet.
- Security: pairing tokens and SSH targets stored with the same sensitivity as
  existing secrets; tokens are short-lived and never logged in full.