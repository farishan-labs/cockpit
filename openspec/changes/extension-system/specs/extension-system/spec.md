## Purpose

Lets FHL cockpit act as the base control plane that hosts optional extensions.
An extension contributes tabs, commands, and (for remote ones) its own data
store; the host discovers, proxies, and mounts them without storing their data.

## ADDED Requirements

### Requirement: Manifest discovery
The system SHALL discover extensions from `~/.config/cockpit/extensions/`
(each subdirectory containing `cockpit-extension.json`) and from paths listed in
the `COCKPIT_EXTENSIONS` environment variable (comma-separated). Duplicate
names SHALL be de-duplicated (first wins). Invalid manifests SHALL be skipped
with a warning.

#### Scenario: Extensions dir present
- **WHEN** `~/.config/cockpit/extensions/hubton/cockpit-extension.json` exists
- **THEN** the hubton extension is discovered and listed

#### Scenario: Env override
- **WHEN** `COCKPIT_EXTENSIONS=/opt/ext/foo` is set
- **THEN** the extension at `/opt/ext/foo` is discovered

#### Scenario: Invalid manifest skipped
- **WHEN** a manifest is missing the required `name` field
- **THEN** it is skipped with a warning and does not crash discovery

### Requirement: Manifest schema
A manifest SHALL require `name` (string). It MAY include `version`, `kind`
(`native` or `remote`, default `remote`), `provides` (string array), `entry`,
`api_base`, `web`, `tabs` (array of `{id,title,icon,route}`), `commands`
(array of `{name,description}`), and `enabled` (boolean, default true).

#### Scenario: Minimal manifest
- **WHEN** a manifest contains only `name`
- **THEN** it is valid and defaults apply

### Requirement: Remote extension HTTP contract
A remote extension SHALL expose `GET /cockpit/health` returning 2xx when
healthy and `GET /cockpit/manifest` returning its manifest JSON. It MAY expose
`POST /cockpit/sync`.

#### Scenario: Health check
- **WHEN** the host pings a remote extension
- **THEN** it calls `GET /cockpit/health` and marks `available` based on 2xx

### Requirement: Reverse proxy
For each enabled remote extension, the host SHALL mount its web bundle at
`/ext/<name>/` and its API at `/ext/<name>/api/`, reverse-proxying to the
manifest's `web` and `api_base` respectively.

#### Scenario: Web request proxied
- **WHEN** a request hits `/ext/hubton/projects`
- **THEN** it is forwarded to `<web>/projects`

### Requirement: Dynamic tabs
The embedded web UI SHALL fetch `/api/extensions` and render one tab per
extension tab entry, loading its route via the proxy. No tabs SHALL be
hardcoded for a specific extension.

#### Scenario: Hubton tabs appear
- **WHEN** the hubton extension is enabled and available
- **THEN** its declared tabs appear in the UI

### Requirement: Enable/disable persistence
The host SHALL persist each extension's enabled state and last-sync timestamp
in SQLite and honor it across restarts.

#### Scenario: Disable persists
- **WHEN** `cockpit extension disable hubton` is run then the daemon restarts
- **THEN** hubton remains disabled

### Requirement: CLI management
The system SHALL provide `cockpit extension list|add|remove|enable|disable|sync`.

#### Scenario: List
- **WHEN** `cockpit extension list` runs
- **THEN** it prints name, kind, version, available, and tabs for each extension

### Requirement: Doctor integration
`cockpit doctor` SHALL list discovered extensions with availability
(best-effort; missing/unavailable extensions produce warnings, not failures).

#### Scenario: Doctor with extension
- **WHEN** `cockpit doctor` runs with hubton available
- **THEN** it prints hubton as available

### Requirement: Standalone compatibility
hubton-cockpit SHALL continue to run standalone (`cockpit serve` in its own
repo) regardless of whether it is installed as an extension.

#### Scenario: Standalone still works
- **WHEN** hubton-cockpit is run directly
- **THEN** its dashboard serves on its own port without FHL cockpit