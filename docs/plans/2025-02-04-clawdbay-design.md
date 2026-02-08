# ClawdBay Design

**Date:** 2025-02-04
**Status:** Draft
**CLI Command:** `cb`

## Overview

ClawdBay is a CLI + TUI tool for managing multi-session Claude Code workflows. It solves the problem of tracking and orchestrating multiple Claude sessions across multiple repositories and worktrees.

### Problems Solved

1. **Discovery** - Losing track of which Claude sessions exist and what they're doing
2. **Context Switching** - Mental overhead when returning to sessions after time away
3. **Lifecycle Management** - Too many manual steps to spin up worktree + session + Claude
4. **Workflow Automation** - Streamlined path from idea to working code

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          ClawdBay                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐     ┌──────────┐     ┌──────────┐                │
│  │   CLI    │────▶│  Prompt  │────▶│  Claude  │                │
│  │   (cb)   │     │ Templates│     │ Sessions │                │
│  └────┬─────┘     └──────────┘     └──────────┘                │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────┐     ┌──────────┐                                 │
│  │   tmux   │     │ worktree │                                 │
│  │ sessions │     │  + files │                                 │
│  └──────────┘     └──────────┘                                 │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Interactive Dashboard                   │  │
│  │  Shows sessions grouped by worktree, idle/working status  │  │
│  │  Navigate and attach to sessions directly                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## CLI Commands

### Core Commands

```bash
# Start a new workflow - creates worktree + tmux session
cb start <branch-name>
cb start proj-123-auth-feature
cb start feature/add-login

# Add a Claude session to current worktree
cb claude [--name <name>] [--prompt <file>]
cb claude                           # Creates default session
cb claude --name research           # Named session
cb claude --name impl --prompt implement.md  # With prompt

# Interactive TUI dashboard
cb dash
cb                                  # Alias for cb dash

# List all active workflows (non-interactive)
cb list

# Archive workflow (kill session + remove worktree, keep branch)
cb archive
cb archive <session-name>
```

### Prompt Management

```bash
# List available prompt templates
cb prompt list

# Copy template to .prompts/, open in nvim for editing
cb prompt add <template-name>
cb prompt add research
cb prompt add plan

# Execute prompt in Claude session
# (Uses input redirection: claude < .prompts/file.md)
cb prompt run <file>
cb prompt run research.md
```

## Directory Structure

### Global Configuration

```
~/.config/cb/
└── prompts/                    # Prompt templates
    ├── research.md             # Deep investigation template
    ├── plan.md                 # Implementation planning
    ├── implement.md            # Code implementation
    ├── verify.md               # Testing and verification
    ├── code-review.md          # PR review template
    └── spike.md                # Technical spike/exploration
```

### Per-Worktree

```
<worktree>/
└── .prompts/                   # Copied & customized prompts
    ├── research.md             # Edited for this specific ticket
    └── plan.md
```

## tmux Structure

### Session Naming Convention

```
cb:<ticket>                     # Parent session for worktree
```

### Window Layout

```
Session: cb:proj-123-auth
├── Window 0: shell             # Terminal in worktree
├── Window 1: claude:default    # First/default Claude session
├── Window 2: claude:research   # Named Claude session
└── Window 3: claude:implement  # Named Claude session
```

## Interactive Dashboard

```
┌─ ClawdBay ──────────────────────────────────────────────────────┐
│                                                                  │
│  ▼ proj-123-auth-feature                         2 sessions     │
│      ● IDLE    :research     5m ago                             │
│      ◐ WORKING :implement    now                                │
│                                                                  │
│  ▼ proj-456-fix-bug                              1 session      │
│      ● IDLE    :default      10m ago                            │
│                                                                  │
│  ▶ proj-789-refactor                             1 session      │
│      ○ DONE    :default      1h ago                             │
│                                                                  │
│  [Enter] Attach  [n] New  [c] Add Claude  [p] Add Prompt  [x] Archive │
└──────────────────────────────────────────────────────────────────┘
```

### Status Indicators

| Symbol | Status | Meaning |
|--------|--------|---------|
| ● IDLE | Needs attention | Claude waiting for input |
| ◐ WORKING | Active | Claude currently executing |
| ○ DONE | Complete | Session finished/inactive |

### Dashboard Actions

| Key | Action |
|-----|--------|
| Enter | Attach to selected session |
| n | Start new workflow (`cb start`) |
| c | Add Claude session to selected worktree |
| p | Add prompt to selected worktree |
| d | Show details (prompts executed, recent activity) |
| x | Archive selected workflow |
| q | Quit dashboard |

## Session State Detection

State is derived from tmux heuristics (no separate state file):

1. **IDLE** - Pane shows Claude prompt, no recent output activity
2. **WORKING** - Recent output activity, Claude is executing
3. **DONE** - Session exists but no activity for extended period

Detection is filtered to `cb:*` sessions only, ignoring other tmux sessions.

## Workflow Example

### Starting Work on a Feature

```bash
# 1. Start workflow with branch name
cb start proj-123-add-user-auth
# Creates: worktree at ../myproject-proj-123-add-user-auth
# Creates: tmux session cb:proj-123-add-user-auth

# 2. Add research prompt, customize for this ticket
cb prompt add research
# Opens nvim with .prompts/research.md

# 3. Spin up Claude with the research prompt
cb claude --name research --prompt research.md
# Creates window claude:research
# Runs: claude < .prompts/research.md

# 4. Later, add implementation session
cb claude --name impl
# Start implementing based on research findings
```

### Managing Multiple Workflows

```bash
# Open dashboard to see all active work
cb dash

# Navigate to session that needs attention (IDLE status)
# Press Enter to attach

# When done with a workflow
cb archive
# Kills tmux session, removes worktree, keeps git branch
```

## Prompt Template Format

Templates are plain markdown files. When copied to a worktree via `cb prompt add`, users customize them manually for their specific task.

```markdown
# Research

## Context

<!-- Add ticket ID and description here -->

## Instructions

1. Explore the codebase to understand current implementation
2. Identify key files and components involved
3. Document findings in docs/plans/YYYY-MM-DD-<topic>-research.md

## Output

Produce a research report with:
- Current state analysis
- Key components identified
- Potential approaches
- Open questions
```

Templates are intentionally simple - no variable substitution. Users edit the copied file to add context specific to their task.

## Implementation Notes

### Technology Choices

- **Language:** Go
- **TUI:** Bubbletea with Lipgloss styling
- **tmux integration:** Direct tmux commands

### Integration with Existing Tools

- Builds on `init-worktree` script patterns
- Complements `tmux-sessionizer` (cb handles workflow sessions, sessionizer handles general navigation)

### Future Enhancements (Path to Autonomous)

Once the prompt-based workflow is solid:

1. **Scheduled execution** - Queue prompts to run sequentially
2. **Cross-session coordination** - Research session feeds into planning session
3. **Autonomous mode** - cb runs through prompts with minimal intervention, surfacing only at decision points

## Open Questions

1. ~~**Bash vs Go**~~ - Resolved: Go for better TUI support with Bubbletea
2. ~~**Prompt variable syntax**~~ - Resolved: No variable substitution, users edit templates manually
3. **tmux binding** - Should `cb dash` also be bindable to a tmux prefix key?
