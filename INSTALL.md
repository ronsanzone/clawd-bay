# Installation & Command Reference

## Prerequisites

### Go 1.21+

```bash
# macOS
brew install go

# Verify
go version
```

### tmux 3.x+

```bash
# macOS
brew install tmux

# Linux (Debian/Ubuntu)
sudo apt install tmux

# Verify
tmux -V
```

### Claude Code CLI

Install from [Anthropic's documentation](https://docs.anthropic.com/en/docs/claude-code).

```bash
# Verify
claude --version
```

## Installation

### Option 1: Go Install (Recommended)

```bash
go install github.com/rsanzone/clawdbay@latest
```

### Option 2: Build from Source

```bash
git clone https://github.com/rsanzone/clawdbay.git
cd clawdbay
make build
# Or: go build -o cb main.go

# Add to your PATH
cp cb ~/bin/
# Or: sudo cp cb /usr/local/bin/
```

### Verify Installation

```bash
cb version
# ClawdBay v0.1.0

cb init
# Creates ~/.config/cb/prompts/ with default templates
```

---

## Command Reference

### cb start

Create a git worktree and tmux session for a new workflow.

```
Usage: cb start <branch-name>
```

**Examples:**
```bash
cb start feat-auth           # Creates worktree + session "cb:feat-auth"
cb start proj-123-bugfix     # Branch and session name derived from input
cb start feature/add-login   # Slashes converted to dashes
```

**What it does:**
1. Creates git worktree at `../<project>-<branch>/`
2. Creates tmux session named `cb:<branch>`
3. Switches to (or attaches) the new session

---

### cb claude

Add a Claude Code session to the current worktree.

```
Usage: cb claude [flags]

Flags:
  -n, --name <name>      Name for the Claude session (default: "default")
  -p, --prompt <file>    Prompt file from .prompts/ to execute
```

**Examples:**
```bash
cb claude                           # Creates window "claude:default"
cb claude --name research           # Creates window "claude:research"
cb claude -n impl -p implement.md   # Named session with prompt
```

**Notes:**
- Must be run from within a `cb:` tmux session or matching worktree
- Creates a new tmux window and launches `claude` in it
- With `--prompt`, runs `claude < .prompts/<file>`

---

### cb dash

Open the interactive TUI dashboard.

```
Usage: cb dash
       cb        # Alias - runs dash by default
```

**Dashboard features:**
- Shows all `cb:` sessions grouped by worktree
- Displays Claude session status (IDLE, WORKING, DONE)
- Navigate with arrow keys or j/k
- Press Enter to attach to selected session
- Press q to quit

**Keybindings:**
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Attach to session |
| `q` | Quit dashboard |

---

### cb list

List all active ClawdBay workflows (non-interactive).

```
Usage: cb list
```

**Example output:**
```
Active workflows:
  cb:feat-auth
  cb:proj-123-bugfix
  cb:refactor-api
```

---

### cb archive

Archive a workflow by killing the tmux session and removing the worktree. The git branch is preserved.

```
Usage: cb archive [session-name]
```

**Examples:**
```bash
cb archive                  # Archives current workflow (auto-detected)
cb archive feat-auth        # Archives specific workflow
cb archive cb:feat-auth     # Full session name also works
```

**What it does:**
1. Prompts for confirmation
2. Kills the tmux session
3. Removes the git worktree
4. Keeps the branch (can be restored later)

---

### cb prompt

Manage prompt templates.

#### cb prompt list

List available prompt templates from `~/.config/cb/prompts/`.

```bash
cb prompt list
# Available templates:
#   - research
#   - plan
#   - implement
#   - verify
```

#### cb prompt add

Copy a template to `.prompts/` in current directory and open in editor.

```
Usage: cb prompt add <template-name>
```

```bash
cb prompt add research      # Copies research.md to .prompts/
cb prompt add plan          # Opens in $EDITOR (default: nvim)
```

#### cb prompt run

Execute a prompt file with Claude.

```
Usage: cb prompt run <prompt-file>
```

```bash
cb prompt run research.md   # Runs: claude < .prompts/research.md
```

---

### cb init

Initialize ClawdBay configuration and install default prompt templates.

```
Usage: cb init
```

**What it creates:**
```
~/.config/cb/
└── prompts/
    ├── research.md
    ├── plan.md
    ├── implement.md
    └── verify.md
```

**Notes:**
- Safe to run multiple times (won't overwrite existing files)
- Run after installation to set up templates

---

### cb version

Print the ClawdBay version.

```bash
cb version
# ClawdBay v0.1.0
```

---

## Prompt Templates

### Default Templates

| Template | Purpose |
|----------|---------|
| `research.md` | Explore codebase, understand existing patterns |
| `plan.md` | Create detailed implementation plan |
| `implement.md` | Execute plan task by task with TDD |
| `verify.md` | Run verification checklist before completion |

### Creating Custom Templates

1. Create or edit files in `~/.config/cb/prompts/`
2. Use `.md` extension
3. Templates are plain markdown—no variable substitution

### Per-Project Templates

For project-specific prompts:
1. Create `.prompts/` directory in your worktree
2. Use `cb prompt add <template>` to copy and customize
3. Reference with `cb claude --prompt <file>`

---

## Troubleshooting

### "not in a git repository"

`cb start` must be run from within a git repository.

```bash
cd /path/to/your/repo
cb start my-feature
```

### "no cb: session found"

`cb claude` requires an active `cb:` tmux session. Either:
- Run from within a tmux session started by `cb start`
- Or run `cb start` first to create a workflow

### "tmux not running"

Start tmux first, or let `cb start` create a session:

```bash
cb start my-feature   # Creates and attaches to session
```

### Dashboard shows no workflows

No `cb:` prefixed tmux sessions exist. Start a workflow:

```bash
cb start my-feature
```

### Templates not found

Run `cb init` to install default templates:

```bash
cb init
```
