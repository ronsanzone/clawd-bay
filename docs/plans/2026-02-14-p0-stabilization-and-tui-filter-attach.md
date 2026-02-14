# P0 Stabilization + TUI Filter/Attach Reliability Plan
Date: 2026-02-14
Status: Proposed
Owner: TBD

## Summary
This plan fixes all known P0 defects and implements dashboard usability/correctness improvements for:
1. Quick-filter in TUI (substring mode).
2. Reliable attach to selected window from dashboard (window index targeting).

The plan is implementation-ready from a fresh context and includes explicit acceptance criteria, test coverage, and a progress tracking section.

## Goals
- Eliminate current build blocker in `clist`.
- Replace ambiguous session detection logic with deterministic path-based resolution.
- Fix slash-branch workflow resolution issues for `cb claude` and `cb archive`.
- Add TUI quick-filter mode for fast navigation at scale.
- Ensure dashboard attach behavior opens the exact selected target reliably.

## Non-Goals
- Command renames (`clist` -> other names).
- Prompt/init feature additions.
- Full TUI redesign.

## Current Known Defects (Baseline)
- Build failure:
  - `cmd/list_claudes.go` uses `fmt.Printf(o.toString())`, causing vet/build failure in tests.
- Ambiguous session resolution:
  - `cmd/claude.go` and `cmd/archive.go` use substring matching against `filepath.Base(cwd)`.
- Slash-branch regression:
  - Workflows with branch names containing `/` are not consistently discoverable by current matching.
- Dashboard attach target reliability:
  - `cmd/dash.go` currently runs `select-window` in a way that can fail silently and not honor selected target.
- `clist` output quality:
  - Inverted status message logic and repeated output emission.

## Scope
### In Scope
- `cmd/list_claudes.go`
- `cmd/claude.go`
- `cmd/archive.go`
- `cmd/dash.go`
- `cmd/start.go` (guard for empty sanitized branch)
- `internal/tmux/tmux.go`
- `internal/tui/model.go`
- `internal/tui/view.go`
- Tests in `cmd/`, `internal/tmux/`, `internal/tui/`

### Out of Scope
- Breaking public CLI redesign.
- Backfilling unrelated docs/features.

## Design Decisions (Locked)
- P0 scope: all identified P0s in this phase.
- TUI filter strategy: case-insensitive substring match (deterministic order).
- Dashboard window targeting: use `window index` (not name, not tmux window id).
- Preserve existing `cb_` session naming in this release.

---

## Implementation Plan

### Phase 1: Unblock Build + Fix `clist` correctness
Files:
- `cmd/list_claudes.go`

Changes:
- Replace `fmt.Printf(o.toString())` with `fmt.Print(o.toString())`.
- Fix output logic:
  - Avoid re-printing accumulated output repeatedly across repo iterations.
  - Correct status text condition (`isClaudeSession` branch currently inverted).
- Keep behavior minimal and non-breaking.

Acceptance:
- `go test ./...` no longer fails on `cmd/list_claudes.go`.
- `cb clist` output no longer duplicates lines and status text is sensible.

### Phase 2: Deterministic session resolution (path-based)
Files:
- `cmd/claude.go`
- `cmd/archive.go`
- New shared helper in `cmd/` (e.g., `cmd/session_resolver.go`)

Changes:
- Implement helper:
  - `resolveSessionForCWD(tmuxClient, cwd) (sessionName string, worktreePath string, err error)`
- Algorithm:
  1. `ListSessions()`.
  2. For each session, `GetPaneWorkingDir(session)`.
  3. Normalize with `filepath.Abs` + `filepath.Clean`.
  4. Match candidates:
     - Exact match (`cwd == panePath`) preferred.
     - Prefix match (`cwd` inside panePath) allowed.
  5. Tie-breaker: longest panePath wins.
  6. If none, explicit error.
- Replace all substring-based resolution paths in `claude` and `archive`.

Acceptance:
- `cb claude` and `cb archive` resolve correct session from nested dirs.
- No false positive session selection from similar branch names.

### Phase 3: Slash-branch robustness and input guard
Files:
- `cmd/start.go`

Changes:
- After sanitization, fail fast if branch is empty:
  - Return actionable error message.
- Keep current naming format; rely on path-based resolution from Phase 2 for slash-branch compatibility.

Acceptance:
- Empty/invalid sanitized branch input is rejected early.
- Slash-branch workflows can still be resolved by path in `claude`/`archive`.

### Phase 4: Dashboard attach correctness (target exact window)
Files:
- `internal/tui/model.go`
- `cmd/dash.go`
- `internal/tmux/tmux.go`

Changes:
- Add to TUI model:
  - `SelectedWindowIndex int` (default `-1`).
- On window-node selection:
  - Set `SelectedName` and `SelectedWindowIndex`.
- Add tmux helpers:
  - `AttachOrSwitchToSession(session string, inTmux bool) error`
  - `SelectWindow(session string, windowIndex int) error`
- `dash` flow:
  - If window selected (`index >= 0`), call `SelectWindow` before attach/switch.
  - Then call `AttachOrSwitchToSession`.
  - Propagate and wrap all errors with target context.

Acceptance:
- Entering on a specific window in dashboard lands user on that window reliably.
- No silent `select-window` failures.

### Phase 5: TUI quick-filter mode
Files:
- `internal/tui/model.go`
- `internal/tui/view.go`

Changes:
- Add model fields:
  - `FilterMode bool`
  - `FilterQuery string`
  - `FilteredNodes []TreeNode`
  - `FilteredCursor int`
- Key behavior:
  - `/`: enter filter mode.
  - Typing: append query.
  - `backspace`: delete char.
  - `esc`: clear query + exit filter mode.
  - `j/k`: navigate filtered list while filtering.
  - `enter`: act on filtered selection.
- Matching:
  - Case-insensitive `strings.Contains`.
  - Search text:
    - Repo node: repo name
    - Session node: session name + repo name
    - Window node: window name + session name + repo name
  - Preserve original node order (deterministic).
- UI:
  - Show active filter hint in footer/status line.
  - Render filtered list while mode is active.

Acceptance:
- Filter narrows interactively in large trees.
- Enter on filtered node selects intended target.
- Esc restores normal tree navigation.

---

## Testing Plan

### Unit Tests
1. `cmd/session_resolver_test.go` (new)
- exact match
- nested prefix match
- longest-prefix tie-breaker
- no match error
- slash-path cases

2. `cmd/dash_test.go` (new)
- selected session only
- selected window index
- order and args for select + attach/switch
- error propagation from select/attach

3. `internal/tmux/tmux_test.go`
- `SelectWindow` args
- `AttachOrSwitchToSession` path based on `inTmux`

4. `internal/tui/model_test.go`
- filter mode enter/exit
- query typing/backspace
- filtered node ordering
- enter selection on filtered window sets `SelectedWindowIndex`

### Existing Test Suite
- Run `go test ./...`
- Optional: `go test -tags=integration ./... -v`

## Acceptance Criteria (Release Gate)
- All tests pass.
- No compile/vet failures.
- P0 scenarios verified:
  - `clist` compiles and outputs sane rows.
  - `claude`/`archive` resolve session from path robustly.
  - Slash-branch workflows work end-to-end.
- Dashboard improvements verified:
  - Filter works as specified.
  - Attach targets selected window reliably.

## Risks and Mitigations
- Risk: path normalization edge cases on symlinks.
  - Mitigation: use clean+abs consistently; add tests.
- Risk: filter mode introduces navigation regressions.
  - Mitigation: isolate filtered cursor state and test both filtered/unfiltered flows.
- Risk: tmux command ordering differences across environments.
  - Mitigation: abstract sequencing in tmux client helpers and test command invocation.

## Fresh-Context Runbook
1. Read baseline files:
- `cmd/list_claudes.go`
- `cmd/claude.go`
- `cmd/archive.go`
- `cmd/dash.go`
- `cmd/start.go`
- `internal/tui/model.go`
- `internal/tui/view.go`
- `internal/tmux/tmux.go`

2. Implement by phase order above (1 -> 5).

3. Add tests immediately per phase before moving on.

4. Validate with:
- `go test ./...`
- optional integration tests if tmux available

5. Update this plan’s tracking section as tasks move state.

---

## Tracking

### Milestones
- [ ] M1: Build unblocked (`clist` compile + output corrections)
- [ ] M2: Path-based resolver integrated in `claude` + `archive`
- [ ] M3: Slash-branch guard added in `start`
- [ ] M4: Dashboard attach reliability with window index
- [ ] M5: TUI quick-filter mode complete
- [ ] M6: Tests + acceptance complete

### Task Tracker
| ID | Task | Owner | Status | Notes |
|---|---|---|---|---|
| T1 | Fix `fmt.Printf` misuse in `clist` | TBD | Todo | Build blocker |
| T2 | Correct `clist` status/output loop logic | TBD | Todo | No duplicates |
| T3 | Add shared path-based resolver | TBD | Todo | deterministic |
| T4 | Wire resolver into `cb claude` | TBD | Todo | remove substring match |
| T5 | Wire resolver into `cb archive` | TBD | Todo | remove substring match |
| T6 | Add empty-sanitized-branch guard in `start` | TBD | Todo | actionable error |
| T7 | Add `SelectedWindowIndex` to TUI model | TBD | Todo | default -1 |
| T8 | Add tmux attach/switch/select helpers | TBD | Todo | centralize sequencing |
| T9 | Refactor dash attach flow to use helpers | TBD | Todo | no silent failures |
| T10 | Implement TUI filter mode | TBD | Todo | `/`, esc, backspace, enter |
| T11 | Add resolver tests | TBD | Todo | new file |
| T12 | Add dash tests | TBD | Todo | new file |
| T13 | Extend tmux helper tests | TBD | Todo | sequence + args |
| T14 | Extend TUI filter tests | TBD | Todo | behavior coverage |
| T15 | Run `go test ./...` and verify gate | TBD | Todo | release gate |

### Change Log
- 2026-02-14: Initial plan drafted.
