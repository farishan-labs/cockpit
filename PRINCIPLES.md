# Cockpit — Principles

These are non-negotiable for the Faris Han Labs. They apply to cockpit and to
every tool built under the lab.

1. **Never reinvent exact wheels.** If a proven tool already does it, depend
   on it, aggregate it, or extend it. Build only the glue and the control
   plane. (herdr for terminals/agents, pi for harness minimalism, opencode
   for coding agents, nakama where it fits, tmux for multiplexing, SQLite for
   storage.)
2. **Zero vendor lock-in.** Open formats, portable code, self-hostable. No
   dependency on a single giant tech company's proprietary service. Users can
   fork and run fully offline.
3. **Minimal core.** Keep the core tiny (Pi: 4 tools, shortest prompt). Push
   everything else into extensions. Draw nothing, guess nothing (herdr-core).
4. **Programmable extensions, not toggles.** Extensions are code with a
   manifest and hook points. The agent can author them ("software that builds
   itself").
5. **Single static binary.** One file to install. Go, no CGO, no runtime deps.
6. **Open and shared.** Apache-2.0. Others self-host and fork freely.
7. **Reuse before build.** Every new feature starts with: "what already does
   this?" If the answer exists, wrap it.
8. **Reliability over novelty.** Tools that hold up in daily use beat clever
   ones that don't.
9. **Small and focused.** One tool, one job. Prefer composition over
   monoliths.
10. **Human sets direction, agents execute.** Cockpit drives co-pilots and
    crews; it never pretends to be the agent.
