## Tasks

- [ ] Add `machines` config schema + DB table + migration
- [ ] Implement `internal/machines` package (registry CRUD, SSH exec helper)
- [ ] (moved to `machines`: tunnel backends + `machines check`) Consume `machines check --json` for reachability instead of a cockpit Tailscale adapter
- [ ] Add T3 Code adapter (local + remote over SSH)
- [ ] Add `/api/machines`, `/api/machines/check`, `/api/machines/pair` routes
- [ ] Add CLI subcommands `machine`, `t3 pair`
- [ ] Add embedded web UI "Machines" panel
- [ ] Add doctor integration (best-effort)
- [ ] Add JSONL event logging for checks/pairs
- [ ] Update SPEC.md (adapters + MVP)
- [ ] Write tests for registry + check aggregation (mock exec)
- [ ] Validate with `openspec validate remote-machines-t3code`