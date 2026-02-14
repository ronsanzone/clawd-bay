# ClawdBay — Sticky Session Home + Mixed Window Indicator Design

**Date:** 2026-02-14
**Status:** Draft

## Summary

This design refines configured project/worktree discovery by making session placement stable and explicit.

1. Each managed `cb_` tmux session is pinned to a canonical home path via tmux session metadata.
2. Session grouping in `cb dash` and `cb list` uses the pinned home path only.
3. Sessions without pinned home metadata are grouped under `(main repo)` for their configured project.
4. Sessions with windows in multiple roots/worktrees are marked with a `[MIXED]` indicator.

## Decision Update (vs earlier design)

Replaces previous fallback behavior:

- **Removed:** fallback to `session:0` pane cwd for unpinned sessions.
- **New:** if a session has no valid pinned home metadata, it is placed in `(main repo)`.

This avoids location “jumping” caused by pane cwd drift.

## Problem

Current session-to-worktree mapping can shift when pane cwd changes. A session can appear to move between a worktree and `(main repo)` even though user intent did not change.

## Goals

1. Stable session grouping across refreshes and window navigation.
2. User-visible signal when a session spans multiple directories/worktrees.
3. Keep configured-project scoping unchanged.
4. Keep attach semantics and status rollup semantics unchanged.

## Non-Goals

1. Per-window re-parenting of sessions in the tree.
2. Automatic migration of legacy sessions.
3. Changes to `cb clist` scoping.

## Metadata Contract

Each managed session stores:

- `@cb_home_path = <canonical absolute path>`

Set at `cb start` when the session is created.

## Discovery Rules

For each session:

1. Read `@cb_home_path`.
2. If present and canonicalizable, map session to configured project/worktree by longest path match.
3. If missing or invalid, place under `(main repo)` of the owning configured project.
4. If no owning configured project exists, exclude session from `cb dash` / `cb list`.

### No fallback

There is no fallback to live pane cwd for session placement.

## Mixed Indicator Rules

After session placement is chosen:

1. Inspect each window cwd.
2. Canonicalize and map each window to project/worktree.
3. If any window maps outside the session’s assigned worktree, mark session `Mixed=true` and increment `MixedCount`.

Rendering:

- Dashboard session row shows `[MIXED]` when `Mixed=true`.
- `cb list` may append a mixed note (recommended for parity).

## UX Behavior

### `cb dash`

- Session remains under pinned home worktree/main repo regardless of transient pane cwd changes.
- `[MIXED]` appears when windows span locations.
- Inactive worktrees remain visible.

### `cb list`

- Same placement logic as dashboard.
- Sessions shown under pinned home worktree or `(main repo)` when unpinned.

### `cb start`

- After session creation, set `@cb_home_path` to canonical worktree path.
- If setting fails, warn but do not fail start flow.

## Data Model Updates

`SessionNode` gains:

```go
type SessionNode struct {
    Name       string
    Status     tmux.Status
    Windows    []tmux.Window
    Mixed      bool
    MixedCount int
}
```

## Tmux API Additions

Add helpers in `internal/tmux`:

1. `SetSessionOption(session, key, value string) error`
2. `GetSessionOption(session, key string) (string, error)`
3. `GetWindowWorkingDir(session string, windowIndex int) string`

## Test Strategy

1. `internal/tmux/tmux_test.go`
   - session option set/get
   - window cwd lookup
2. `cmd/start_test.go`
   - `@cb_home_path` write attempted
   - non-fatal warning path
3. `internal/discovery/discovery_test.go`
   - pinned session placement stability
   - untagged session -> `(main repo)` placement
   - mixed window detection
4. `internal/tui/view_test.go`
   - `[MIXED]` rendering
5. `cmd/list_test.go`
   - mixed note parity (if enabled)

## Acceptance Criteria

1. Session placement no longer changes when users `cd` within panes.
2. Untagged sessions are grouped under `(main repo)`.
3. `[MIXED]` appears for sessions with windows outside assigned worktree.
4. Existing status priority and attach behavior remain unchanged.
5. `cb clist` behavior remains unchanged.
