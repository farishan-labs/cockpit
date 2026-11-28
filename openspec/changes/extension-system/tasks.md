## Tasks

- [x] Create `internal/extensions` package: manifest struct, discovery (dir + env), validation
- [x] Define `extensions.Extension` interface and `Tab`/`Command` types
- [x] Add native extension registration (build-tag or explicit list) and `Mount`
- [x] Implement remote extension HTTP contract client (`/cockpit/health`, `/cockpit/manifest`)
- [x] Add SQLite `extensions` table + migration in `internal/storage`
- [x] Implement enable/disable persistence and `available`/`last_sync` updates
- [x] Add reverse-proxy handler for remote extensions (`/ext/<name>/*`)
- [x] Add `/api/extensions` daemon route (list + tabs + availability)
- [x] Add CLI subcommands `extension list|add|remove|enable|disable|sync`
- [x] Make embedded web UI tab bar data-driven from `/api/extensions` (iframe tabs)
- [x] Add `cockpit extension` to `cockpit doctor` output (best-effort)
- [ ] Add tests: manifest parse, discovery de-dup, proxy round-trip (mock upstream)
- [ ] Update SPEC.md (extensions + aggregator)
- [ ] Validate with `openspec validate extension-system`
- [ ] Hubton side: add `src/lib/cockpitContract.ts` + `/cockpit/health` + `/cockpit/manifest` endpoints
- [ ] Hubton side: install manifest under `~/.config/cockpit/extensions/hubton` for this setup
- [ ] Verify: FHL cockpit shows Hubton tabs; hubton standalone still works