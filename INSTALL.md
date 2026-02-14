# Installation & Command Reference

## Prerequisites

- Go 1.25.7+
- tmux 3.x+
- A coding agent CLI available in tmux panes (`claude`, `codex`, or `open-code`)

## Build / Install

```bash
git clone https://github.com/rsanzone/clawdbay.git
cd clawdbay
make build
```

Run from source:

```bash
./cb --help
```

## Commands

### `cb project`

Manage project roots used by `cb dash` and `cb list`.

```bash
cb project add <path> [--name <display>]
cb project remove <path>
cb project remove --name <display>
cb project list
```

Behavior:
- `add` canonicalizes and persists the project path.
- `remove <path>` requires canonical-path matching.
- `remove --name` is explicit and must match exactly one project.
- `list` shows configured paths and validation status (`OK` / `INVALID`).

### `cb start`

Create a new git worktree and tmux session.

```bash
cb start <branch-name>
cb start --detach <branch-name>
```

Behavior:
- Creates worktree at `<repo>/.worktrees/<repo>-<branch>`.
- Ensures `.worktrees/` exists and is in `.gitignore`.
- Creates tmux session `cb_<branch>` and a `claude` window.
- Warns if current repo is not configured in `config.toml`.

### `cb claude`

Add a Claude window to the matching workflow session.

```bash
cb claude
cb claude --name review
```

### `cb dash` (or `cb`)

Open the interactive dashboard.

Hierarchy:
- Project
- Worktree
- Session
- Window

Scoping:
- Only configured projects are shown.
- Inactive worktrees are still shown.
- Session placement is pinned to tmux metadata (`@cb_home_path`) written by `cb start`.
- Sessions without valid home metadata are grouped under `(main repo)` for their owning configured project.

### `cb list`

Print project/worktree/session tree output with rolled-up status.

```bash
cb list
```

### `cb archive`

Archive workflow by killing session and removing worktree.

```bash
cb archive
cb archive <session-name>
```

### `cb clist`

List windows and detected agents across tmux sessions.

```bash
cb clist
```

`clist` intentionally does **not** use project configuration scope.

## Config File

Path: `~/.config/cb/config.toml`

```toml
version = 1

[[projects]]
path = "/Users/you/code/repo-a"
name = "repo-a"
```

Rules:
- `version` must be `1`.
- `projects` may be empty.
- Paths are canonicalized and deduplicated by canonical path.
- Writes are atomic and persisted with `0600` mode.

## Troubleshooting

### `cb dash` / `cb list` shows no projects

Configure at least one project:

```bash
cb project add /absolute/path/to/repo
```

### Started a session but it does not appear in `dash`/`list`

The repo is likely not configured. Verify with:

```bash
cb project list
```

Then add it:

```bash
cb project add /absolute/path/to/repo
```

### `project list` shows `INVALID`

Configured path no longer canonicalizes (moved/deleted/symlink target missing). Fix by removing and re-adding the project path.
