## Tasks

- [ ] Add T3-state schema + DB table + migration (keyed by `machines` ID: port, pairing tokens, service status, check history)
- [ ] Implement `internal/machines` package (SSH exec helper + T3 state CRUD; NO machine inventory CRUD — consumed from `machines`)
- [ ] Consume `machines list/check --json` for inventory + reachability (already in adapters; extend adapter with check)
- [ ] Add T3 Code adapter (local + remote over SSH)
- [ ] Add `/api/machines`, `/api/machines/check`, `/api/machines/pair` routes
- [ ] Add CLI subcommands `machine list|check` (read-only, from `machines`), `t3 pair`
- [ ] Add embedded web UI "Machines" panel
- [ ] Add doctor integration (best-effort)
- [ ] Add JSONL event logging for checks/pairs
- [ ] Update SPEC.md (adapters + MVP)
- [ ] Write tests for T3 state CRUD + check aggregation (mock exec)
- [ ] Validate with `openspec validate remote-machines-t3code`