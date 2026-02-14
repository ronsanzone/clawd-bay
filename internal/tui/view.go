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

func (m Model) modeLabel() DashboardMode {
	if m.Mode == DashboardModeAgents {
		return DashboardModeAgents
	}
	return DashboardModeWorktree
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
	nodes := m.nodesForView()
	if len(nodes) == 0 {
		if m.FilterMode {
			return "No matches.\n  Press esc to clear filter."
		}
		if m.Mode == DashboardModeAgents {
			return "No detected agent windows.\n  Start an agent in any tmux window."
		}
		if m.ConfigMissing {
			return "No project config found.\n  Add one with: cb project add <path>"
		}
		return "No configured projects.\n  Add one with: cb project add <path>"
	}

	lines := m.buildDisplayLines(nodes)
	treeHeight := m.treeHeight()

	cursorLine := m.cursorForView()
	if !m.FilterMode && m.Mode != DashboardModeAgents {
		cursorLine = CursorToLine(nodes, cursorLine)
	}
	start, end, _ := VisibleRange(len(lines), treeHeight, cursorLine, m.ScrollOffset)

	visibleLines := lines[start:end]

	var result []string
	for _, line := range visibleLines {
		result = append(result, padToWidth(line, width))
	}

	for len(result) < treeHeight {
		result = append(result, strings.Repeat(" ", width))
	}

	return strings.Join(result, "\n")
}

// buildDisplayLines renders all tree nodes to display lines.
func (m Model) buildDisplayLines(nodes []TreeNode) []string {
	var lines []string

	for i, node := range nodes {
		// Insert blank separator before each repo (except first) in normal tree mode.
		if m.Mode != DashboardModeAgents && !m.FilterMode && node.Type == NodeRepo && i > 0 {
			lines = append(lines, "")
		}

		lines = append(lines, m.renderNodeLine(node, i))
	}

	return lines
}

// renderNodeLine renders one tree node.
func (m Model) renderNodeLine(node TreeNode, nodeIdx int) string {
	selected := nodeIdx == m.cursorForView()
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
		if repo.InvalidError != "" {
			line = cursor + icon + " " + m.Styles.Repo.Render(repo.Name) + " " + m.Styles.StatusWaiting.Render("[INVALID]")
		} else {
			line = cursor + icon + " " + m.Styles.Repo.Render(repo.Name)
		}

	case NodeWorktree:
		worktree := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex]
		icon := "▸"
		if worktree.Expanded {
			icon = "▼"
		}
		line = cursor + "  " + icon + " " + m.Styles.StatusDone.Render(worktree.Name)

	case NodeSession:
		session := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex]
		icon := "▸"
		if session.Expanded {
			icon = "▼"
		}
		badge := m.renderStatusBadge(session.Status)
		line = cursor + "    " + icon + " " + badge + " " + m.Styles.Session.Render(session.Name)

	case NodeWindow:
		session := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		key := session.Name + ":" + window.Name
		badge := " "
		if status, ok := m.WindowStatuses[key]; ok {
			badge = m.renderStatusBadge(status)
		}
		tag := m.renderAgentTag(m.WindowAgentTypes[key])
		if tag != "" {
			line = cursor + "      " + badge + " " + tag + " " + m.Styles.Window.Render(window.Name)
		} else {
			line = cursor + "      " + badge + " " + m.Styles.Window.Render(window.Name)
		}

	case NodeAgentWindow:
		row := m.AgentRows[node.AgentIndex]
		target := fmt.Sprintf("%s:%d", row.SessionName, row.WindowIndex)
		repo := row.RepoName
		if repo == "" {
			repo = "Unknown"
		}
		tag := m.renderAgentTag(row.AgentType)
		badge := m.renderStatusBadge(row.Status)
		line = cursor + badge + " " + tag + " " + m.Styles.Window.Render(row.WindowName) +
			"  " + m.Styles.Session.Render(target) +
			"  " + m.Styles.StatusBar.Render("repo="+repo)

	default:
		line = cursor + "Unknown"
	}

	if selected {
		line = m.Styles.Selected.Render(line)
	}

	return line
}

func (m Model) renderAgentTag(agentType tmux.AgentType) string {
	switch agentType {
	case tmux.AgentClaude:
		return m.Styles.StatusBar.Render("[CLAUDE]")
	case tmux.AgentCodex:
		return m.Styles.StatusBar.Render("[CODEX]")
	case tmux.AgentOpenCode:
		return m.Styles.StatusBar.Render("[OPEN]")
	default:
		return ""
	}
}

// renderStatusBadge renders a colored status badge.
func (m Model) renderStatusBadge(status tmux.Status) string {
	switch status {
	case tmux.StatusWorking:
		return m.Styles.StatusWorking.Render("•")
	case tmux.StatusWaiting:
		return m.Styles.StatusWaiting.Render("◐")
	case tmux.StatusIdle:
		return m.Styles.StatusIdle.Render("◦")
	default:
		return m.Styles.StatusDone.Render("·")
	}
}

// renderStatusBar renders the session count summary.
func (m Model) renderStatusBar() string {
	total, working, waiting, idle := m.SessionCounts()

	var parts []string
	if m.modeLabel() == DashboardModeAgents {
		parts = append(parts, fmt.Sprintf("mode: %s", DashboardModeAgents))
		parts = append(parts, fmt.Sprintf("%d agent windows", total))
	} else {
		parts = append(parts, fmt.Sprintf("mode: %s", DashboardModeWorktree))
		parts = append(parts, fmt.Sprintf("%d sessions", total))
	}

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
	if m.FilterMode {
		return fmt.Sprintf("filter: %q  ·  type to search  ·  j/k navigate  ·  enter select  ·  esc clear  ·  m mode", m.FilterQuery)
	}

	if m.Cursor >= len(m.Nodes) {
		return "/ filter  ·  j/k navigate  ·  m mode  ·  q quit"
	}

	if m.Mode == DashboardModeAgents {
		return "/ filter  ·  j/k navigate  ·  enter attach  ·  m mode  ·  r refresh  ·  q quit"
	}

	node := m.Nodes[m.Cursor]
	switch node.Type {
	case NodeRepo:
		return "/ filter  ·  j/k navigate  ·  enter toggle  ·  h/l collapse/expand  ·  m mode  ·  q quit"
	case NodeWorktree:
		return "/ filter  ·  j/k navigate  ·  enter toggle  ·  h/l collapse/expand  ·  m mode  ·  q quit"
	case NodeSession:
		return "/ filter  ·  j/k navigate  ·  enter attach  ·  c claude  ·  h collapse  ·  m mode  ·  r refresh  ·  q quit"
	case NodeWindow:
		return "/ filter  ·  j/k navigate  ·  enter attach  ·  c claude  ·  h collapse  ·  m mode  ·  r refresh  ·  q quit"
	default:
		return "/ filter  ·  j/k navigate  ·  q quit"
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
	title := m.Styles.Title.Render(fmt.Sprintf(" ClawdBay · %s ", m.modeLabel()))
	titleW := lipgloss.Width(title)
	topLine := bStyle.Render(border.TopLeft+border.Top) +
		title +
		bStyle.Render(strings.Repeat(border.Top, max(0, w-titleW-3))+border.TopRight)

	midLine := bStyle.Render(border.MiddleLeft) +
		bStyle.Render(strings.Repeat(border.Top, w-2)) +
		bStyle.Render(border.MiddleRight)

	footerText := m.Styles.Footer.Render(" " + footer + " ")
	footerW := lipgloss.Width(footerText)
	botLine := bStyle.Render(border.BottomLeft+border.Bottom) +
		footerText +
		bStyle.Render(strings.Repeat(border.Bottom, max(0, w-footerW-3))+border.BottomRight)

	side := bStyle.Render(border.Left)
	sideR := bStyle.Render(border.Right)

	var lines []string
	lines = append(lines, topLine)

	for _, cl := range strings.Split(tree, "\n") {
		lines = append(lines, side+padToWidth(cl, w-2)+sideR)
	}

	lines = append(lines, midLine)
	lines = append(lines, side+padToWidth(statusBar, w-2)+sideR)
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
