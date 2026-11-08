# Cockpit

The first thing installed and opened when operating the **Faris Han Labs**.

A **human+AI control plane**: one place to see the dashboard, control things
(install/enable/disable tools), and drive co-pilots and crews (agents).

Cockpit is an **aggregator, not a replacement**. It reuses, aggregates, and
extends existing tooling — herdr, pi, opencode, nakama, tmux, SQLite — and
never reinvents an exact wheel. Zero vendor lock-in.

## Status

M0 — foundation. See [PLAN.md](./PLAN.md) and [SPEC.md](./SPEC.md).

## Principles

See [PRINCIPLES.md](./PRINCIPLES.md). The big ones:

- Never reinvent exact wheels.
- Zero vendor lock-in.
- Minimal core; programmable extensions.
- Single static binary.
- Open and shared.

## Quick start (dev)

```sh
go build ./...
./cockpit version
./cockpit doctor   # (M1)
./cockpit serve    # (M1)
```

## Layout

```
cmd/cockpit/        # CLI entrypoint (cobra)
internal/version/   # build version
SPEC.md             # specification
PLAN.md             # milestones
PRINCIPLES.md       # non-negotiables
```

## License

Apache-2.0 (to be added).
