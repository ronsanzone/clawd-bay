# ClawdBay v0.2 — Simplification & Cross-Repo Design

**Date:** 2026-02-05
**Status:** Draft

## Problem Statement

ClawdBay v0.1 has two conceptual layers pulling in different directions:

1. **Session orchestration** (worktrees, tmux, navigation, lifecycle) — the daily-driver functionality
2. **Prompt templating** (embedded templates, `cb prompt`, `cb init`, `.prompts/`) — superseded by Claude skills/commands

Additionally, the TUI dashboard is single-repo scoped and several key actions (n, c, x) are displayed in the footer but not implemented.

## Goals

1. **Remove the prompt/template system** — dead weight now that skills handle this
2. **Add cross-repo session discovery** — unified dashboard showing all sessions across all repos
3. **Simplify the command surface** — merge `cb start` with Claude window creation
4. **Fix TUI key actions** — make all displayed actions functional
5. **Change worktree location** — use `.worktrees/` inside the repo instead of sibling directories
6. **Add auto-refresh** — dashboard stays current without manual intervention

## Non-Goals

- State files or databases (remain fully stateless, derive everything from tmux)
- Prompt generation or template management
- Autonomous Claude session orchestration (queued prompts, etc.)

---

## Command Surface

### `cb` / `cb dash` — Interactive Dashboard (default)

Opens the TUI dashboard. Shows all `cb_` tmux sessions across all repos in a three-level hierarchy.

### `cb start <branch>` — Create Working Session

Creates a complete working session in one shot:

1. Sanitize branch name (existing logic)
2. Create git worktree at `.worktrees/<project>-<branch>/`
3. Add `.worktrees/` to `.gitignore` if not already present
4. Create tmux session `cb_<branch>` with working directory set to the worktree
5. Create a `claude` window in that session, launch `claude` in it
6. Attach or switch to the session (unless `--detach`)

**Flags:**
- `--detach` — create without attaching

### `cb claude [--name NAME]` — Add Claude Window

Adds a Claude window to the current `cb_` session.

- Detects current session from `$TMUX` environment
- Creates window named `claude` or `claude:<name>`
- Launches `claude` in the new window

The `--prompt` flag is removed (no more template system).

### `cb list` — Non-Interactive List

Text output showing all sessions grouped by repo:

```
claude-essentials
  cb_feat-auth      2 windows  (WORKING)
  cb_refactor-tui   1 window   (IDLE)
my-other-project
  cb_fix-login      1 window   (DONE)
```

### `cb archive [session]` — Cleanup

Unchanged from v0.1:
1. Prompt for confirmation
2. Kill tmux session
3. Remove git worktree
4. Preserve git branch

Auto-detects session from current directory if no argument provided.
Worktree path detection updated for `.worktrees/` location.

### Removed Commands

- `cb prompt` — skills handle prompting now
- `cb init` — no templates to install
- `cb version` — replaced by `cb --version` flag on root

---

## Cross-Repo Discovery

### How It Works (Stateless)

All state is derived from tmux at render time. No state files.

1. `tmux list-sessions` → filter for `cb_` prefix
2. For each session, query the first pane's working directory:
   ```
   tmux display-message -p -t cb_feat-auth:0 '#{pane_current_path}'
   → /Users/ron/code/claude-essentials/.worktrees/claude-essentials-feat-auth
   ```
3. From that path, determine the repo:
   ```
   git -C <path> rev-parse --show-toplevel
   → /Users/ron/code/claude-essentials
   ```
4. The repo root's basename becomes the group name: `claude-essentials`

### Edge Cases

- **Pane has `cd`'d away from worktree:** Mitigated by checking the shell pane (window 0), which typically stays in the worktree directory.
- **Repo deleted or inaccessible:** Session appears under an "Unknown" group.
- **tmux not running:** Dashboard shows "No active sessions" gracefully.

---

## Data Model

```go
type RepoGroup struct {
    Name     string             // Repo basename (e.g., "claude-essentials")
    Path     string             // Full repo root path
    Sessions []WorktreeSession
    Expanded bool
}

type WorktreeSession struct {
    Name     string    // tmux session name (e.g., "cb_feat-auth")
    Status   Status    // Rolled-up status: most active across Claude windows
    Windows  []Window
    Expanded bool
}

type Window struct {
    Index  int
    Name   string  // "shell", "claude", "claude:research"
    Active bool
    Status Status  // Per-window status (IDLE/WORKING/DONE for Claude windows)
}
```

**Status rollup:** A session's status is the most active status across its Claude windows:
`WORKING > IDLE > DONE`

If a session has no Claude windows, status is derived from the shell pane.

---

## TUI Dashboard

### Three-Level Tree Display

```
─ ClawdBay ─────────────────────────────────
▼ claude-essentials
  ▼ cb_feat-auth              ● WORKING
      shell
      claude              ● WORKING
      claude:research     ○ IDLE
  ▸ cb_refactor-tui           ○ IDLE
▸ my-other-project
─────────────────────────────────────────────
[enter] attach  [n] new  [c] claude  [x] archive  [r] refresh  [q] quit
```

Each Claude window displays its own status indicator. The session line shows the rolled-up status (most active across all Claude windows). When a session is collapsed, the rollup gives an at-a-glance picture. When expanded, individual window statuses are visible.

Non-Claude windows (e.g., `shell`) do not display a status indicator.

### Key Bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `Enter` | Context-sensitive: repo → toggle expand, session → attach, window → attach to session at window |
| `l` / `→` | Expand node |
| `h` / `←` | Collapse node |
| `n` | New session — prompts for branch name, runs `cb start --detach`, refreshes |
| `c` | Add Claude window to selected session |
| `x` | Archive selected session (with confirmation) |
| `r` | Manual refresh |
| `q` / `Ctrl+C` | Quit |

### Auto-Refresh

The dashboard polls tmux every 2-3 seconds using Bubbletea's `tea.Tick` command. This picks up:
- New sessions created outside the TUI
- Status changes (IDLE → WORKING → DONE)
- Sessions killed externally

### Dynamic Footer

Footer changes based on selected item type:
- **On a repo:** `[enter] expand  [n] new session  [q] quit`
- **On a session:** `[enter] attach  [c] add claude  [x] archive  [q] quit`
- **On a window:** `[enter] attach  [q] quit`

### Attach Behavior

When attaching from the TUI:
1. TUI sends `tea.Quit`
2. If targeting a specific window: `tmux select-window -t cb_<session>:<window>`
3. If inside tmux: `tmux switch-client -t cb_<session>`
4. If outside tmux: `tmux attach -t cb_<session>`

---

## Worktree Location

**Before (v0.1):**
```
~/code/
├── claude-essentials/              # Main repo
├── claude-essentials-feat-auth/    # Worktree (sibling)
└── claude-essentials-refactor/     # Worktree (sibling)
```

**After (v0.2):**
```
~/code/claude-essentials/
├── .worktrees/                     # All worktrees contained here
│   ├── claude-essentials-feat-auth/
│   └── claude-essentials-refactor/
├── .gitignore                      # Includes .worktrees/
├── cmd/
└── internal/
```

`cb start` will automatically add `.worktrees/` to `.gitignore` if not already present.

---

## Files to Remove

| File | Reason |
|------|--------|
| `cmd/prompt.go` | Prompt management command |
| `cmd/init.go` | Template installation command |
| `internal/prompt/prompt.go` | Template listing logic |
| `internal/prompt/prompt_test.go` | Tests for above |
| `templates/embed.go` | Embedded template filesystem |
| `templates/prompts/research.md` | Prompt template |
| `templates/prompts/plan.md` | Prompt template |
| `templates/prompts/implement.md` | Prompt template |
| `templates/prompts/verify.md` | Prompt template |

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/root.go` | Remove `prompt` and `init` subcommand registration |
| `cmd/start.go` | Worktree path → `.worktrees/`, add Claude window creation, gitignore management |
| `cmd/claude.go` | Remove `--prompt` flag |
| `cmd/list.go` | Add repo grouping to output |
| `cmd/archive.go` | Update worktree path detection for `.worktrees/` |
| `internal/tmux/tmux.go` | Add `GetPaneWorkingDir()`, support repo discovery |
| `internal/tmux/tmux_test.go` | Tests for new methods |
| `internal/tui/model.go` | Three-level data model, tree navigation, expand/collapse, auto-refresh tick, action handlers |
| `internal/tui/view.go` | Three-level tree rendering, dynamic footer, per-window status |
| `internal/tui/model_test.go` | Tests for new data model and navigation |

---

## Testing Strategy

- **Unit tests:** Table-driven tests for repo discovery, tree navigation, status rollup
- **tmux client tests:** Mock-based tests for new `GetPaneWorkingDir()` method
- **TUI model tests:** Navigation through three-level tree, expand/collapse state, action dispatch
- **Integration test:** Update `TestCLI_StartWorkflow` for new `.worktrees/` path and Claude window creation
