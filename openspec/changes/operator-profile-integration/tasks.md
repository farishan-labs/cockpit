## 1. Config

- [ ] 1.1 Add `UserMDPath string` field to `Config` struct in `internal/config/config.go` with toml tag `user_md_path`.
- [ ] 1.2 Default to `~/.config/cockpit/USER.md` when unset; create parent dir in `Load()`.

## 2. Doctor

- [ ] 2.1 Implement `runUserMDCheck(path)` helper using `exec.LookPath("usermd-check")`; skip with warning if absent.
- [ ] 2.2 Wire into `cockpit doctor` output (pass/fail + cap usage).
- [ ] 2.3 Add unit test in `internal/config/config_test.go` for default path.

## 3. Launch injection

- [ ] 3.1 In agent launch path, set `USER_MD_PATH` env var on the exec.Cmd.
- [ ] 3.2 Ensure `USER.md` symlink in session cwd (create if missing).
- [ ] 3.3 Log event to JSONL event log on injection.

## 4. Web UI

- [ ] 4.1 Add `/operator` handler in daemon serving profile text + validation status.
- [ ] 4.2 Add Operator panel to embedded web UI (htmx).
- [ ] 4.3 Wire htmx refresh on doctor result.

## 5. Docs & validation

- [ ] 5.1 Update SPEC.md (already cross-referenced).
- [ ] 5.2 Run `openspec validate operator-profile-integration`.
- [ ] 5.3 Run `go build ./...` and `go test ./...`.