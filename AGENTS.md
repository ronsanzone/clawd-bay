# AGENTS.md

## Purpose
This file gives coding agents the minimum high-value context needed to work safely and effectively in this repository.

## Project Overview
- ClawdBay is a Go CLI/TUI for managing multi-session coding-agent workflows in tmux.
- Core flow: create worktree + tmux session (`cb start`), monitor/attach (`cb` or `cb dash`), cleanup (`cb archive`).
- Runtime dependencies: Go 1.25.7, tmux 3.x+, and a coding agent CLI (`claude`, `codex`, `open-code`) for agent-driven pane workflows.
- The system is stateless by design: session/workflow state is derived from tmux at runtime.

## Repository Map
- `/main.go`: program entrypoint.
- `/cmd`: Cobra command layer (`start`, `dash`, `list`, `archive`, `clist`, resolver helpers).
- `/internal/tmux`: tmux command client, session/window parsing, agent/status detection.
- `/internal/tui`: Bubble Tea model/view/theme for dashboard UX.
- `/internal/config`: config path management.
- `/internal/logging`: structured logging setup.
- `/integration_test.go`: end-to-end CLI tests (build tag: `integration`).
- `/docs/plans`: design/implementation notes; useful context, but code + tests are authoritative.

## Source-of-Truth Rules
- Trust code and tests first.
- Current command surface from source: `cb start`, `cb dash` (default `cb`), `cb list`, `cb archive`, `cb clist`.

## Critical Invariants
- Tmux session names for managed workflows must be prefixed with `cb_`.
- Worktrees live under `<repo>/.worktrees/<repo>-<branch>`.
- Session resolution from cwd must remain path-based (`resolveSessionForCWD`), not substring matching.
- Dashboard attach semantics:
  - If a window is selected, select by window index first.
  - Then attach/switch session based on whether inside tmux.
- Status rollup priority must remain: `WORKING > WAITING > IDLE > DONE`.
- Agent activity detection priority must remain: busy indicators before prompt indicators.

## Build, Run, and Test Commands
- Build: `make build`
- Run app: `make run`
- Debug run: `make debug`
- CLI help smoke check: `make smoke`
- Unit tests (required): `make test`
- Integration tests (optional, environment-dependent): `make test-integration`
  - Requires tmux + writable git refs.
  - Creates/removes worktrees, branches, and tmux sessions.
- Full local verify target: `make verify` (runs `test` + `lint`; lint requires `golangci-lint` installed).

## Code Change Expectations
- Always add or update tests with behavior changes.
- Touching command logic in `/cmd`:
  - Update/add `cmd/*_test.go`.
  - If command surface/help changes, update integration assertions in `/integration_test.go`.
- Touching tmux parsing or status heuristics:
  - Update `/internal/tmux/tmux_test.go` with table-driven cases.
- Touching TUI navigation/rendering:
  - Update `/internal/tui/model_test.go` and `/internal/tui/view_test.go`.
- If user-facing behavior changes, keep docs in sync (`/README.md`, `/INSTALL.md`).

## Coding Conventions
- Keep changes idiomatic Go and gofmt-formatted.
- Prefer small focused helpers and table-driven tests.
- Wrap errors with context using `%w`.
- Keep tmux interactions centralized in `/internal/tmux` where possible.
- Avoid introducing global mutable state outside Cobra flag wiring.

## Safety and Operational Guardrails
- Assume tmux/git commands have side effects.
- `cb archive` destroys session + worktree (branch preserved); keep confirmations for destructive operations.
- Prefer `cb start --detach` in non-interactive or test flows to avoid TTY attach issues.
- Do not run destructive cleanup commands unless explicitly required for the task.

## Quick Validation Checklist for Agent Changes
1. `make verify` It's important that this is ran on every change before claiming complete.
2. If command behavior changed: `make smoke` and check affected command help manually.
3. If tmux/TUI behavior changed and environment allows: `make test-integration`.
