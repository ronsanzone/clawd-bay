# ClawdBay TUI Add Action Design (Session/Window + Name Popup)
Date: 2026-02-14
Status: Proposed
Owner: TBD

## Summary
Add a single `a` action in dashboard worktree mode that supports two add flows with an inline name popup:
1. Create tmux session when cursor is on repo/worktree rows.
2. Create tmux window when cursor is on session/window rows.

This design replaces the current `c` Claude-only shortcut and keeps users in the dashboard after creation.

## Locked Requirements
1. Remove `c` add-Claude shortcut from TUI; `a` is the only add action.
2. `a` on repo row creates a session in that repo's main worktree.
3. `a` on worktree row creates a session in that worktree.
4. `a` on session row creates a plain tmux window in that session.
5. `a` on window row creates a plain tmux window in that window's parent session.
6. After create, stay in dashboard (no attach/switch).
7. Session names must be sanitized and auto-prefixed with `cb_`.
8. Duplicate names must auto-suffix (`-2`, `-3`, etc.) for both sessions and windows.

## Non-Goals
1. Agent launch integration for new windows (plain shell only for now).
2. Agents-mode add behavior.
3. Changing enter-to-attach behavior.
4. Replacing existing filter semantics.

## Current Baseline
1. Add/create path currently exists only for Claude windows via `c` and only on session/window rows in `internal/tui/model.go`.
2. There is no popup/modal state in the TUI model.
3. Footer key hints still advertise `c claude` in worktree mode.
4. Discovery grouping prefers pinned `@cb_home_path`; therefore session creation from TUI must set this option to keep grouping stable.

Reference files:
- `internal/tui/model.go`
- `internal/tui/view.go`
- `internal/discovery/discovery.go`
- `internal/tmux/tmux.go`

## User Experience Specification

### Keymap
1. Replace `c` with `a` in worktree mode.
2. Keep `a` disabled in agents mode.
3. While filter mode is active, `a` remains text input (no add action); user must exit filter (`esc`) to add.

### Action by Cursor Node Type
1. `NodeRepo` -> open popup to create session in repo main worktree.
2. `NodeWorktree` -> open popup to create session in selected worktree.
3. `NodeSession` -> open popup to create window in selected session.
4. `NodeWindow` -> open popup to create window in parent session.
5. `NodeAgentWindow` -> no-op.

### Popup Behavior
1. Small centered popup with title, target, and single-line name input.
2. `enter` submits.
3. `esc` cancels.
4. `backspace` deletes.
5. Runes append input.
6. Inline validation errors are shown in popup and do not close it.

### Success/Failure Feedback
1. On submit: set transient status message `Creating ...`.
2. On success: set `Session created: <name>` or `Window created: <name>` and refresh.
3. On failure: set `Error: ...` and refresh.

## Technical Design

### Model Changes (`internal/tui/model.go`)

### New Types
```go
type AddKind int

const (
    AddKindNone AddKind = iota
    AddKindSession
    AddKindWindow
)

type AddDialogState struct {
    Active      bool
    Kind        AddKind
    Input       string
    Error       string
    RepoIndex   int
    WorktreeIdx int
    SessionName string
}

type addResultMsg struct {
    Kind   AddKind
    Name   string
    Target string
    Err    error
}
```

### Model Fields
```go
AddDialog AddDialogState
```

### Key Handling Order
1. If `AddDialog.Active`: handle dialog-only keys and return.
2. Else if `FilterMode`: existing filter handling unchanged.
3. Else normal mode handling; add `case "a": return m.handleOpenAddDialog()`.

Rationale: modal interaction must be deterministic and prevent conflicting navigation.

### Open Dialog Target Resolution
Add helper:
```go
func (m Model) openAddDialogForNode(node TreeNode) (Model, tea.Cmd)
```

Resolution rules:
1. `NodeRepo`: locate main worktree (`IsMainRepo == true`) in repo's worktrees.
2. `NodeWorktree`: use selected worktree.
3. `NodeSession` / `NodeWindow`: use parent session name.
4. Otherwise: no-op.

Failure to find main worktree should set `StatusMsg` error and not open dialog.

### Submit Flow
Add helper:
```go
func (m Model) submitAddDialog() (tea.Model, tea.Cmd)
```

Flow:
1. Sanitize input.
2. Validate non-empty.
3. Resolve final unique name via suffixing.
4. Dispatch async tmux command and close popup.

#### Session Create Command
1. Build final tmux session name:
   1. `sanitize` input.
   2. Ensure prefix `cb_`.
   3. Deduplicate against existing sessions.
2. Resolve worktree path from selected repo/worktree indices.
3. Call `TmuxClient.CreateSession(name, worktreePath)`.
4. Canonicalize worktree path with `config.CanonicalPath`.
5. Call `TmuxClient.SetSessionOption(name, tmux.SessionOptionHomePath, canonicalPath)`.
6. Return `addResultMsg`.

#### Window Create Command
1. Build final window name:
   1. `sanitize` input.
   2. Deduplicate against windows in target session.
2. Call `TmuxClient.CreateWindow(sessionName, windowName, "")`.
3. Return `addResultMsg`.

### Name Sanitization and Dedupe
Add helpers in `internal/tui/model.go` (or split to small `internal/tui/naming.go` if preferred):
```go
func sanitizeAddName(raw string) string
func ensureSessionPrefix(name string) string
func uniquifyName(base string, exists func(string) bool) string
```

Sanitize rules:
1. Lowercase.
2. Convert spaces to `-`.
3. Keep `[a-z0-9-_/]`.
4. Collapse repeated `-`.
5. Trim leading/trailing `-` and `/`.

Prefix rules:
1. Session names must start with `cb_`.
2. If already prefixed, keep as-is.
3. If sanitized result becomes empty or only `cb_`, validation error.

Dedupe rules:
1. First candidate is base name.
2. If taken, try `<base>-2`, `<base>-3`, ... until free.

Session existence source:
1. Prefer `TmuxClient.ListSessions()` to cover all managed sessions, not only visible tree.

Window existence source:
1. Current session windows from model node snapshot (best effort).
2. If race causes tmux collision anyway, surface tmux error in status message.

### View Changes (`internal/tui/view.go`)

### Footer Updates
Replace create hints:
1. Repo/worktree rows: include `a add session`.
2. Session/window rows: include `a add window`.
3. Remove `c claude` references.

### Popup Rendering
Constraint: lipgloss `v1.1.0` has no `PlaceOverlay`, so use modal-style composition in tree area.

Implementation approach:
1. Keep existing frame rendering.
2. When `AddDialog.Active`, render tree body as:
   1. top spacer lines,
   2. centered popup box,
   3. bottom spacer lines.
3. Popup box content:
   1. Title: `Add Session` or `Add Window`.
   2. Target line: worktree path or session name.
   3. Input line: `name: <current input>`.
   4. Hint line: `enter create  esc cancel`.
   5. Optional error line.

Keep popup width fixed with min/max clamping (for example 44-64 columns) based on frame width.

## tmux/Discovery Integration Notes
1. Session creation must set `@cb_home_path` after `CreateSession` to preserve worktree placement in discovery.
2. Window creation intentionally does not launch agents.
3. No attach/switch should be triggered by add flow.

## Testing Plan

### Model Tests (`internal/tui/model_test.go`)
1. `a` on `NodeRepo` opens session dialog and targets main worktree.
2. `a` on `NodeWorktree` opens session dialog for selected worktree.
3. `a` on `NodeSession` opens window dialog for selected session.
4. `a` on `NodeWindow` opens window dialog for parent session.
5. Dialog input handling: rune append, backspace, esc cancel.
6. Submit validation failure on empty sanitized input keeps dialog open with error.
7. `sanitizeAddName` table tests.
8. `ensureSessionPrefix` tests.
9. `uniquifyName` tests for `-2`, `-3` sequence.
10. Agents mode ignores `a`.

### View Tests (`internal/tui/view_test.go`)
1. Footer for repo/worktree includes `a add session`.
2. Footer for session/window includes `a add window`.
3. Footer does not include `c claude`.
4. Popup is rendered when `AddDialog.Active` with title/input/hints.

### Optional Integration Test
If environment allows tmux integration:
1. Start dashboard with a seeded project.
2. Trigger add session from repo row, confirm new `cb_` session exists.
3. Trigger add window from session row, confirm new window exists.

## Implementation Sequence
1. Introduce dialog state and add-kind/result msg types.
2. Wire key handling (`a` and dialog mode).
3. Implement target resolution by node type, including repo->main-worktree lookup.
4. Implement sanitize/prefix/dedupe helpers with unit tests.
5. Implement async create commands and result handling.
6. Update footer and popup rendering.
7. Run `make test` and adjust tests for new footer strings.

## Acceptance Criteria
1. Pressing `a` in worktree mode opens name popup for valid target rows.
2. Repo/worktree rows create tmux sessions in correct worktree path.
3. Session/window rows create plain tmux windows in correct session.
4. Session names are sanitized, auto-prefixed with `cb_`, and deduped.
5. Window names are sanitized and deduped.
6. Dashboard remains open after successful create.
7. Footer/help text reflects `a` behavior; no stale `c claude` hints remain.
8. `make test` passes.

## Risks and Mitigations
1. Risk: popup rendering complexity without overlay primitive.
   Mitigation: modal tree replacement, not ANSI substring overlay.
2. Risk: stale local window list can miss cross-client window creations.
   Mitigation: dedupe best effort + explicit tmux error path.
3. Risk: name sanitization drift vs `cmd/start.go` branch sanitization.
   Mitigation: document rules and add table-driven tests.

## Fresh Context Runbook
1. Read:
   1. `internal/tui/model.go`
   2. `internal/tui/view.go`
   3. `internal/discovery/discovery.go`
   4. `internal/tmux/tmux.go`
2. Implement model state + key routing first.
3. Add pure naming helper tests before async creation wiring.
4. Add popup rendering and footer updates.
5. Run `make test`.
6. Manually verify with `make run` + `cb dash`.
