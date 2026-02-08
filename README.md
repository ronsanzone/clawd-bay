# ClawdBay

A CLI + TUI tool for managing multi-session Claude Code workflows. Stop losing track of Claude sessions across worktrees—start, monitor, and switch between them from one dashboard.

## Quick Start

**Prerequisites:** Go 1.21+, tmux 3.x+, [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)

```bash
# Install
go install github.com/rsanzone/clawdbay@latest

# Initialize config and prompt templates
cb init

# Start your first workflow
cb start my-feature
cb claude --name research
cb dash
```

## Core Workflow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  cb start   │────▶│  cb claude  │────▶│   cb dash   │────▶│ cb archive  │
│  (worktree  │     │  (add       │     │  (monitor   │     │  (cleanup   │
│   + tmux)   │     │   sessions) │     │   & switch) │     │   when done)│
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

**The idea:** Each feature gets an isolated git worktree with a dedicated tmux session. Spin up multiple Claude sessions per worktree (research, implementation, review). The dashboard shows everything at a glance.

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `cb start <branch>` | Create worktree + tmux session | `cb start feat-auth` |
| `cb claude` | Add Claude session to current worktree | `cb claude --name research` |
| `cb dash` | Interactive dashboard (default) | `cb` |
| `cb list` | List active workflows | `cb list` |
| `cb archive` | Clean up workflow (keeps branch) | `cb archive` |
| `cb prompt list` | Show available templates | `cb prompt list` |
| `cb prompt add` | Copy template for customization | `cb prompt add research` |
| `cb init` | Initialize config directory | `cb init` |

## Configuration

```
~/.config/cb/
└── prompts/           # Prompt templates
    ├── research.md    # Codebase exploration
    ├── plan.md        # Implementation planning
    ├── implement.md   # Code execution
    └── verify.md      # Testing checklist
```

Customize templates by editing files in `~/.config/cb/prompts/`. Per-project prompts go in `.prompts/` within your worktree.

## Documentation

- [Installation & Command Reference](INSTALL.md) - Full setup and detailed command docs
- [Design Document](../plans/2025-02-04-clawdbay-design.md) - Architecture and rationale

## License

MIT
