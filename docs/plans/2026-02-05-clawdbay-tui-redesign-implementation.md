# ClawdBay TUI Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform the ClawdBay dashboard from a minimal text-flow tree into a polished full-screen TUI with a Kanagawa-inspired color palette, rounded borders, scrolling, and composable layout.

**Architecture:** Single-panel full-screen TUI using Bubbletea's alternate screen mode. A new `theme.go` file isolates the color palette. `view.go` is refactored into composable sub-renders (renderTree, renderStatusBar, renderFrame) that compose via `lipgloss.JoinVertical`. The Model gains Width/Height tracking and scroll offset. Future preview panel can be added with `lipgloss.JoinHorizontal`.

**Tech Stack:** Go, Bubbletea v1.3.10, Lipgloss v1.1.0 (no new dependencies)

**Design doc:** `docs/plans/2026-02-05-clawdbay-tui-redesign.md`

**Verification:** `go test ./internal/tui/ -v` and `go vet ./...` after each task.

---

## Task 1: Create theme.go with Kanagawa Claw palette

**Files:**
- Create: `internal/tui/theme.go`
- Test: `internal/tui/theme_test.go`

**Context:** All colors are currently hardcoded ANSI-256 numbers scattered as `var` declarations in `view.go:11-41`. We're replacing them with a structured theme using hex colors from the Kanagawa palette.

**Step 1: Write the failing test**

Create `internal/tui/theme_test.go`:

```go
package tui

import (
	"testing"
)

func TestKanagawaClawThemeHasAllColors(t *testing.T) {
	theme := KanagawaClaw

	// Verify all palette colors are set (non-empty)
	colors := map[string]string{
		"Bg":        string(theme.Bg),
		"BgDark":    string(theme.BgDark),
		"BgLight":   string(theme.BgLight),
		"Border":    string(theme.Border),
		"Fg":        string(theme.Fg),
		"FgDim":     string(theme.FgDim),
		"FgMuted":   string(theme.FgMuted),
		"Accent":    string(theme.Accent),
		"Highlight": string(theme.Highlight),
		"Info":      string(theme.Info),
		"Working":   string(theme.Working),
		"Idle":      string(theme.Idle),
		"Done":      string(theme.Done),
	}

	for name, val := range colors {
		if val == "" {
			t.Errorf("theme color %s is empty", name)
		}
	}
}

func TestBuildStylesFromTheme(t *testing.T) {
	styles := NewStyles(KanagawaClaw)

	// Verify styles were created (non-zero value check via rendering)
	// Rendering a string through a style should produce output
	if styles.Title.Render("test") == "" {
		t.Error("Title style renders empty")
	}
	if styles.Repo.Render("test") == "" {
		t.Error("Repo style renders empty")
	}
	if styles.Session.Render("test") == "" {
		t.Error("Session style renders empty")
	}
	if styles.Window.Render("test") == "" {
		t.Error("Window style renders empty")
	}
	if styles.StatusWorking.Render("test") == "" {
		t.Error("StatusWorking style renders empty")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestKanagawaClaw -v`
Expected: FAIL — `KanagawaClaw` and `NewStyles` undefined

**Step 3: Write theme.go**

Create `internal/tui/theme.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines all colors for the TUI.
type Theme struct {
	Bg      lipgloss.Color
	BgDark  lipgloss.Color
	BgLight lipgloss.Color
	Border  lipgloss.Color

	Fg      lipgloss.Color
	FgDim   lipgloss.Color
	FgMuted lipgloss.Color

	Accent    lipgloss.Color
	Highlight lipgloss.Color
	Info      lipgloss.Color

	Working lipgloss.Color
	Idle    lipgloss.Color
	Done    lipgloss.Color
}

// KanagawaClaw is the default theme inspired by Kanagawa.nvim.
var KanagawaClaw = Theme{
	Bg:      lipgloss.Color("#1F1F28"),
	BgDark:  lipgloss.Color("#16161D"),
	BgLight: lipgloss.Color("#2A2A37"),
	Border:  lipgloss.Color("#363646"),

	Fg:      lipgloss.Color("#DCD7BA"),
	FgDim:   lipgloss.Color("#C8C093"),
	FgMuted: lipgloss.Color("#727169"),

	Accent:    lipgloss.Color("#957FB8"),
	Highlight: lipgloss.Color("#D27E99"),
	Info:      lipgloss.Color("#7E9CD8"),

	Working: lipgloss.Color("#98BB6C"),
	Idle:    lipgloss.Color("#FF9E3B"),
	Done:    lipgloss.Color("#54546D"),
}

// Styles holds all pre-built lipgloss styles derived from a Theme.
type Styles struct {
	// Frame
	Title lipgloss.Style
	Frame lipgloss.Style

	// Tree nodes
	Repo     lipgloss.Style
	Session  lipgloss.Style
	Window   lipgloss.Style
	Selected lipgloss.Style

	// Status badges
	StatusWorking lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusDone    lipgloss.Style

	// UI chrome
	Footer    lipgloss.Style
	StatusBar lipgloss.Style
}

// NewStyles builds all styles from the given theme.
func NewStyles(t Theme) Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent),

		Frame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),

		Repo: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent),

		Session: lipgloss.NewStyle().
			Foreground(t.FgDim),

		Window: lipgloss.NewStyle().
			Foreground(t.FgMuted),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Highlight).
			Background(t.BgLight),

		StatusWorking: lipgloss.NewStyle().
			Foreground(t.Working),

		StatusIdle: lipgloss.NewStyle().
			Foreground(t.Idle),

		StatusDone: lipgloss.NewStyle().
			Foreground(t.Done),

		Footer: lipgloss.NewStyle().
			Foreground(t.FgMuted),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.FgMuted),
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestKanagawaClaw -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./internal/tui/ -v && go vet ./...`
Expected: ALL PASS (theme.go is additive, no existing code changes)

**Step 6: Commit**

```bash
git add internal/tui/theme.go internal/tui/theme_test.go
git commit -m "feat(tui): add Kanagawa Claw theme with style builders"
```

---

## Task 2: Add terminal dimensions and scroll offset to Model

**Files:**
- Modify: `internal/tui/model.go:59-68` (Model struct)
- Modify: `internal/tui/model.go:227-265` (Update method — add WindowSizeMsg handler)
- Test: `internal/tui/model_test.go` (add new tests)

**Context:** The current Model has no awareness of terminal size. We need `Width`, `Height`, and `ScrollOffset` fields, plus a `tea.WindowSizeMsg` handler. The scroll offset tracks which node-line is at the top of the viewport.

**Step 1: Write the failing test**

Add to `internal/tui/model_test.go`:

```go
func TestSessionCounts(t *testing.T) {
	tests := []struct {
		name                          string
		groups                        []RepoGroup
		wantTotal, wantWork, wantIdle int
	}{
		{
			name: "mixed statuses",
			groups: []RepoGroup{
				{
					Name: "repo-a",
					Sessions: []WorktreeSession{
						{Name: "s1", Status: tmux.StatusWorking},
						{Name: "s2", Status: tmux.StatusIdle},
					},
				},
				{
					Name: "repo-b",
					Sessions: []WorktreeSession{
						{Name: "s3", Status: tmux.StatusDone},
					},
				},
			},
			wantTotal: 3, wantWork: 1, wantIdle: 1,
		},
		{
			name:      "empty",
			groups:    []RepoGroup{},
			wantTotal: 0, wantWork: 0, wantIdle: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{Groups: tt.groups}
			total, working, idle := m.SessionCounts()
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if working != tt.wantWork {
				t.Errorf("working = %d, want %d", working, tt.wantWork)
			}
			if idle != tt.wantIdle {
				t.Errorf("idle = %d, want %d", idle, tt.wantIdle)
			}
		})
	}
}

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		name       string
		lineCount  int
		viewHeight int
		cursorLine int
		scrollOff  int
		wantStart  int
		wantEnd    int
		wantScroll int
	}{
		{
			name:       "fits in view",
			lineCount:  5,
			viewHeight: 10,
			cursorLine: 2,
			scrollOff:  0,
			wantStart:  0,
			wantEnd:    5,
			wantScroll: 0,
		},
		{
			name:       "cursor below viewport scrolls down",
			lineCount:  20,
			viewHeight: 10,
			cursorLine: 12,
			scrollOff:  0,
			wantStart:  3,
			wantEnd:    13,
			wantScroll: 3,
		},
		{
			name:       "cursor above viewport scrolls up",
			lineCount:  20,
			viewHeight: 10,
			cursorLine: 2,
			scrollOff:  5,
			wantStart:  2,
			wantEnd:    12,
			wantScroll: 2,
		},
		{
			name:       "cursor at end",
			lineCount:  20,
			viewHeight: 10,
			cursorLine: 19,
			scrollOff:  0,
			wantStart:  10,
			wantEnd:    20,
			wantScroll: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, newScroll := VisibleRange(tt.lineCount, tt.viewHeight, tt.cursorLine, tt.scrollOff)
			if start != tt.wantStart {
				t.Errorf("start = %d, want %d", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %d, want %d", end, tt.wantEnd)
			}
			if newScroll != tt.wantScroll {
				t.Errorf("scroll = %d, want %d", newScroll, tt.wantScroll)
			}
		})
	}
}

func TestCursorToLine(t *testing.T) {
	groups := []RepoGroup{
		{
			Name: "repo-a", Expanded: true,
			Sessions: []WorktreeSession{
				{Name: "s1", Expanded: true, Windows: []tmux.Window{{Name: "shell"}, {Name: "claude"}}},
			},
		},
		{
			Name: "repo-b", Expanded: true,
			Sessions: []WorktreeSession{
				{Name: "s2", Expanded: false},
			},
		},
	}
	nodes := BuildNodes(groups)
	// Nodes: [repo-a, s1, shell, claude, repo-b, s2]
	// Lines: [repo-a, s1, shell, claude, <blank>, repo-b, s2]
	// Blank line before second repo group

	tests := []struct {
		cursor   int
		wantLine int
	}{
		{0, 0}, // repo-a → line 0
		{1, 1}, // s1 → line 1
		{2, 2}, // shell → line 2
		{3, 3}, // claude → line 3
		{4, 5}, // repo-b → line 5 (after blank line)
		{5, 6}, // s2 → line 6
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("cursor_%d", tt.cursor), func(t *testing.T) {
			line := CursorToLine(nodes, tt.cursor)
			if line != tt.wantLine {
				t.Errorf("CursorToLine(%d) = %d, want %d", tt.cursor, line, tt.wantLine)
			}
		})
	}
}
```

Note: Add `"fmt"` to the imports at the top of model_test.go.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestSessionCounts|TestVisibleRange|TestCursorToLine" -v`
Expected: FAIL — `SessionCounts`, `VisibleRange`, `CursorToLine` undefined

**Step 3: Implement Model changes**

Add fields to Model struct in `internal/tui/model.go:59-68`:

```go
// Model is the Bubbletea model for the dashboard.
type Model struct {
	Groups         []RepoGroup
	Cursor         int
	Nodes          []TreeNode
	Quitting       bool
	TmuxClient     *tmux.Client
	SelectedName   string
	SelectedWindow string
	WindowStatuses map[string]tmux.Status
	Width          int
	Height         int
	ScrollOffset   int
	Styles         Styles
}
```

Add `tea.WindowSizeMsg` handler in the Update switch in `internal/tui/model.go`. Add this case before the `tea.KeyMsg` case:

```go
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
```

Add `SessionCounts` method to `internal/tui/model.go`:

```go
// SessionCounts returns total sessions and counts by status.
func (m Model) SessionCounts() (total, working, idle int) {
	for _, g := range m.Groups {
		for _, s := range g.Sessions {
			total++
			switch s.Status {
			case tmux.StatusWorking:
				working++
			case tmux.StatusIdle:
				idle++
			}
		}
	}
	return
}
```

Add `VisibleRange` function to `internal/tui/model.go`:

```go
// VisibleRange calculates which lines to display given viewport constraints.
// Returns start (inclusive), end (exclusive), and new scroll offset.
func VisibleRange(lineCount, viewHeight, cursorLine, scrollOffset int) (start, end, newOffset int) {
	if lineCount <= viewHeight {
		return 0, lineCount, 0
	}

	newOffset = scrollOffset
	if cursorLine < newOffset {
		newOffset = cursorLine
	}
	if cursorLine >= newOffset+viewHeight {
		newOffset = cursorLine - viewHeight + 1
	}

	start = newOffset
	end = min(newOffset+viewHeight, lineCount)
	return start, end, newOffset
}
```

Add `CursorToLine` function to `internal/tui/model.go`:

```go
// CursorToLine maps a cursor position (node index) to a display line index,
// accounting for blank separator lines between repo groups.
func CursorToLine(nodes []TreeNode, cursor int) int {
	line := 0
	for i := 0; i < cursor && i < len(nodes); i++ {
		line++
		// A repo node that isn't the first node means a blank line was inserted before it
		if i+1 < len(nodes) && nodes[i+1].Type == NodeRepo {
			line++ // blank separator line
		}
	}
	return line
}
```

Update `InitialModel` in `internal/tui/model.go:166-172` to initialize Styles:

```go
// InitialModel creates the initial dashboard model.
func InitialModel(tmuxClient *tmux.Client) Model {
	return Model{
		Groups:         []RepoGroup{},
		TmuxClient:     tmuxClient,
		WindowStatuses: make(map[string]tmux.Status),
		Styles:         NewStyles(KanagawaClaw),
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestSessionCounts|TestVisibleRange|TestCursorToLine" -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./internal/tui/ -v && go vet ./...`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat(tui): add terminal dimensions, scroll support, and session counts"
```

---

## Task 3: Rewrite view.go with full-screen themed rendering

**Files:**
- Rewrite: `internal/tui/view.go` (complete rewrite)
- Test: `internal/tui/view_test.go` (new — view-specific tests)

**Context:** This is the main visual change. We're replacing the entire `view.go` with composable sub-renders that use the theme. The old `var` style declarations (lines 11-41) are removed entirely — all styles come from `Model.Styles`. The rendering builds display lines, applies scrolling, then wraps in a bordered frame.

**Important lipgloss behavior:** `lipgloss.Width()` returns the visual width of a styled string (excluding ANSI codes). Use this instead of `len()` for padding calculations. The frame border consumes 2 columns (left + right) and the title goes in the top border.

**Step 1: Write the failing tests**

Create `internal/tui/view_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/rsanzone/clawdbay/internal/tmux"
)

func TestRenderStatusBadge(t *testing.T) {
	m := Model{Styles: NewStyles(KanagawaClaw)}

	tests := []struct {
		status tmux.Status
		want   string
	}{
		{tmux.StatusWorking, "● WORKING"},
		{tmux.StatusIdle, "○ IDLE"},
		{tmux.StatusDone, "◌ DONE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := m.renderStatusBadge(tt.status)
			if !strings.Contains(got, tt.want) {
				t.Errorf("renderStatusBadge(%s) = %q, want to contain %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRenderStatusBar(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo",
				Sessions: []WorktreeSession{
					{Name: "s1", Status: tmux.StatusWorking},
					{Name: "s2", Status: tmux.StatusIdle},
					{Name: "s3", Status: tmux.StatusDone},
				},
			},
		},
		Styles: NewStyles(KanagawaClaw),
		Width:  80,
	}

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "3 sessions") {
		t.Errorf("status bar missing session count: %q", bar)
	}
	if !strings.Contains(bar, "1 working") {
		t.Errorf("status bar missing working count: %q", bar)
	}
	if !strings.Contains(bar, "1 idle") {
		t.Errorf("status bar missing idle count: %q", bar)
	}
}

func TestBuildDisplayLines(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo-a", Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "s1", Expanded: true, Windows: []tmux.Window{{Name: "shell"}}},
				},
			},
			{
				Name: "repo-b", Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "s2", Expanded: false},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Cursor:         0,
	}
	m.Nodes = BuildNodes(m.Groups)

	lines := m.buildDisplayLines()

	// Expect: repo-a, s1, shell, <blank>, repo-b, s2 = 6 lines
	if len(lines) != 6 {
		t.Errorf("got %d display lines, want 6. Lines: %v", len(lines), lines)
	}

	// Line 3 (index 3) should be the blank separator
	if strings.TrimSpace(lines[3]) != "" {
		t.Errorf("line 3 should be blank separator, got %q", lines[3])
	}
}

func TestViewContainsFrameElements(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo", Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "s1", Expanded: false, Status: tmux.StatusIdle},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
		Cursor:         0,
	}
	m.Nodes = BuildNodes(m.Groups)

	view := m.View()

	// Should contain the title
	if !strings.Contains(view, "ClawdBay") {
		t.Error("view missing ClawdBay title")
	}

	// Should contain rounded border characters
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") {
		t.Error("view missing rounded border characters")
	}

	// Should contain footer keybindings
	if !strings.Contains(view, "quit") {
		t.Error("view missing footer keybindings")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestRenderStatus|TestBuildDisplay|TestViewContains" -v`
Expected: FAIL — new methods not yet implemented

**Step 3: Rewrite view.go**

Replace the entire contents of `internal/tui/view.go` with:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	if m.Width == 0 || m.Height == 0 {
		return "Initializing..."
	}

	// Content width inside the border (border takes 2 cols)
	innerWidth := m.Width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Build the tree content
	tree := m.renderTree(innerWidth)

	// Build the status bar
	statusBar := m.renderStatusBar()

	// Build the footer
	footer := m.renderFooter()

	// Compose: tree on top, status bar below
	content := lipgloss.JoinVertical(lipgloss.Left, tree, statusBar)

	// Wrap in frame with title and footer
	return m.renderFrame(content, footer)
}

// renderTree builds the scrollable tree view.
func (m Model) renderTree(width int) string {
	if len(m.Nodes) == 0 {
		empty := fmt.Sprintf("  No active sessions.\n  Start one with: cb start <branch-name>")
		style := lipgloss.NewStyle().Foreground(m.Styles.Session.GetForeground())
		return style.Render(empty)
	}

	lines := m.buildDisplayLines()

	// Calculate available height for the tree
	// Total height minus: top border (1) + bottom border (1) + status bar (1) + footer in border (0)
	treeHeight := m.Height - 4
	if treeHeight < 1 {
		treeHeight = 1
	}

	// Apply scrolling
	cursorLine := CursorToLine(m.Nodes, m.Cursor)
	start, end, newOffset := VisibleRange(len(lines), treeHeight, cursorLine, m.ScrollOffset)
	m.ScrollOffset = newOffset

	visible := lines[start:end]

	// Pad each line to full width and pad to fill remaining height
	var padded []string
	for _, line := range visible {
		padded = append(padded, padToWidth(line, width))
	}
	for len(padded) < treeHeight {
		padded = append(padded, strings.Repeat(" ", width))
	}

	return strings.Join(padded, "\n")
}

// buildDisplayLines generates all tree lines including blank separators between repo groups.
func (m Model) buildDisplayLines() []string {
	var lines []string

	for i, node := range m.Nodes {
		// Blank line between repo groups
		if node.Type == NodeRepo && i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, m.renderNodeLine(node, i))
	}

	return lines
}

// renderNodeLine renders a single tree node as a display line.
func (m Model) renderNodeLine(node TreeNode, nodeIdx int) string {
	isSelected := nodeIdx == m.Cursor
	cursor := "  "
	if isSelected {
		cursor = "❯ "
	}

	var line string

	switch node.Type {
	case NodeRepo:
		repo := m.Groups[node.RepoIndex]
		icon := "▸"
		if repo.Expanded {
			icon = "▼"
		}
		name := m.Styles.Repo.Render(repo.Name)
		line = fmt.Sprintf("%s%s %s", cursor, icon, name)

	case NodeSession:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		icon := "▸"
		if session.Expanded {
			icon = "▼"
		}
		name := m.Styles.Session.Render(session.Name)
		badge := m.renderStatusBadge(session.Status)
		left := fmt.Sprintf("%s  %s %s", cursor, icon, name)
		line = m.rightAlign(left, badge)

	case NodeWindow:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		name := m.Styles.Window.Render(window.Name)
		badge := ""
		if strings.HasPrefix(window.Name, "claude") {
			key := session.Name + ":" + window.Name
			if status, ok := m.WindowStatuses[key]; ok {
				badge = m.renderStatusBadge(status)
			}
		}
		left := fmt.Sprintf("%s      %s", cursor, name)
		line = m.rightAlign(left, badge)

	default:
		line = cursor + "Unknown"
	}

	if isSelected {
		return m.Styles.Selected.Render(line)
	}
	return line
}

// rightAlign places a badge on the right side of the available width.
func (m Model) rightAlign(left, right string) string {
	if right == "" {
		return left
	}
	availWidth := m.Width - 4 // border (2) + padding (2)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := availWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderStatusBadge renders a colored status indicator.
func (m Model) renderStatusBadge(status tmux.Status) string {
	switch status {
	case tmux.StatusWorking:
		return m.Styles.StatusWorking.Render("● WORKING")
	case tmux.StatusIdle:
		return m.Styles.StatusIdle.Render("○ IDLE")
	case tmux.StatusDone:
		return m.Styles.StatusDone.Render("◌ DONE")
	default:
		return m.Styles.StatusDone.Render("◌ DONE")
	}
}

// renderStatusBar renders the session count summary line.
func (m Model) renderStatusBar() string {
	total, working, idle := m.SessionCounts()

	parts := []string{
		m.Styles.StatusBar.Render(fmt.Sprintf("%d sessions", total)),
	}
	if working > 0 {
		parts = append(parts, m.Styles.StatusWorking.Render(fmt.Sprintf("%d working", working)))
	}
	if idle > 0 {
		parts = append(parts, m.Styles.StatusIdle.Render(fmt.Sprintf("%d idle", idle)))
	}

	return "  " + strings.Join(parts, m.Styles.StatusBar.Render("  ·  "))
}

// renderFooter returns context-sensitive keybinding help text.
func (m Model) renderFooter() string {
	if m.Cursor >= len(m.Nodes) {
		return "q quit"
	}

	node := m.Nodes[m.Cursor]
	switch node.Type {
	case NodeRepo:
		return "enter expand  ·  n new  ·  q quit"
	case NodeSession:
		return "enter attach  ·  c claude  ·  x archive  ·  r refresh  ·  q quit"
	case NodeWindow:
		return "enter attach  ·  r refresh  ·  q quit"
	default:
		return "q quit"
	}
}

// renderFrame wraps content in a rounded border with title and footer.
func (m Model) renderFrame(content, footer string) string {
	titleText := m.Styles.Title.Render(" ClawdBay ")
	footerText := m.Styles.Footer.Render(" " + footer + " ")

	frame := m.Styles.Frame.
		Width(m.Width - 2).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.PlaceHorizontal(m.Width, lipgloss.Left,
			frame.Render(content),
			lipgloss.WithWhitespaceChars(" "),
		),
	) + "\n" +
		// Overlay: replace top-left border segment with title
		// and bottom-left with footer
		// NOTE: This uses a simple approach — lipgloss doesn't natively
		// support border titles, so we render the frame then use string
		// replacement to insert the title and footer into the border lines.
		""

	// Simpler approach: build the frame manually
}
```

**IMPORTANT NOTE:** The `renderFrame` function above has a known challenge — lipgloss doesn't natively support titles in borders. The implementing engineer should use this approach instead:

Build the frame manually using lipgloss border characters:

```go
func (m Model) renderFrame(content, footer string) string {
	w := m.Width
	if w < 20 {
		w = 20
	}

	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(m.Styles.Frame.GetBorderTopForeground())

	// Top border with title
	title := m.Styles.Title.Render(" ClawdBay ")
	titleWidth := lipgloss.Width(title)
	topLeft := borderStyle.Render(border.TopLeft + border.Top)
	topRight := borderStyle.Render(strings.Repeat(border.Top, max(0, w-titleWidth-4)) + border.TopRight)
	topLine := topLeft + title + topRight

	// Middle separator (before status bar)
	midLeft := borderStyle.Render(border.MiddleLeft)
	midRight := borderStyle.Render(border.MiddleRight)
	midLine := midLeft + borderStyle.Render(strings.Repeat(border.Top, w-2)) + midRight

	// Bottom border with footer
	footerText := m.Styles.Footer.Render(" " + footer + " ")
	footerWidth := lipgloss.Width(footerText)
	botLeft := borderStyle.Render(border.BottomLeft + border.Bottom)
	botRight := borderStyle.Render(strings.Repeat(border.Bottom, max(0, w-footerWidth-4)) + border.BottomRight)
	botLine := botLeft + footerText + botRight

	// Content lines with side borders
	side := borderStyle.Render(border.Left)
	sideR := borderStyle.Render(border.Right)

	var lines []string
	lines = append(lines, topLine)

	// Split content into tree and status bar (last line after split)
	contentLines := strings.Split(content, "\n")
	for _, cl := range contentLines {
		padded := padToWidth(cl, w-2)
		lines = append(lines, side+padded+sideR)
	}

	lines = append(lines, midLine)

	// Status bar is the last section of content — already included above
	// The midLine separates tree from footer visually

	lines = append(lines, botLine)

	return strings.Join(lines, "\n")
}
```

Wait — this is getting complex. Let me simplify the frame approach. The implementing engineer should split the content (tree vs status bar) BEFORE calling renderFrame, and renderFrame receives them separately:

```go
func (m Model) renderFrame(tree, statusBar, footer string) string {
	// ... builds top border with title, tree section, mid separator, status bar, bottom border with footer
}
```

The exact renderFrame implementation will need iteration — **the implementing engineer should get the basic structure working first (top border + content + bottom border), then refine the title/footer placement.** The key constraint is: use `lipgloss.RoundedBorder()` characters and build the border lines manually.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestRenderStatus|TestBuildDisplay|TestViewContains" -v`
Expected: PASS

**Step 5: Run ALL tests**

Run: `go test ./internal/tui/ -v && go vet ./...`
Expected: ALL PASS. Existing tests (TestGroupByRepo, TestBuildNodes, TestStatusRollup) should still pass since the model logic is unchanged.

**Step 6: Commit**

```bash
git add internal/tui/view.go internal/tui/view_test.go
git commit -m "feat(tui): rewrite view with Kanagawa theme, bordered frame, and scrolling"
```

---

## Task 4: Enable alternate screen in cmd/dash.go

**Files:**
- Modify: `cmd/dash.go:21` (add WithAltScreen option)

**Context:** This is the simplest change — one line. `tea.WithAltScreen()` tells Bubbletea to use the terminal's alternate screen buffer (like vim). The terminal is fully restored on exit.

**Step 1: Modify dash.go**

In `cmd/dash.go:21`, change:

```go
p := tea.NewProgram(model)
```

to:

```go
p := tea.NewProgram(model, tea.WithAltScreen())
```

**Step 2: Verify build**

Run: `go build ./... && go vet ./...`
Expected: PASS

**Step 3: Manual test**

Run: `go run . dash`
Expected: Dashboard takes over the full terminal with the new themed appearance. Press `q` to exit — terminal should be fully restored.

**Step 4: Commit**

```bash
git add cmd/dash.go
git commit -m "feat(tui): enable alternate screen for full-screen dashboard"
```

---

## Task 5: Fix ScrollOffset mutation in View (value receiver issue)

**Files:**
- Modify: `internal/tui/model.go` (move scroll adjustment from View to Update)

**Context:** The `View()` method has a value receiver (`func (m Model) View() string`), which means `m.ScrollOffset = newOffset` inside `renderTree` is silently lost. Bubbletea's architecture requires all state mutation to happen in `Update()`. The scroll offset must be adjusted in `Update()` after cursor changes, not in `View()`.

**Step 1: Add scroll adjustment to cursor movement in Update()**

In `internal/tui/model.go`, in the Update method, after every cursor change (up/down/enter/expand/collapse), call a scroll adjustment helper:

```go
// adjustScroll updates ScrollOffset to keep cursor visible.
func (m *Model) adjustScroll() {
	treeHeight := m.Height - 4
	if treeHeight < 1 {
		treeHeight = 1
	}
	cursorLine := CursorToLine(m.Nodes, m.Cursor)
	_, _, m.ScrollOffset = VisibleRange(
		m.totalDisplayLines(), treeHeight, cursorLine, m.ScrollOffset,
	)
}

// totalDisplayLines returns the total number of display lines including blank separators.
func (m Model) totalDisplayLines() int {
	count := len(m.Nodes)
	// Add blank lines between repo groups (one before each repo after the first)
	for i, node := range m.Nodes {
		if node.Type == NodeRepo && i > 0 {
			count++
		}
	}
	return count
}
```

Then in `Update()`, call `m.adjustScroll()` after cursor changes:

```go
case "up", "k":
	if m.Cursor > 0 {
		m.Cursor--
		m.adjustScroll()
	}
case "down", "j":
	if m.Cursor < len(m.Nodes)-1 {
		m.Cursor++
		m.adjustScroll()
	}
```

Also call `m.adjustScroll()` after `m.Nodes = BuildNodes(m.Groups)` in the `refreshMsg` handler and all expand/collapse handlers.

**Step 2: Remove scroll mutation from View**

In `renderTree()`, change the scroll computation to be read-only:

```go
start, end, _ := VisibleRange(len(lines), treeHeight, cursorLine, m.ScrollOffset)
// Do NOT assign to m.ScrollOffset here — View has a value receiver
```

**Step 3: Run tests**

Run: `go test ./internal/tui/ -v && go vet ./...`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/tui/model.go internal/tui/view.go
git commit -m "fix(tui): move scroll adjustment to Update to fix value receiver mutation"
```

---

## Task 6: Final polish and integration verification

**Files:**
- Possibly adjust: `internal/tui/view.go` (padding, alignment tweaks)
- Possibly adjust: `internal/tui/theme.go` (color tweaks based on visual testing)

**Step 1: Run all unit tests**

Run: `go test ./internal/tui/ -v`
Expected: ALL PASS

**Step 2: Run full test suite**

Run: `go test ./... -v && go vet ./...`
Expected: ALL PASS (including any integration tests that don't require tmux)

**Step 3: Manual visual testing**

Run: `go run . dash`

Verify:
- [ ] Full-screen alternate screen mode (terminal taken over)
- [ ] Rounded border frame with "ClawdBay" title in top border
- [ ] Tree uses Kanagawa colors (purple repos, warm white sessions, muted gray windows)
- [ ] Cursor shows `❯` with pink text and subtle background highlight
- [ ] Status badges: green WORKING, orange IDLE, muted DONE
- [ ] Status summary bar below tree with separator
- [ ] Footer keybindings in bottom border, changes with selection
- [ ] Scrolling works when tree exceeds terminal height (test by collapsing/expanding)
- [ ] Pressing `q` cleanly exits and restores terminal
- [ ] Terminal resize updates layout dynamically
- [ ] Blank line between repo groups

**Step 4: Fix any visual issues**

Common adjustments:
- Padding values may need tweaking (try 1-3 chars)
- Border color may need more/less contrast
- Status bar separator alignment
- Selected row background width (should span full inner width)

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat(tui): complete Kanagawa Claw TUI redesign"
```

---

## Summary of changes

| File | Action | What changes |
|------|--------|-------------|
| `internal/tui/theme.go` | Create | Kanagawa Claw palette + style builders |
| `internal/tui/theme_test.go` | Create | Theme and style validation tests |
| `internal/tui/model.go` | Modify | Add Width/Height/ScrollOffset/Styles, WindowSizeMsg handler, SessionCounts, VisibleRange, CursorToLine, adjustScroll |
| `internal/tui/model_test.go` | Modify | Add SessionCounts, VisibleRange, CursorToLine tests |
| `internal/tui/view.go` | Rewrite | Full-screen themed rendering with composable sub-renders |
| `internal/tui/view_test.go` | Create | View rendering tests |
| `cmd/dash.go` | Modify | Add `tea.WithAltScreen()` |

**No new dependencies.** All changes use existing Bubbletea v1.3.10 and Lipgloss v1.1.0.
