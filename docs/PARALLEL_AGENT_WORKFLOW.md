# Parallel Agent Workflow (Draft ADR)

## Context
Cockpit and related FHL repos need faster delivery. Single-agent sequential work is slow; naive parallel edits cause conflicts.

## Decision
Adopt worktree-per-agent with contract-first, ownership-scoped tasks and test-gated merges.

## Workflow
1. **Contract first**: Write/lock API/schema/CLI contract (e.g., `/api/machines` shape, DB tables) in a spec or `CONTRACT.md` before spawning agents.
2. **Provision worktrees**: Each agent gets an isolated checkout via T3 worktree (base ref from origin/main). No shared working tree.
3. **Ownership scopes**: Assign non-overlapping file surfaces (adapters, UI, CLI, tests). One owner per contested file.
4. **Agent execution**: Agent implements its scope, runs tests/lint/typecheck, commits with author Faris Han + Co-authored-by FHL Director.
5. **Coordinator merge**: Sequentially merge agent branches; resolve conflicts only at boundaries; re-run full test suite after each merge.
6. **Environment isolation**: Per-worktree ports/logs to avoid dev-server clashes.
7. **Observability**: agent-time per task; OptMem note per meaningful decision; T3 thread status for tracking.

## Safety Rules
- No two agents edit the same file.
- Worktree must be clean before merge.
- Failed tests block merge.
- Never force-push shared branches.

## Tooling Requirements
- Fix `delegate_task` worktree commit resolution or standardize `t3_thread_launch` with `workspaceStrategy: worktree`.
- Add a `worktree status` check in CI/pre-merge.

## Status
Draft — not yet adopted.