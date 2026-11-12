## Purpose

Lets cockpit (the control plane) consume, validate, and inject the operator profile
(`USER.md`) into agent sessions it launches, so every agent knows who the operator is.

## ADDED Requirements

### Requirement: Configured operator profile path
The system SHALL read `user_md_path` from `~/.config/cockpit/config.toml`. When unset,
it SHALL default to `~/.config/cockpit/USER.md`.

#### Scenario: Default path used
- **WHEN** config has no `user_md_path` key
- **THEN** cockpit uses `~/.config/cockpit/USER.md`

#### Scenario: Custom path used
- **WHEN** config sets `user_md_path = "/x/USER.md"`
- **THEN** cockpit uses `/x/USER.md`

### Requirement: Doctor validates operator profile
The system SHALL run `usermd-check` on the configured profile during `cockpit doctor`
and report pass/fail plus character-cap usage. If `usermd-check` is not on PATH,
doctor SHALL skip with a warning rather than fail.

#### Scenario: Valid profile
- **WHEN** `cockpit doctor` runs and the profile passes all checks
- **THEN** doctor prints a success line including cap usage

#### Scenario: Invalid profile
- **WHEN** `cockpit doctor` runs and the profile fails a check
- **THEN** doctor prints the failing check and exits non-zero

#### Scenario: Validator absent
- **WHEN** `usermd-check` is not installed
- **THEN** doctor prints a warning and continues

### Requirement: Profile injected into launched agents
When launching an agent session, cockpit SHALL set the `USER_MD_PATH` environment
variable to the configured profile path and ensure a `USER.md` file is present in the
session's working directory (symlink if missing).

#### Scenario: Launch sets env
- **WHEN** an agent is launched via cockpit
- **THEN** the process environment contains `USER_MD_PATH` pointing at the profile

#### Scenario: Missing file in cwd
- **WHEN** the session cwd has no `USER.md`
- **THEN** cockpit creates a symlink to the configured profile before launch

### Requirement: Operator panel in web UI
The web UI SHALL display the current operator profile text, validation status, cap
usage, and last audit date in an "Operator" panel.

#### Scenario: Panel renders
- **WHEN** the web UI loads
- **THEN** the Operator panel shows profile text and validation status