## Why

FHL cockpit is the intended base control plane, but Hubton-specific
functionality currently lives in a separate product (`farishubton/hubton-cockpit`)
that duplicates generic cockpit concerns (its own web shell, server, data dir).
To make FHL cockpit the single "first thing opened" control plane, Hubton
functionality must become an **extension** of FHL cockpit rather than a parallel
cockpit. Today FHL cockpit has no extension mechanism at all — no plugin
discovery, no manifest, no way to mount external tabs or proxy external APIs.

## What Changes

- Introduce an **Extension** concept in FHL cockpit: a named, versioned unit
  that contributes tabs, CLI commands, and (optionally) its own data store.
- Add **discovery** of extensions from `~/.config/cockpit/extensions/`
  (directory of extension folders) and the `COCKPIT_EXTENSIONS` env var
  (comma-separated paths), each folder containing a `cockpit-extension.json`
  manifest. Mirrors the `machines` plugin discovery pattern.
- Define a **manifest schema** (`name`, `version`, `provides`, `kind`
  `native|remote`, `entry`, `web` bundle path/URL, `api_base` URL, `tabs`,
  `commands`).
- Support two kinds:
  - **Native (Go)**: a Go package implementing the `extensions.Extension`
    interface, compiled in via build tags or a registered list.
  - **Remote (HTTP)**: an external service (e.g. hubton-cockpit on Bun)
    exposing a small HTTP contract; FHL cockpit proxies its API and mounts its
    web bundle.
- Define the **remote extension HTTP contract**: `GET /cockpit/health`,
  `GET /cockpit/manifest`, `POST /cockpit/sync` (optional).
- Add a **proxy layer** in the daemon: mount each remote extension's API under
  `/ext/<name>/api/...` and its web bundle under `/ext/<name>/...`, forwarding
  requests and injecting the manifest's `api_base`.
- Add **dynamic tab mounting** in the embedded web UI: render tab slots from
  discovered extension manifests instead of hardcoding tabs.
- Add CLI: `cockpit extension list|add|remove|enable|disable` and
  `cockpit extension sync <name>`.
- Add `extensions` to `/api/adapters`-style aggregation and `cockpit doctor`
  (best-effort, skip missing extensions).
- Storage: persist enabled/disabled state and last-sync timestamps in SQLite;
  no extension data stored by the host (extensions own their data).

## Capabilities

### New Capabilities
- `extensions`: discovery, manifest loading, enable/disable, lifecycle.
- `extension-proxy`: reverse proxy for remote extension APIs and web bundles.
- `extension-tabs`: dynamic web UI tab mounting from manifests.

### Modified Capabilities
- `adapters`: extensions appear alongside adapters in `/api/adapters` and
  doctor output (best-effort).
- `daemon`: routes gain `/ext/<name>/*` proxy handlers and `/api/extensions`.
- `web-ui`: tab bar becomes data-driven from extension manifests.

## Impact

- Affected: `internal/daemon` (routes + proxy), `internal/config`, `internal/storage`,
  `cmd/cockpit` (new subcommands), embedded web UI, `SPEC.md`.
- New package: `internal/extensions`.
- New external (soft) dep: none for native; remote extensions are separate
  processes (e.g. hubton-cockpit) that must be installed/running.
- Non-goals: no rewriting Hubton's Bun sources into Go; no breaking
  hubton-cockpit standalone mode during transition; no vendor lock-in (any
  extension is just an HTTP service + manifest); no central registry/marketplace.
- Security: remote extensions run as the operator's user; proxy only binds
  localhost; extension manifests are trusted local files (same trust as
  `tools.toml`). Pairing tokens/secrets stay in the extension's own store.
- Compatibility: hubton-cockpit keeps working standalone; becomes an
  extension only when its manifest is installed under
  `~/.config/cockpit/extensions/hubton`.