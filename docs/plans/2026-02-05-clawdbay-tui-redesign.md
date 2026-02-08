# ClawdBay TUI Redesign — "Kanagawa Claw"

**Date:** 2026-02-05
**Status:** Design

## Overview

Full visual overhaul of the ClawdBay dashboard. The current TUI is a minimal text-flow tree that floats in the top-left of the terminal. The redesign makes it a proper full-screen TUI with a cohesive color palette inspired by Kanagawa.nvim, rounded borders, and a layout structured for future extensibility (preview panel).

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Screen mode | Alternate screen (`tea.WithAltScreen()`) | Immersive, clean exit, standard TUI practice |
| Layout | Single panel, 4-zone vertical | Simple now, composable for future dual-panel |
| Color palette | Kanagawa-inspired custom ("Kanagawa Claw") | Warm, muted, distinctive identity |
| Borders | Rounded (`lipgloss.RoundedBorder()`) | Modern, soft feel |
| Cursor | `❯` with sakuraPink + bgLight row | Clear selection without being harsh |

## Color Palette

```
── Backgrounds ──────────────────────────────
bg         #1F1F28  sumiInk3 — main background
bgDark     #16161D  sumiInk0 — title bar, footer
bgLight    #2A2A37  sumiInk4 — selected row highlight
border     #363646  sumiInk5 — border color

── Text ─────────────────────────────────────
fg         #DCD7BA  fujiWhite — primary text
fgDim      #C8C093  oldWhite — session names
fgMuted    #727169  fujiGray — window names, help text

── Accent ───────────────────────────────────
accent     #957FB8  oniViolet — title, repo names
highlight  #D27E99  sakuraPink — cursor/selection
info       #7E9CD8  crystalBlue — informational

── Status ───────────────────────────────────
working    #98BB6C  springGreen — active processing
idle       #FF9E3B  roninYellow — waiting for input
done       #54546D  sumiInk6 — finished, recedes visually
```

## Layout Structure

```
╭─ ClawdBay ──────────────────────────────────────────────────────╮
│                                                                 │
│  ▼ claude-essentials                                            │
│    ▼ cb_main                                    ● WORKING       │
│        shell                                                    │
│      ❯ claude                                   ● WORKING       │
│    ▸ cb_feat-auth                               ○ IDLE          │
│                                                                 │
│  ▼ other-project                                                │
│    ▸ cb_feature-123                             ◌ DONE          │
│                                                                 │
│                                                                 │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  3 sessions  ·  2 working  ·  1 idle                            │
╰─ enter attach  ·  c claude  ·  x archive  ·  q quit ───────────╯
```

### Zone 1: Title Bar
- "ClawdBay" rendered in the top border of the rounded box
- Uses `accent` color (oniViolet, #957FB8)

### Zone 2: Tree View (scrollable)
- Takes all remaining vertical space
- Scrolls to keep cursor visible when content exceeds height
- 2-character left padding from border

**Repo nodes:**
- Bold, `accent` color
- `▼`/`▸` expand/collapse arrows

**Session nodes:**
- `fgDim` color (#C8C093)
- Status badge right-aligned (dynamic padding based on terminal width)
- `● WORKING` (springGreen), `○ IDLE` (roninYellow), `◌ DONE` (sumiInk6)

**Window nodes:**
- `fgMuted` color (#727169)
- 8-space total indent
- Only claude windows show status badges

**Cursor/selection:**
- `❯` marker replaces `>`
- Text becomes `highlight` color (sakuraPink)
- Entire row gets `bgLight` (#2A2A37) background

**Spacing:**
- Blank line between repo groups

### Zone 3: Status Summary
- Horizontal separator (`├───┤`)
- One-line summary: "N sessions · X working · Y idle"
- Counts use their respective status colors

### Zone 4: Help Footer
- Context-sensitive keybindings in bottom border line
- Changes based on selected node type (repo/session/window)
- `fgMuted` color

## Code Architecture

### File Structure

```
internal/tui/
├── model.go       ← data model, Update(), business logic (modified)
├── view.go        ← View(), layout composition (refactored)
├── theme.go       ← NEW: color palette, style builders
└── model_test.go  ← tests (existing)
```

### theme.go (new)

Defines the color palette as a Theme struct with named constants:

```go
type Theme struct {
    Bg, BgDark, BgLight, Border lipgloss.Color
    Fg, FgDim, FgMuted          lipgloss.Color
    Accent, Highlight, Info     lipgloss.Color
    Working, Idle, Done         lipgloss.Color
}

var KanagawaClaw = Theme{ ... }
```

Style builders derive lipgloss styles from the theme. All styles reference the theme — no magic color numbers in rendering code.

### model.go (modified)

Add terminal size tracking:

```go
type Model struct {
    // ... existing fields
    Width  int
    Height int
}
```

Handle `tea.WindowSizeMsg` in Update() to track terminal dimensions.

### view.go (refactored)

Split View() into composable sub-renders:

```go
func (m Model) View() string {
    tree := m.renderTree(width, treeHeight)
    status := m.renderStatusBar(width)
    content := lipgloss.JoinVertical(lipgloss.Left, tree, status)
    return m.renderFrame(content, width, height)
}
```

Future extensibility for preview panel:

```go
tree := m.renderTree(leftWidth, height)
preview := m.renderPreview(rightWidth, height)
content := lipgloss.JoinHorizontal(lipgloss.Top, tree, preview)
```

### cmd/dash.go (modified)

Add `tea.WithAltScreen()` to the program options.

## Scrolling

When tree content exceeds available height:
- Track a `scrollOffset` in the model
- Render only the visible slice of nodes
- Adjust scroll to keep cursor in view (scroll up/down as cursor moves)

## Future Extensibility

The composable View() architecture allows adding:
- Right-side preview panel (session details, branch info, working dir)
- Search/filter bar at the top
- Dialog overlays for actions (archive confirmation, new session)
