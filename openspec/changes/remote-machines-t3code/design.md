# Design: remote-machines-t3code

## Approach

Follow cockpit's aggregator principle: shell out to existing tools (`tailscale`,
`ssh`, `t3`) rather than reimplement. Each capability is an adapter implementing
the existing `adapters.Adapter` interface, so the dashboard and `/api/adapters`
endpoint gain them for free.

### Machine registry

- New config section `[[machines]]` in `~/.config/cockpit/config.toml`:
  ```toml
  [[machines]]
  name = "oscar"
  host = "oscar"              # tailscale magicdns or IP
  ssh_user = "faris"
  t3_port = 3773
  tags = ["personal", "linux"]
  ```
- Stored in SQLite (`machines` table) for runtime status; config is source of
  truth for definitions (mirrors tool registry pattern).
- `cockpit machine add <name> --host ...` writes config + DB row.

### Tailscale adapter

- `Available()` = `tailscale` on PATH.
- `Status()` runs `tailscale status --json` (or `tailscale ping -c1 <host>`)
  and extracts node IP/online for each registered machine. Report per-machine
  reachability in a `machines` key.

### T3 Code adapter (per machine)

- For local machine: run `t3 status --json` and `t3 --version`.
- For remote: `ssh <user>@<host> '<t3 path> status --json'`.
- Report: version, port, lan enabled, connect enabled, web URL, last pairing
  token (if cached).
- `Pair(machine, ttl)` runs `t3 pair [--tailscale]` over SSH and returns the
  pairing URL + token; token cached (short TTL) in DB, never in logs.

### Remote check flow

1. Tailscale reachability (ping).
2. SSH auth probe (`ssh -o BatchMode=yes ... true`).
3. T3 service status.
4. HTTP GET on `http://<host>:<port>/` (200 expected).

Result stored as `last_check` with timestamp + per-step pass/fail; emitted as
JSONL event.

### Web UI

- New "Machines" panel (htmx) hitting `/api/machines` and `/api/machines/check`.
- Machine card: name, host, status dot, T3 web link ("Open"), "Pair" button
  (shows token + QR), "Re-check".
- Detail view shows last check steps and cached pairing token.

### CLI

- `cockpit machine list|add|remove|check [name]`
- `cockpit t3 pair <machine> [--ttl 24h] [--tailscale]`

### Integration with `machines` product (optional)

If `machines` CLI/skill is present, cockpit can import its machine list
instead of (or in addition to) its own config. Keep cockpit's registry the
fallback so it works standalone.

## Risks / open questions

- SSH auth UX: first connection may require Tailscale SSH web approval; surface
  that in check output rather than hanging.
- `t3 pair --tailscale` may fail without Tailscale Serve permissions; fall
  back to tailnet base URL + token (as we did manually).
- Secret storage: reuse cockpit's SQLite; consider encrypting pairing tokens at
  rest (follow T3's own secret-file pattern).