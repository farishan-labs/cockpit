# Faris Han Labs — Principles

Canonical source: https://github.com/farishan-labs/principles
This file is **derived** from that repo. Do not edit by hand;
regenerate with `scripts/render.sh`.

These are non-negotiable for the Faris Han Labs. They apply to cockpit
and to every tool built under the lab.

## P-001 — Never reinvent exact wheels

If a proven tool already does it, depend on it, aggregate it, or extend it.
Build only the glue and the control plane.

**Rationale.** Rebuilding an existing wheel wastes effort and inherits none of
the battle-testing. Composition over reinvention keeps the core small and the
surface area honest.

**Examples.** herdr for terminals/agents, pi for harness minimalism, opencode
for coding agents, nakama where it fits, tmux for multiplexing, SQLite for
storage.

**Applies to.** Every tool and feature in the lab.

## P-002 — Zero vendor lock-in

Open formats, portable code, self-hostable. No dependency on a single giant
tech company's proprietary service. Users can fork and run fully offline.

**Rationale.** Proprietary services can change pricing, terms, or disappear.
Open formats and self-hostability keep the user in control and the tool alive.

**Applies to.** Every tool and feature in the lab.

## P-003 — Minimal core

Keep the core tiny (Pi: 4 tools, shortest prompt). Push everything else into
extensions. Draw nothing, guess nothing (herdr-core).

**Rationale.** A small core is auditable, fast to understand, and hard to
break. Capability grows through extensions, not through core bloat.

**Applies to.** Every tool and feature in the lab.

## P-004 — Programmable extensions, not toggles

Extensions are code with a manifest and hook points. The agent can author them
("software that builds itself"). Configuration toggles are not extensions.

**Rationale.** Toggles cap capability at what the author anticipated.
Programmable extensions let users and agents grow the tool without core
changes.

**Applies to.** Every tool and feature in the lab.

## P-005 — Single static binary

One file to install. Go, no CGO, no runtime deps.

**Rationale.** A single static binary removes install friction and makes the
tool trivially portable across machines and CI.

**Applies to.** Every tool and feature in the lab.

## P-006 — Open and shared

Apache-2.0. Others self-host and fork freely.

**Rationale.** Open licensing maximises reuse, contribution, and longevity.
Closed licensing contradicts "zero vendor lock-in".

**Applies to.** Every tool and feature in the lab.

## P-007 — Reuse before build

Every new feature starts with: "what already does this?" If the answer exists,
wrap it.

**Rationale.** Forces a research step before implementation, preventing
silent reinvention and keeping the lab aligned with P-001.

**Applies to.** Every tool and feature in the lab.

## P-008 — Reliability over novelty

Tools that hold up in daily use beat clever ones that don't.

**Rationale.** A tool that breaks daily use is worse than no tool. Prefer
boring, correct, and durable.

**Applies to.** Every tool and feature in the lab.

## P-009 — Small and focused

One tool, one job. Prefer composition over monoliths.

**Rationale.** Focused tools compose; monoliths calcify. Smaller surface means
fewer bugs and clearer ownership.

**Applies to.** Every tool and feature in the lab.

## P-010 — Human sets direction, agents execute

Cockpit drives co-pilots and crews; it never pretends to be the agent.

**Rationale.** Keeping the human as the decision-maker and agents as executors
avoids overreach and keeps control with the operator.

**Applies to.** Every tool and feature in the lab.

## P-011 — Research before creating a new repo

Start research in the research repo; graduate to a dedicated repo only after
research is complete and a named consumer + pain exists. No new repo from a
hunch.

**Rationale.** A new repo is a long-lived commitment. Researching first in a
shared research space keeps experiments cheap and prevents orphaned repos.
Graduation requires evidence, not enthusiasm.

**Applies to.** Every tool and feature in the lab.
