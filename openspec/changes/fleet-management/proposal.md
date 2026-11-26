## Why

Cockpit needs a unified view of managed processes (daemons, menubar apps, jobs)
across the FHL stack. The `processor` registry already tracks processes with
Type/HealthURL metadata, but cockpit has no aggregation endpoint or UI.

## What Changes

- Add `/api/fleet` endpoint that reads the processor registry and probes each
  process's HealthURL, returning JSON with name, type, command, health.
- Add a Fleet panel to the cockpit web UI.
- Document the process metadata schema in OpenSpec.

## Capabilities

### New Capabilities
- `fleet-management`: aggregation and health probing of registered processes.

## Impact

- Affected: `internal/daemon` (route + handler), embedded web UI.
- Soft dep: `processor` registry file at `~/.config/fhl/processor.json`.
