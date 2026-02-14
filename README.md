# ClawdBay

A CLI + TUI tool for managing multi-session coding-agent workflows across git worktrees and tmux sessions.

## Quick Start

**Prerequisites:** Go 1.25.7+, tmux 3.x+, and at least one coding agent CLI (`claude`, `codex`, or `open-code`) on your PATH.

```bash
# Install
make build

# Configure projects that should appear in dash/list
cb project add /absolute/path/to/repo-a --name repo-a

# Start a workflow in a configured repo
cd /absolute/path/to/repo-a
cb start feat-auth

# Add another agent window
cb claude --name review

# Open dashboard
cb dash
cb dash --mode agents
```

## Core Commands

| Command | Description |
|---------|-------------|
| `cb start <branch>` | Create `.worktrees/<repo>-<branch>` + tmux session `cb_<branch>` |
| `cb claude` | Add a Claude window to the matching `cb_` session |
| `cb dash` / `cb` | Interactive dashboard (project-scoped) |
| `cb dash --mode agents` | Dashboard listing detected agent windows across all tmux sessions |
| `cb list` | Non-interactive project/worktree/session tree (project-scoped) |
| `cb project add/remove/list` | Manage configured project roots |
| `cb archive [session]` | Kill workflow session + remove worktree (branch preserved) |
| `cb clist` | List all tmux sessions/windows with agent detection (intentionally unscoped) |

## Configuration

ClawdBay project scope is configured in `~/.config/cb/config.toml`:

```toml
version = 1

[[projects]]
path = "/Users/you/code/repo-a"
name = "repo-a" # optional
```

Notes:
- Paths are canonicalized via symlink resolution when added.
- `cb dash` and `cb list` only show configured projects.
- Sessions running from a configured repo root (outside `.worktrees/`) appear under `(main repo)`.
- If you run `cb start` from an unconfigured repo, ClawdBay warns that the session will not appear in `cb dash` / `cb list`.

## Documentation

- [Installation & Command Reference](INSTALL.md)
