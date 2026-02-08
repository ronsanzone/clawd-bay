package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

const maxPanelWidth = 100

// frameWidth returns the panel width, capped at maxPanelWidth.
func (m Model) frameWidth() int {
	return min(m.Width, maxPanelWidth)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	if m.Width == 0 || m.Height == 0 {
		return "Initializing..."
	}

	fw := m.frameWidth()
	innerWidth := fw - 2
	if innerWidth < 10 {
		innerWidth = 10
	}

	tree := m.renderTree(innerWidth)
	statusBar := m.renderStatusBar()
	footer := m.renderFooter()

	frame := m.renderFrame(tree, statusBar, footer)

	// Center the panel in the terminal
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, frame)
}

// renderTree renders the scrollable tree content.
func (m Model) renderTree(width int) string {
	if len(m.Nodes) == 0 {
		return "No active sessions.\n  Start one with: cb start <branch-name>"
	}

	lines := m.buildDisplayLines()
	treeHeight := m.treeHeight()

	cursorLine := CursorToLine(m.Nodes, m.Cursor)
	start, end, _ := VisibleRange(len(lines), treeHeight, cursorLine, m.ScrollOffset)

	visibleLines := lines[start:end]

	var result []string
	for _, line := range visibleLines {
		result = append(result, padToWidth(line, width))
	}

	// Pad remaining lines to fill treeHeight
	for len(result) < treeHeight {
		result = append(result, strings.Repeat(" ", width))
	}

	return strings.Join(result, "\n")
}

// buildDisplayLines renders all tree nodes to display lines.
func (m Model) buildDisplayLines() []string {
	var lines []string

	for i, node := range m.Nodes {
		// Insert blank separator before each repo (except first)
		if node.Type == NodeRepo && i > 0 {
			lines = append(lines, "")
		}

		lines = append(lines, m.renderNodeLine(node, i))
	}

	return lines
}

// renderNodeLine renders one tree node.
func (m Model) renderNodeLine(node TreeNode, nodeIdx int) string {
	selected := nodeIdx == m.Cursor
	cursor := "  "
	if selected {
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
		line = cursor + icon + " " + m.Styles.Repo.Render(repo.Name)

	case NodeSession:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		icon := "▸"
		if session.Expanded {
			icon = "▼"
		}
		badge := m.renderStatusBadge(session.Status)
		left := cursor + "  " + icon + " " + m.Styles.Session.Render(session.Name)
		line = m.rightAlign(left, badge)

	case NodeWindow:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		badge := ""
		if strings.HasPrefix(window.Name, "claude") {
			key := session.Name + ":" + window.Name
			if status, ok := m.WindowStatuses[key]; ok {
				badge = m.renderStatusBadge(status)
			}
		}
		left := cursor + "      " + m.Styles.Window.Render(window.Name)
		line = m.rightAlign(left, badge)

	default:
		line = cursor + "Unknown"
	}

	if selected {
		line = m.Styles.Selected.Render(line)
	}

	return line
}

// rightAlign aligns the right string to the right edge.
func (m Model) rightAlign(left, right string) string {
	if right == "" {
		return left
	}

	available := m.frameWidth() - 4
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := available - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderStatusBadge renders a colored status badge.
// Badge symbols show activity level: ● (working) > ◉ (waiting) > ○ (idle) > ◌ (done)
func (m Model) renderStatusBadge(status tmux.Status) string {
	switch status {
	case tmux.StatusWorking:
		return m.Styles.StatusWorking.Render("● WORKING")
	case tmux.StatusWaiting:
		return m.Styles.StatusWaiting.Render("◉ WAITING")
	case tmux.StatusIdle:
		return m.Styles.StatusIdle.Render("○ IDLE")
	default: // StatusDone
		return m.Styles.StatusDone.Render("◌ DONE")
	}
}

// renderStatusBar renders the session count summary.
func (m Model) renderStatusBar() string {
	total, working, waiting, idle := m.SessionCounts()

	var parts []string
	parts = append(parts, fmt.Sprintf("%d sessions", total))

	if working > 0 {
		parts = append(parts, m.Styles.StatusWorking.Render(fmt.Sprintf("%d working", working)))
	}
	if waiting > 0 {
		parts = append(parts, m.Styles.StatusWaiting.Render(fmt.Sprintf("%d waiting", waiting)))
	}
	if idle > 0 {
		parts = append(parts, m.Styles.StatusIdle.Render(fmt.Sprintf("%d idle", idle)))
	}

	if m.StatusMsg != "" {
		parts = append(parts, m.Styles.StatusDone.Render(m.StatusMsg))
	}

	sep := m.Styles.StatusBar.Render(" · ")
	return "  " + strings.Join(parts, sep)
}

// renderFooter renders context-sensitive keybindings.
func (m Model) renderFooter() string {
	if m.Cursor >= len(m.Nodes) {
		return "j/k navigate  ·  q quit"
	}

	node := m.Nodes[m.Cursor]
	switch node.Type {
	case NodeRepo:
		return "j/k navigate  ·  enter toggle  ·  h/l collapse/expand  ·  q quit"
	case NodeSession:
		return "j/k navigate  ·  enter attach  ·  c claude  ·  h collapse  ·  r refresh  ·  q quit"
	case NodeWindow:
		return "j/k navigate  ·  enter attach  ·  c claude  ·  h collapse  ·  r refresh  ·  q quit"
	default:
		return "j/k navigate  ·  q quit"
	}
}

// renderFrame builds the bordered frame manually.
func (m Model) renderFrame(tree, statusBar, footer string) string {
	w := m.frameWidth()
	if w < 20 {
		w = 20
	}

	border := lipgloss.RoundedBorder()
	bStyle := lipgloss.NewStyle().Foreground(m.Styles.Frame.GetBorderTopForeground())

	// Top border with title: ╭─ ClawdBay ─────────────────╮
	title := m.Styles.Title.Render(" ClawdBay ")
	titleW := lipgloss.Width(title)
	topLine := bStyle.Render(border.TopLeft+border.Top) +
		title +
		bStyle.Render(strings.Repeat(border.Top, max(0, w-titleW-3))+border.TopRight)

	// Middle separator: ├─────────────────────────────┤
	midLine := bStyle.Render(border.MiddleLeft) +
		bStyle.Render(strings.Repeat(border.Top, w-2)) +
		bStyle.Render(border.MiddleRight)

	// Bottom border with footer: ╰─ enter attach · q quit ────╯
	footerText := m.Styles.Footer.Render(" " + footer + " ")
	footerW := lipgloss.Width(footerText)
	botLine := bStyle.Render(border.BottomLeft+border.Bottom) +
		footerText +
		bStyle.Render(strings.Repeat(border.Bottom, max(0, w-footerW-3))+border.BottomRight)

	// Side borders for content
	side := bStyle.Render(border.Left)
	sideR := bStyle.Render(border.Right)

	var lines []string
	lines = append(lines, topLine)

	// Tree content with side borders
	for _, cl := range strings.Split(tree, "\n") {
		lines = append(lines, side+padToWidth(cl, w-2)+sideR)
	}

	// Separator + status bar
	lines = append(lines, midLine)
	lines = append(lines, side+padToWidth(statusBar, w-2)+sideR)

	// Bottom
	lines = append(lines, botLine)

	return strings.Join(lines, "\n")
}

// padToWidth pads a string to exact visual width.
func padToWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
