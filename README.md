# ClawdBay

![Go 1.25.7+](https://img.shields.io/badge/Go-1.25.7%2B-00ADD8?logo=go)
![tmux 3.x+](https://img.shields.io/badge/tmux-3.x%2B-1BB91F?logo=tmux)
![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![CI](https://github.com/ronsanzone/clawd-bay/actions/workflows/ci.yml/badge.svg)

Run multiple coding-agent workflows in parallel without losing context.

ClawdBay combines git worktrees, tmux sessions, and a fast terminal dashboard so each task stays isolated, observable, and easy to switch into.
You run your preferred coding agent CLI in tmux panes; ClawdBay focuses on monitoring and fast session/window switching.

## Why ClawdBay

- Keep each task isolated with `cb start <branch>` in its own worktree and `cb_<branch>` tmux session.
- Monitor and jump to the exact session/window from `cb dash` using status-aware navigation.
- Stay stateless: workflow state is derived directly from tmux, not a background database.


## Quick Start

**Prerequisites:** Go 1.25.7+, tmux 3.x+, and any coding agent CLI(s) you plan to run in tmux panes (for example `claude`, `codex`, or `open-code`).

```bash
# Install latest published version
go install github.com/ronsanzone/clawd-bay@latest

# Or via Makefile wrapper
make install

# Or build from this checkout
make build

# Configure projects that should appear in dash/list
cb project add /absolute/path/to/repo-a --name repo-a

# Start a workflow in a configured repo
cd /absolute/path/to/repo-a
cb start feat-auth

# Open dashboard
cb dash
cb dash --mode agents
```

## Core Commands

| Command | Description |
|---------|-------------|
| `cb start <branch>` | Create `.worktrees/<repo>-<branch>` + tmux session `cb_<branch>` |
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
- Session placement is pinned to tmux metadata (`@cb_home_path`) set by `cb start`, so grouping stays stable as pane cwd changes.
- Sessions missing valid home metadata are grouped under `(main repo)` for their owning configured project.
- If you run `cb start` from an unconfigured repo, ClawdBay warns that the session will not appear in `cb dash` / `cb list`.

## Documentation

- [Installation & Command Reference](INSTALL.md)
- [README Media Workflow](docs/media/README.md)
