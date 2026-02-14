package tui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

const refreshInterval = 3 * time.Second

// tickMsg triggers periodic refresh.
type tickMsg time.Time

// refreshMsg carries new data from a refresh.
type refreshMsg struct {
	Groups         []RepoGroup
	WindowStatuses map[string]tmux.Status
	WindowAgents   map[string]tmux.AgentType
}

// claudeWindowMsg is sent after attempting to create a Claude window.
type claudeWindowMsg struct {
	Err error
}

// NodeType represents what kind of tree node the cursor is on.
type NodeType int

const (
	// NodeRepo is a repository group node.
	NodeRepo NodeType = iota
	// NodeSession is a worktree session node.
	NodeSession
	// NodeWindow is a tmux window node.
	NodeWindow
)

// RepoGroup represents a repository with its worktree sessions.
type RepoGroup struct {
	Name     string
	Path     string
	Sessions []WorktreeSession
	Expanded bool
}

// WorktreeSession represents a tmux session tied to a worktree.
type WorktreeSession struct {
	Name     string
	Status   tmux.Status
	Windows  []tmux.Window
	Expanded bool
}

// TreeNode represents a flattened position in the tree for cursor navigation.
type TreeNode struct {
	Type         NodeType
	RepoIndex    int
	SessionIndex int
	WindowIndex  int
}

// Model is the Bubbletea model for the dashboard.
type Model struct {
	Groups              []RepoGroup
	Cursor              int
	Nodes               []TreeNode
	FilterMode          bool
	FilterQuery         string
	FilteredNodes       []TreeNode
	FilteredCursor      int
	Quitting            bool
	TmuxClient          *tmux.Client
	SelectedName        string
	SelectedWindow      string
	SelectedWindowIndex int
	WindowStatuses      map[string]tmux.Status
	WindowAgentTypes    map[string]tmux.AgentType
	Width               int
	Height              int
	ScrollOffset        int
	Styles              Styles
	StatusMsg           string // transient feedback message shown in status bar
}

// RollupStatus returns the most active status from a slice.
// Priority: WORKING > WAITING > IDLE > DONE
func RollupStatus(statuses []tmux.Status) tmux.Status {
	hasWaiting := false
	hasIdle := false
	for _, s := range statuses {
		switch s {
		case tmux.StatusWorking:
			return tmux.StatusWorking
		case tmux.StatusWaiting:
			hasWaiting = true
		case tmux.StatusIdle:
			hasIdle = true
		}
	}
	if hasWaiting {
		return tmux.StatusWaiting
	}
	if hasIdle {
		return tmux.StatusIdle
	}
	return tmux.StatusDone
}

// SessionCounts returns total sessions and counts by status.
func (m Model) SessionCounts() (total, working, waiting, idle int) {
	for _, g := range m.Groups {
		for _, s := range g.Sessions {
			total++
			switch s.Status {
			case tmux.StatusWorking:
				working++
			case tmux.StatusWaiting:
				waiting++
			case tmux.StatusIdle:
				idle++
			}
		}
	}
	return
}

// GroupByRepo groups sessions by their repository name.
func GroupByRepo(
	sessions []tmux.Session,
	repoNames map[string]string,
	windows map[string][]tmux.Window,
	statuses map[string]tmux.Status,
) []RepoGroup {
	repoMap := make(map[string]*RepoGroup)
	var repoOrder []string

	for _, session := range sessions {
		repoName := repoNames[session.Name]
		if repoName == "" {
			repoName = "Unknown"
		}

		if _, exists := repoMap[repoName]; !exists {
			repoMap[repoName] = &RepoGroup{
				Name:     repoName,
				Expanded: true,
			}
			repoOrder = append(repoOrder, repoName)
		}

		wins := windows[session.Name]
		var windowStatuses []tmux.Status
		for _, w := range wins {
			key := session.Name + ":" + w.Name
			if status, ok := statuses[key]; ok {
				windowStatuses = append(windowStatuses, status)
			}
		}

		ws := WorktreeSession{
			Name:     session.Name,
			Status:   RollupStatus(windowStatuses),
			Windows:  wins,
			Expanded: true,
		}

		repoMap[repoName].Sessions = append(repoMap[repoName].Sessions, ws)
	}

	var groups []RepoGroup
	for _, name := range repoOrder {
		groups = append(groups, *repoMap[name])
	}
	return groups
}

// BuildNodes flattens the tree into a list of navigable nodes.
func BuildNodes(groups []RepoGroup) []TreeNode {
	var nodes []TreeNode

	for ri, repo := range groups {
		nodes = append(nodes, TreeNode{Type: NodeRepo, RepoIndex: ri})

		if !repo.Expanded {
			continue
		}

		for si, session := range repo.Sessions {
			nodes = append(nodes, TreeNode{Type: NodeSession, RepoIndex: ri, SessionIndex: si})

			if !session.Expanded {
				continue
			}

			for wi := range session.Windows {
				nodes = append(nodes, TreeNode{Type: NodeWindow, RepoIndex: ri, SessionIndex: si, WindowIndex: wi})
			}
		}
	}

	return nodes
}

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

// CursorToLine maps a cursor position (node index) to a display line index,
// accounting for blank separator lines between repo groups.
func CursorToLine(nodes []TreeNode, cursor int) int {
	line := 0
	for i := 0; i < cursor && i < len(nodes); i++ {
		line++
		// A repo node that isn't the first means a blank line was inserted before it
		if i+1 < len(nodes) && nodes[i+1].Type == NodeRepo {
			line++ // blank separator line
		}
	}
	return line
}

// InitialModel creates the initial dashboard model.
func InitialModel(tmuxClient *tmux.Client) Model {
	return Model{
		Groups:              []RepoGroup{},
		TmuxClient:          tmuxClient,
		WindowStatuses:      make(map[string]tmux.Status),
		WindowAgentTypes:    make(map[string]tmux.AgentType),
		SelectedWindowIndex: -1,
		Styles:              NewStyles(KanagawaClaw),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.tickCmd())
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		groups, statuses, agents := fetchGroups(m.TmuxClient)
		return refreshMsg{Groups: groups, WindowStatuses: statuses, WindowAgents: agents}
	}
}

// fetchGroups queries tmux for all data.
func fetchGroups(tmuxClient *tmux.Client) ([]RepoGroup, map[string]tmux.Status, map[string]tmux.AgentType) {
	slog.Debug("fetchGroups called")
	if tmuxClient == nil {
		slog.Debug("fetchGroups: tmuxClient is nil")
		return nil, nil, nil
	}

	sessions, err := tmuxClient.ListSessions()
	if err != nil {
		slog.Debug("fetchGroups: ListSessions failed", "err", err)
		return nil, nil, nil
	}
	slog.Debug("fetchGroups: found sessions", "count", len(sessions))

	repoNames := make(map[string]string)
	windowMap := make(map[string][]tmux.Window)
	statusMap := make(map[string]tmux.Status)
	agentMap := make(map[string]tmux.AgentType)

	for _, s := range sessions {
		repoNames[s.Name] = tmuxClient.GetRepoName(s.Name)

		wins, winErr := tmuxClient.ListWindows(s.Name)
		if winErr != nil {
			continue
		}
		windowMap[s.Name] = wins

		for _, w := range wins {
			key := s.Name + ":" + w.Name
			info := tmuxClient.DetectAgentInfo(s.Name, w.Name)
			if info.Detected {
				statusMap[key] = info.Status
				agentMap[key] = info.Type
			}
		}
	}

	return GroupByRepo(sessions, repoNames, windowMap, statusMap), statusMap, agentMap
}

// adjustScroll updates ScrollOffset to keep cursor visible in the viewport.
func (m *Model) adjustScroll() {
	treeHeight := m.treeHeight()
	if treeHeight < 1 {
		return
	}

	activeNodes := m.nodesForView()
	if len(activeNodes) == 0 {
		m.ScrollOffset = 0
		return
	}

	cursorLine := m.cursorForView()
	lineCount := len(activeNodes)
	if !m.FilterMode {
		cursorLine = CursorToLine(activeNodes, cursorLine)
		lineCount = m.totalDisplayLines()
	}

	_, _, m.ScrollOffset = VisibleRange(
		lineCount, treeHeight, cursorLine, m.ScrollOffset,
	)
}

// treeHeight returns the number of lines available for the tree view.
// Accounts for borders (2), status bar (1), and frame padding (1).
func (m Model) treeHeight() int {
	h := max(m.Height-4, 1)
	return h
}

// totalDisplayLines returns the total number of display lines including blank separators.
func (m Model) totalDisplayLines() int {
	count := len(m.Nodes)
	for i, node := range m.Nodes {
		if node.Type == NodeRepo && i > 0 {
			count++
		}
	}
	return count
}

func (m *Model) updateFilteredNodes() {
	query := strings.ToLower(strings.TrimSpace(m.FilterQuery))
	if query == "" {
		m.FilteredNodes = append([]TreeNode(nil), m.Nodes...)
	} else {
		m.FilteredNodes = m.FilteredNodes[:0]
		for _, node := range m.Nodes {
			if strings.Contains(strings.ToLower(m.filterSearchText(node)), query) {
				m.FilteredNodes = append(m.FilteredNodes, node)
			}
		}
	}

	if m.FilteredCursor >= len(m.FilteredNodes) {
		m.FilteredCursor = max(0, len(m.FilteredNodes)-1)
	}
	if m.FilteredCursor < 0 {
		m.FilteredCursor = 0
	}
}

func (m Model) filterSearchText(node TreeNode) string {
	switch node.Type {
	case NodeRepo:
		return m.Groups[node.RepoIndex].Name
	case NodeSession:
		group := m.Groups[node.RepoIndex]
		session := group.Sessions[node.SessionIndex]
		return session.Name + " " + group.Name
	case NodeWindow:
		group := m.Groups[node.RepoIndex]
		session := group.Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		return window.Name + " " + session.Name + " " + group.Name
	default:
		return ""
	}
}

func (m Model) nodesForView() []TreeNode {
	if m.FilterMode {
		return m.FilteredNodes
	}
	return m.Nodes
}

func (m Model) cursorForView() int {
	if m.FilterMode {
		return m.FilteredCursor
	}
	return m.Cursor
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		m.Groups = mergeExpandState(m.Groups, msg.Groups)
		m.WindowStatuses = msg.WindowStatuses
		m.WindowAgentTypes = msg.WindowAgents
		m.Nodes = BuildNodes(m.Groups)
		if m.FilterMode {
			m.updateFilteredNodes()
		}
		if m.Cursor >= len(m.Nodes) {
			m.Cursor = max(0, len(m.Nodes)-1)
		}
		m.adjustScroll()
		return m, nil

	case claudeWindowMsg:
		if msg.Err != nil {
			m.StatusMsg = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.StatusMsg = "Claude window created"
		}
		return m, m.refreshCmd()

	case tickMsg:
		m.StatusMsg = "" // clear transient messages on tick
		return m, tea.Batch(m.refreshCmd(), m.tickCmd())

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.FilterMode {
			switch msg.String() {
			case "esc":
				m.FilterMode = false
				m.FilterQuery = ""
				m.FilteredNodes = nil
				m.FilteredCursor = 0
				m.adjustScroll()
				return m, nil
			case "backspace", "ctrl+h":
				if m.FilterQuery != "" {
					runes := []rune(m.FilterQuery)
					m.FilterQuery = string(runes[:len(runes)-1])
				}
				m.updateFilteredNodes()
				m.adjustScroll()
				return m, nil
			case "up", "k":
				if m.FilteredCursor > 0 {
					m.FilteredCursor--
					m.adjustScroll()
				}
				return m, nil
			case "down", "j":
				if m.FilteredCursor < len(m.FilteredNodes)-1 {
					m.FilteredCursor++
					m.adjustScroll()
				}
				return m, nil
			case "enter":
				return m.handleEnter()
			}

			if len(msg.Runes) > 0 {
				m.FilterQuery += string(msg.Runes)
				m.updateFilteredNodes()
				m.adjustScroll()
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
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
		case "enter":
			return m.handleEnter()
		case "l", "right":
			return m.handleExpand()
		case "h", "left":
			return m.handleCollapse()
		case "c":
			return m.handleAddClaude()
		case "r":
			return m, m.refreshCmd()
		case "/":
			m.FilterMode = true
			m.FilterQuery = ""
			m.FilteredCursor = 0
			m.updateFilteredNodes()
			m.adjustScroll()
		}
	}
	return m, nil
}

// mergeExpandState preserves expand/collapse state across refreshes.
func mergeExpandState(old, updated []RepoGroup) []RepoGroup {
	oldState := make(map[string]bool)
	oldSessionState := make(map[string]bool)

	for _, g := range old {
		oldState[g.Name] = g.Expanded
		for _, s := range g.Sessions {
			oldSessionState[s.Name] = s.Expanded
		}
	}

	for i := range updated {
		if expanded, ok := oldState[updated[i].Name]; ok {
			updated[i].Expanded = expanded
		}
		for j := range updated[i].Sessions {
			if expanded, ok := oldSessionState[updated[i].Sessions[j].Name]; ok {
				updated[i].Sessions[j].Expanded = expanded
			}
		}
	}
	return updated
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	activeNodes := m.nodesForView()
	activeCursor := m.cursorForView()
	if activeCursor >= len(activeNodes) {
		return m, nil
	}
	node := activeNodes[activeCursor]

	switch node.Type {
	case NodeRepo:
		m.Groups[node.RepoIndex].Expanded = !m.Groups[node.RepoIndex].Expanded
		m.Nodes = BuildNodes(m.Groups)
		if m.FilterMode {
			m.updateFilteredNodes()
		}
		m.adjustScroll()
	case NodeSession:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		m.SelectedName = session.Name
		m.SelectedWindowIndex = -1
		return m, tea.Quit
	case NodeWindow:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		m.SelectedName = session.Name
		m.SelectedWindow = window.Name
		m.SelectedWindowIndex = window.Index
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleExpand() (tea.Model, tea.Cmd) {
	if m.Cursor >= len(m.Nodes) {
		return m, nil
	}
	node := m.Nodes[m.Cursor]

	switch node.Type {
	case NodeRepo:
		m.Groups[node.RepoIndex].Expanded = true
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeSession:
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = true
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	}
	return m, nil
}

func (m Model) handleCollapse() (tea.Model, tea.Cmd) {
	if m.Cursor >= len(m.Nodes) {
		return m, nil
	}
	node := m.Nodes[m.Cursor]

	switch node.Type {
	case NodeRepo:
		m.Groups[node.RepoIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeSession:
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeWindow:
		// Collapse parent session
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	}
	return m, nil
}

// handleAddClaude creates a new Claude window in the session under the cursor.
// Works when the cursor is on a session or window node.
func (m Model) handleAddClaude() (tea.Model, tea.Cmd) {
	if m.Cursor >= len(m.Nodes) {
		return m, nil
	}
	node := m.Nodes[m.Cursor]

	var sessionName string
	switch node.Type {
	case NodeSession:
		sessionName = m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Name
	case NodeWindow:
		sessionName = m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Name
	default:
		return m, nil
	}

	// Count existing claude windows to generate a unique name
	session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
	claudeCount := 0
	for _, w := range session.Windows {
		if strings.HasPrefix(w.Name, "claude") {
			claudeCount++
		}
	}

	windowName := "claude"
	if claudeCount > 0 {
		windowName = fmt.Sprintf("claude:%d", claudeCount+1)
	}

	m.StatusMsg = fmt.Sprintf("Creating %s in %s...", windowName, sessionName)
	client := m.TmuxClient
	return m, func() tea.Msg {
		err := client.CreateWindowWithShell(sessionName, windowName, "claude")
		return claudeWindowMsg{Err: err}
	}
}
