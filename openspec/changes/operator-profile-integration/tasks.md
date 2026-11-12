## 1. Config

- [x] 1.1 Add `UserMDPath string` field to `Config` struct in `internal/config/config.go` with toml tag `user_md_path`.
- [x] 1.2 Default to `~/.config/cockpit/USER.md` when unset; create parent dir in `Load()`.

## 2. Doctor

- [x] 2.1 Implement `runUserMDCheck(path)` helper using `exec.LookPath("usermd-check")`; skip with warning if absent.
- [x] 2.2 Wire into `cockpit doctor` output (pass/fail + cap usage).
- [x] 2.3 Add unit test in `internal/config/config_test.go` for default path.

## 3. Launch injection

- [x] 3.1 In agent launch path, set `USER_MD_PATH` env var on the exec.Cmd.
- [x] 3.2 Ensure `USER.md` symlink in session cwd (create if missing).
- [x] 3.3 Log event to JSONL event log on injection.

## 4. Web UI

- [x] 4.1 Add `/operator` handler in daemon serving profile text + validation status.
- [x] 4.2 Add Operator panel to embedded web UI (htmx).
- [x] 4.3 Wire htmx refresh on doctor result.

## 5. Docs & validation

- [x] 5.1 Update SPEC.md (already cross-referenced).
- [x] 5.2 Run `openspec validate operator-profile-integration`.
- [x] 5.3 Run `go build ./...` and `go test ./...`.