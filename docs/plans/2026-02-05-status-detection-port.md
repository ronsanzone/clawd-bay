# ClawdBay Status Detection Improvement

**Date:** 2026-02-05
**Status:** Approved
**Source:** Port from agent-deck's detection approach

## Overview

Improve ClawdBay's status detection by porting agent-deck's more robust pattern-based approach. Expands from 3 states to 4 states, with better busy and prompt detection.

## Status Model

| State | Meaning | Badge | Color |
|-------|---------|-------|-------|
| WORKING | Actively executing | `● WORKING` | Green `#98BB6C` |
| WAITING | Needs user input | `◉ WAITING` | Yellow `#E6C384` |
| IDLE | Running, nothing happening | `○ IDLE` | Orange `#FF9E3B` |
| DONE | Session finished | `◌ DONE` | Gray `#54546D` |

**Rollup Priority:** `WORKING > WAITING > IDLE > DONE`

## Detection Patterns

### Busy Indicators (→ WORKING)

**String patterns:**
- `"ctrl+c to interrupt"` - Current Claude Code's primary busy signal
- `"esc to interrupt"` - Older versions / fallback

**Spinner characters:**
- Braille: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
- Asterisk: `✳✽✶✢`

### Prompt Indicators (→ WAITING)

**Permission prompts:**
- `"yes, allow once"`
- `"yes, allow always"`
- `"no, and tell claude"`

**Confirmation prompts:**
- `"continue?"`, `"proceed?"`, `"(y/n)"`, `"[yes/no]"`

**Input prompt:**
- Last non-empty line ends with `>` or `❯`

## Detection Priority

Within `detectClaudeActivity()`:
1. Check busy indicators first → WORKING
2. Check prompt indicators → WAITING
3. Default → IDLE

Busy takes precedence over prompts because Claude can show both simultaneously during transitions.

## Files to Modify

### `internal/tmux/tmux.go`
- Expand `Status` const from 3 to 4 states
- Refactor `detectClaudeActivity()` with new detection logic
- Add `hasBusyIndicator()` function
- Add `hasPromptIndicator()` function
- Increase pane capture from 5 to 10 lines

### `internal/tui/theme.go`
- Add `colorWaiting` yellow (`#E6C384`)

### `internal/tui/view.go`
- Update `renderStatusBadge()` for 4 states

### `internal/tui/model.go`
- Update `RollupStatus()` with new priority order

### `internal/tmux/tmux_test.go`
- Add `TestHasBusyIndicator` table-driven tests
- Add `TestHasPromptIndicator` table-driven tests
- Add `TestDetectClaudeActivity_Priority` integration test

### `internal/tui/model_test.go`
- Update `TestRollupStatus` for 4 states

## Implementation Notes

- Simple string matching is fast and reliable
- No regex needed for these patterns
- Easy to extend with new patterns later
- Matches agent-deck's proven detection logic
