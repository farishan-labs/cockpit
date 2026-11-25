# Design: extension-system

## Approach

Follow cockpit's aggregator principle and reuse the `machines` plugin discovery
pattern. Extensions are discovered from local manifest files; no network
registry. Native extensions are Go interfaces; remote extensions are HTTP
services proxied by the daemon. The host never stores extension data.

### Extension manifest (`cockpit-extension.json`)

```json
{
  "name": "hubton",
  "version": "0.1.0",
  "kind": "remote",
  "provides": ["hubton-projects", "hubton-kanban", "hubton-sponsors"],
  "entry": null,
  "api_base": "http://127.0.0.1:4777",
  "web": "http://127.0.0.1:4777",
  "tabs": [
    { "id": "projects", "title": "Projects", "icon": "folder", "route": "/projects" },
    { "id": "kanban", "title": "Kanban", "icon": "columns", "route": "/kanban" }
  ],
  "commands": [
    { "name": "sync", "description": "Refresh Hubton Universe mirror" }
  ],
  "enabled": true
}
```

For `kind: "native"`, `entry` points at a Go build tag / registered name and
`api_base`/`web` are empty (the extension implements `extensions.Extension`
in-process).

### Discovery

- Base dir: `~/.config/cockpit/extensions/` (each subdirectory is an
  extension; must contain `cockpit-extension.json`).
- Env override: `COCKPIT_EXTENSIONS` (comma-separated paths to extension
  folders or manifests).
- De-dup by manifest `name`; first wins.
- Broken/invalid manifests are skipped with a warning (mirrors machines).

### `extensions.Extension` interface (Go)

```go
type Extension interface {
    Name() string
    Version() string
    Tabs() []Tab
    Commands() []Command
    Health() error
    // Native-only: register in-process HTTP handlers on a mux.
    Mount(mux *http.ServeMux, prefix string) error
}
```

Remote extensions implement the same shape via the HTTP contract instead.

### Remote extension HTTP contract

- `GET /cockpit/health` → 200 OK (body ignored).
- `GET /cockpit/manifest` → JSON manifest (same schema as file, authoritative).
- `POST /cockpit/sync` → trigger refresh (optional; 404 means not supported).

The file manifest's `api_base`/`web` are used for proxying; the remote
manifest endpoint is used to verify reachability and refresh tab lists.

### Proxy layer (daemon)

- For each enabled remote extension, register:
  - `/ext/<name>/` → reverse proxy to `web` base (strip prefix).
  - `/ext/<name>/api/` → reverse proxy to `api_base`.
- Use `net/http/httputil.ReverseProxy` with a custom director that rewrites
  the Host header and strips the prefix.
- Health check on startup + every 30s; mark extension `available` accordingly.
- Extension endpoints are localhost-only (cockpit already binds 127.0.0.1).

### Web UI

- Tab bar reads `/api/extensions` (returns discovered extensions + tabs).
- Each tab renders an iframe (or fetches the extension's web bundle) pointed
  at `/ext/<name>/<route>`.
- No hard-coded Hubton tabs; everything comes from manifests.

### CLI

- `cockpit extension list` — name, kind, version, available, tabs.
- `cockpit extension add <path>` — copy/symlink a manifest folder into
  `~/.config/cockpit/extensions/`.
- `cockpit extension remove <name>` — remove from extensions dir.
- `cockpit extension enable|disable <name>` — toggle (persisted in SQLite).
- `cockpit extension sync <name>` — call `POST /cockpit/sync` (remote) or
  in-process sync (native).

### Storage

- SQLite table `extensions(name PK, kind, version, enabled, available,
  last_sync, last_seen)`.
- No extension payload data; extensions own their stores (e.g. hubton's
  cache.db).

### Integration with hubton-cockpit

- hubton-cockpit keeps its Bun server + sources. It gains a small
  `src/lib/cockpitContract.ts` exposing `/cockpit/health` and
  `/cockpit/manifest` (tabs derived from its existing routes).
- A manifest folder `~/.config/cockpit/extensions/hubton` (symlink to a
  `cockpit-extension.json` in the hubton repo or a small extension package)
  is installed by default for this setup.
- hubton-cockpit's generic web shell becomes optional (standalone fallback);
  when run as an extension, FHL cockpit's web shell hosts its tabs.

### Risks / open questions

- Iframe vs. federated modules: iframe is simplest and preserves extension
  isolation; federated modules allow deeper integration later. Start with
  iframe.
- CORS/cookies: extensions are same-origin via the proxy, so no CORS needed.
- Port collisions: remote extensions choose their own ports; manifest's
  `api_base`/`web` must match. Document in manifest schema.
- Version skew: host checks manifest `version` but does not enforce; log
  mismatches as warnings.
- Trust: manifests are local files; same trust model as `tools.toml`. No
  signature verification in v1.