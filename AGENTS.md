# Cockpit (FHL) — agent notes

- `cockpit.local` (nginx) always points to **this** cockpit (FHL), listening on `127.0.0.1:7777`.
- Hubton cockpit is a separate app on `127.0.0.1:4777`; expose it via `hubton.local` (see `~/.config/valet/Nginx/hubton.local`), never via `cockpit.local`.
- If `cockpit.local` returns 502, the FHL daemon is down — restart with `cockpit serve` (menubar icon turns red automatically).
- Menubar icon: `menubar/cockpit-icon.swift` → `menubar/cockpit-icon`; rebuild with `menubar/build-icon.sh`. Health check polls `http://127.0.0.1:7777/` every 10s.