package tui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/rsanzone/clawdbay/internal/discovery"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

const refreshInterval = 3 * time.Second

// tickMsg triggers periodic refresh.
type tickMsg time.Time

// refreshMsg carries new data from a refresh.
type refreshMsg struct {
	Groups         []RepoGroup
	AgentRows      []AgentWindowRow
	WindowStatuses map[string]tmux.Status
	WindowAgents   map[string]tmux.AgentType
	ConfigMissing  bool
	Err            error
}

// AddKind identifies which add flow is active.
type AddKind int

const (
	AddKindNone AddKind = iota
	AddKindSession
	AddKindWindow
)

// AddDialogState stores state for the add name dialog.
type AddDialogState struct {
	Active      bool
	Kind        AddKind
	Input       string
	Error       string
	RepoIndex   int
	WorktreeIdx int
	SessionName string
}

// addResultMsg is sent after attempting to create a session or window.
type addResultMsg struct {
	Kind   AddKind
	Name   string
	Target string
	Err    error
}

// NodeType represents what kind of tree node the cursor is on.
type NodeType int

const (
	// NodeRepo is a configured project node.
	NodeRepo NodeType = iota
	// NodeWorktree is a discovered worktree node.
	NodeWorktree
	// NodeSession is a tmux session node.
	NodeSession
	// NodeWindow is a tmux window node.
	NodeWindow
	// NodeAgentWindow is a flat agent window row in agents mode.
	NodeAgentWindow
)

// DashboardMode controls which dashboard representation is shown.
type DashboardMode string

const (
	DashboardModeWorktree DashboardMode = "worktree"
	DashboardModeAgents   DashboardMode = "agents"
)

// ParseDashboardMode parses a user-supplied mode string.
func ParseDashboardMode(raw string) (DashboardMode, error) {
	mode := DashboardMode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return DashboardModeWorktree, nil
	}
	switch mode {
	case DashboardModeWorktree, DashboardModeAgents:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid dashboard mode %q (valid: %s, %s)", raw, DashboardModeWorktree, DashboardModeAgents)
	}
}

// RepoGroup represents a configured project and its discovered worktrees.
type RepoGroup struct {
	Name         string
	Path         string
	InvalidError string
	Worktrees    []WorktreeGroup
	Expanded     bool
}

// WorktreeGroup represents one discovered worktree path under a project.
type WorktreeGroup struct {
	Name       string
	Path       string
	IsMainRepo bool
	Sessions   []WorktreeSession
	Expanded   bool
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
	Type          NodeType
	RepoIndex     int
	WorktreeIndex int
	SessionIndex  int
	WindowIndex   int
	AgentIndex    int
}

// AgentWindowRow represents one detected coding-agent window across all tmux sessions.
type AgentWindowRow struct {
	SessionName string
	WindowName  string
	WindowIndex int
	RepoName    string
	AgentType   tmux.AgentType
	Status      tmux.Status
	Managed     bool
}

// Discoverer loads the project/worktree/session hierarchy.
type Discoverer interface {
	Discover() (discovery.Result, error)
}

// Model is the Bubbletea model for the dashboard.
type Model struct {
	Mode                DashboardMode
	Groups              []RepoGroup
	AgentRows           []AgentWindowRow
	Cursor              int
	Nodes               []TreeNode
	FilterMode          bool
	FilterQuery         string
	FilteredNodes       []TreeNode
	FilteredCursor      int
	Quitting            bool
	TmuxClient          *tmux.Client
	Discoverer          Discoverer
	SelectedName        string
	SelectedWindow      string
	SelectedWindowIndex int
	WindowStatuses      map[string]tmux.Status
	WindowAgentTypes    map[string]tmux.AgentType
	Width               int
	Height              int
	ScrollOffset        int
	Styles              Styles
	StatusMsg           string
	ConfigMissing       bool
	AddDialog           AddDialogState
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
	if m.Mode == DashboardModeAgents {
		for _, row := range m.AgentRows {
			total++
			switch row.Status {
			case tmux.StatusWorking:
				working++
			case tmux.StatusWaiting:
				waiting++
			case tmux.StatusIdle:
				idle++
			}
		}
		return
	}

	for _, g := range m.Groups {
		for _, wt := range g.Worktrees {
			for _, s := range wt.Sessions {
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
	}
	return
}

// BuildNodes flattens the tree into a list of navigable nodes.
func BuildNodes(groups []RepoGroup) []TreeNode {
	var nodes []TreeNode

	for ri, repo := range groups {
		nodes = append(nodes, TreeNode{Type: NodeRepo, RepoIndex: ri})

		if !repo.Expanded {
			continue
		}

		for wi, worktree := range repo.Worktrees {
			nodes = append(nodes, TreeNode{Type: NodeWorktree, RepoIndex: ri, WorktreeIndex: wi})

			if !worktree.Expanded {
				continue
			}

			for si, session := range worktree.Sessions {
				nodes = append(nodes, TreeNode{Type: NodeSession, RepoIndex: ri, WorktreeIndex: wi, SessionIndex: si})

				if !session.Expanded {
					continue
				}

				for widx := range session.Windows {
					nodes = append(nodes, TreeNode{Type: NodeWindow, RepoIndex: ri, WorktreeIndex: wi, SessionIndex: si, WindowIndex: widx})
				}
			}
		}
	}

	return nodes
}

// BuildAgentNodes flattens agent rows into a list of navigable nodes.
func BuildAgentNodes(rows []AgentWindowRow) []TreeNode {
	nodes := make([]TreeNode, 0, len(rows))
	for i := range rows {
		nodes = append(nodes, TreeNode{Type: NodeAgentWindow, AgentIndex: i})
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
// accounting for blank separator lines between project groups.
func CursorToLine(nodes []TreeNode, cursor int) int {
	line := 0
	for i := 0; i < cursor && i < len(nodes); i++ {
		line++
		if i+1 < len(nodes) && nodes[i+1].Type == NodeRepo {
			line++
		}
	}
	return line
}

// InitialModel creates the initial dashboard model.
func InitialModel(tmuxClient *tmux.Client) Model {
	return InitialModelWithMode(tmuxClient, DashboardModeWorktree)
}

// InitialModelWithMode creates the initial dashboard model with an explicit mode.
func InitialModelWithMode(tmuxClient *tmux.Client, mode DashboardMode) Model {
	return Model{
		Mode:                mode,
		Groups:              []RepoGroup{},
		AgentRows:           []AgentWindowRow{},
		TmuxClient:          tmuxClient,
		Discoverer:          discovery.NewService(tmuxClient),
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
		groups, rows, statuses, agents, missing, err := fetchDashboardData(m.Discoverer, m.TmuxClient, m.Mode)
		return refreshMsg{
			Groups:         groups,
			AgentRows:      rows,
			WindowStatuses: statuses,
			WindowAgents:   agents,
			ConfigMissing:  missing,
			Err:            err,
		}
	}
}

// fetchDashboardData queries tmux for all data needed by the selected mode.
func fetchDashboardData(
	discoverer Discoverer,
	tmuxClient *tmux.Client,
	mode DashboardMode,
) ([]RepoGroup, []AgentWindowRow, map[string]tmux.Status, map[string]tmux.AgentType, bool, error) {
	switch mode {
	case DashboardModeAgents:
		rows, statuses, agents := fetchAgentRowsData(tmuxClient)
		return nil, rows, statuses, agents, false, nil
	default:
		groups, statuses, agents, missing, err := fetchGroups(discoverer)
		return groups, nil, statuses, agents, missing, err
	}
}

// fetchGroups queries shared discovery data.
func fetchGroups(discoverer Discoverer) ([]RepoGroup, map[string]tmux.Status, map[string]tmux.AgentType, bool, error) {
	slog.Debug("fetchGroups called")
	if discoverer == nil {
		slog.Debug("fetchGroups: discoverer is nil")
		return nil, map[string]tmux.Status{}, map[string]tmux.AgentType{}, false, nil
	}

	result, err := discoverer.Discover()
	if err != nil {
		return nil, nil, nil, false, err
	}

	groups := make([]RepoGroup, 0, len(result.Projects))
	for _, p := range result.Projects {
		group := RepoGroup{
			Name:         p.Name,
			Path:         p.Path,
			InvalidError: p.InvalidError,
			Expanded:     true,
			Worktrees:    make([]WorktreeGroup, 0, len(p.Worktrees)),
		}
		for _, wt := range p.Worktrees {
			worktree := WorktreeGroup{
				Name:       wt.Name,
				Path:       wt.Path,
				IsMainRepo: wt.IsMainRepo,
				Expanded:   true,
				Sessions:   make([]WorktreeSession, 0, len(wt.Sessions)),
			}
			for _, s := range wt.Sessions {
				worktree.Sessions = append(worktree.Sessions, WorktreeSession{
					Name:     s.Name,
					Status:   s.Status,
					Windows:  s.Windows,
					Expanded: true,
				})
			}
			group.Worktrees = append(group.Worktrees, worktree)
		}
		groups = append(groups, group)
	}

	return groups, result.WindowStatuses, result.WindowAgents, result.ConfigMissing, nil
}

func fetchAgentRowsData(tmuxClient *tmux.Client) ([]AgentWindowRow, map[string]tmux.Status, map[string]tmux.AgentType) {
	slog.Debug("fetchAgentRowsData called")
	if tmuxClient == nil {
		slog.Debug("fetchAgentRowsData: tmuxClient is nil")
		return nil, map[string]tmux.Status{}, map[string]tmux.AgentType{}
	}

	infos, err := tmuxClient.ListSessionWindowInfo()
	if err != nil {
		slog.Debug("fetchAgentRowsData: ListSessionWindowInfo failed", "err", err)
		return nil, map[string]tmux.Status{}, map[string]tmux.AgentType{}
	}

	rows := make([]AgentWindowRow, 0, len(infos))
	statusMap := make(map[string]tmux.Status)
	agentMap := make(map[string]tmux.AgentType)

	for _, info := range infos {
		if !info.AgentInfo.Detected {
			continue
		}

		row := AgentWindowRow{
			SessionName: info.SessionName,
			WindowName:  info.Window.Name,
			WindowIndex: info.Window.Index,
			RepoName:    info.RepoName,
			AgentType:   info.AgentInfo.Type,
			Status:      info.AgentInfo.Status,
			Managed:     info.Managed,
		}
		rows = append(rows, row)

		key := row.SessionName + ":" + row.WindowName
		statusMap[key] = row.Status
		agentMap[key] = row.AgentType
	}

	return rows, statusMap, agentMap
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
	if m.Mode == DashboardModeAgents {
		return len(m.Nodes)
	}

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
		group := m.Groups[node.RepoIndex]
		return group.Name + " " + group.Path
	case NodeWorktree:
		group := m.Groups[node.RepoIndex]
		worktree := group.Worktrees[node.WorktreeIndex]
		return worktree.Name + " " + worktree.Path + " " + group.Name
	case NodeSession:
		group := m.Groups[node.RepoIndex]
		worktree := group.Worktrees[node.WorktreeIndex]
		session := worktree.Sessions[node.SessionIndex]
		return session.Name + " " + worktree.Name + " " + group.Name
	case NodeWindow:
		group := m.Groups[node.RepoIndex]
		worktree := group.Worktrees[node.WorktreeIndex]
		session := worktree.Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		return window.Name + " " + session.Name + " " + worktree.Name + " " + group.Name
	case NodeAgentWindow:
		row := m.AgentRows[node.AgentIndex]
		return strings.Join([]string{
			row.WindowName,
			row.SessionName,
			row.RepoName,
			string(row.AgentType),
			string(row.Status),
		}, " ")
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
		if msg.Err != nil {
			m.StatusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.ConfigMissing = msg.ConfigMissing

		if m.Mode == DashboardModeAgents {
			m.AgentRows = msg.AgentRows
			m.Nodes = BuildAgentNodes(m.AgentRows)
			m.Groups = nil
		} else {
			m.Groups = mergeExpandState(m.Groups, msg.Groups)
			m.Nodes = BuildNodes(m.Groups)
			m.AgentRows = nil
		}
		m.WindowStatuses = msg.WindowStatuses
		m.WindowAgentTypes = msg.WindowAgents
		if m.FilterMode {
			m.updateFilteredNodes()
		}
		if m.Cursor >= len(m.Nodes) {
			m.Cursor = max(0, len(m.Nodes)-1)
		}
		m.adjustScroll()
		return m, nil

	case addResultMsg:
		if msg.Err != nil {
			m.StatusMsg = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			switch msg.Kind {
			case AddKindSession:
				m.StatusMsg = fmt.Sprintf("Session created: %s", msg.Name)
			case AddKindWindow:
				m.StatusMsg = fmt.Sprintf("Window created: %s", msg.Name)
			default:
				m.StatusMsg = "Created"
			}
		}
		return m, m.refreshCmd()

	case tickMsg:
		m.StatusMsg = ""
		return m, tea.Batch(m.refreshCmd(), m.tickCmd())

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.AddDialog.Active {
			switch msg.String() {
			case "esc":
				m.AddDialog = AddDialogState{}
				return m, nil
			case "backspace", "ctrl+h":
				if m.AddDialog.Input != "" {
					runes := []rune(m.AddDialog.Input)
					m.AddDialog.Input = string(runes[:len(runes)-1])
					m.AddDialog.Error = ""
				}
				return m, nil
			case "enter":
				return m.submitAddDialog()
			}

			if len(msg.Runes) > 0 {
				m.AddDialog.Input += string(msg.Runes)
				m.AddDialog.Error = ""
			}
			return m, nil
		}

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
		case "q", "esc", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "m":
			m.toggleMode()
			return m, m.refreshCmd()
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
			if m.Mode == DashboardModeAgents {
				return m, nil
			}
			return m.handleExpand()
		case "h", "left":
			if m.Mode == DashboardModeAgents {
				return m, nil
			}
			return m.handleCollapse()
		case "a":
			if m.Mode == DashboardModeAgents {
				return m, nil
			}
			if m.Cursor >= len(m.Nodes) {
				return m, nil
			}
			return m.openAddDialogForNode(m.Nodes[m.Cursor])
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

func (m *Model) toggleMode() {
	if m.Mode == DashboardModeAgents {
		m.Mode = DashboardModeWorktree
	} else {
		m.Mode = DashboardModeAgents
	}

	m.Cursor = 0
	m.Nodes = nil
	m.Groups = nil
	m.AgentRows = nil
	m.ScrollOffset = 0

	m.FilterMode = false
	m.FilterQuery = ""
	m.FilteredNodes = nil
	m.FilteredCursor = 0
	m.AddDialog = AddDialogState{}
}

// mergeExpandState preserves expand/collapse state across refreshes.
func mergeExpandState(old, updated []RepoGroup) []RepoGroup {
	repoState := make(map[string]bool)
	worktreeState := make(map[string]bool)
	sessionState := make(map[string]bool)

	for _, g := range old {
		repoKey := g.Path
		if repoKey == "" {
			repoKey = g.Name
		}
		repoState[repoKey] = g.Expanded
		for _, wt := range g.Worktrees {
			worktreeKey := repoKey + "|" + wt.Path
			worktreeState[worktreeKey] = wt.Expanded
			for _, s := range wt.Sessions {
				sessionKey := worktreeKey + "|" + s.Name
				sessionState[sessionKey] = s.Expanded
			}
		}
	}

	for i := range updated {
		repoKey := updated[i].Path
		if repoKey == "" {
			repoKey = updated[i].Name
		}
		if expanded, ok := repoState[repoKey]; ok {
			updated[i].Expanded = expanded
		}
		for wi := range updated[i].Worktrees {
			worktreeKey := repoKey + "|" + updated[i].Worktrees[wi].Path
			if expanded, ok := worktreeState[worktreeKey]; ok {
				updated[i].Worktrees[wi].Expanded = expanded
			}
			for si := range updated[i].Worktrees[wi].Sessions {
				sessionKey := worktreeKey + "|" + updated[i].Worktrees[wi].Sessions[si].Name
				if expanded, ok := sessionState[sessionKey]; ok {
					updated[i].Worktrees[wi].Sessions[si].Expanded = expanded
				}
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
	case NodeWorktree:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Expanded = !m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Expanded
		m.Nodes = BuildNodes(m.Groups)
		if m.FilterMode {
			m.updateFilteredNodes()
		}
		m.adjustScroll()
	case NodeSession:
		session := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex]
		m.SelectedName = session.Name
		m.SelectedWindowIndex = -1
		return m, tea.Quit
	case NodeWindow:
		session := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		m.SelectedName = session.Name
		m.SelectedWindow = window.Name
		m.SelectedWindowIndex = window.Index
		return m, tea.Quit
	case NodeAgentWindow:
		row := m.AgentRows[node.AgentIndex]
		m.SelectedName = row.SessionName
		m.SelectedWindow = row.WindowName
		m.SelectedWindowIndex = row.WindowIndex
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
	case NodeWorktree:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Expanded = true
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeSession:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex].Expanded = true
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
	case NodeWorktree:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeSession:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	case NodeWindow:
		m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
		m.adjustScroll()
	}
	return m, nil
}

func (m Model) openAddDialogForNode(node TreeNode) (Model, tea.Cmd) {
	switch node.Type {
	case NodeRepo:
		if node.RepoIndex < 0 || node.RepoIndex >= len(m.Groups) {
			return m, nil
		}
		group := m.Groups[node.RepoIndex]
		mainIdx := -1
		for i, wt := range group.Worktrees {
			if wt.IsMainRepo {
				mainIdx = i
				break
			}
		}
		if mainIdx == -1 {
			m.StatusMsg = fmt.Sprintf("Error: main worktree not found for %s", group.Name)
			return m, nil
		}
		m.AddDialog = AddDialogState{
			Active:      true,
			Kind:        AddKindSession,
			RepoIndex:   node.RepoIndex,
			WorktreeIdx: mainIdx,
		}
		return m, nil
	case NodeWorktree:
		if node.RepoIndex < 0 || node.RepoIndex >= len(m.Groups) {
			return m, nil
		}
		if node.WorktreeIndex < 0 || node.WorktreeIndex >= len(m.Groups[node.RepoIndex].Worktrees) {
			return m, nil
		}
		m.AddDialog = AddDialogState{
			Active:      true,
			Kind:        AddKindSession,
			RepoIndex:   node.RepoIndex,
			WorktreeIdx: node.WorktreeIndex,
		}
		return m, nil
	case NodeSession, NodeWindow:
		if node.RepoIndex < 0 || node.RepoIndex >= len(m.Groups) {
			return m, nil
		}
		if node.WorktreeIndex < 0 || node.WorktreeIndex >= len(m.Groups[node.RepoIndex].Worktrees) {
			return m, nil
		}
		worktree := m.Groups[node.RepoIndex].Worktrees[node.WorktreeIndex]
		if node.SessionIndex < 0 || node.SessionIndex >= len(worktree.Sessions) {
			return m, nil
		}
		sessionName := worktree.Sessions[node.SessionIndex].Name
		m.AddDialog = AddDialogState{
			Active:      true,
			Kind:        AddKindWindow,
			RepoIndex:   node.RepoIndex,
			WorktreeIdx: node.WorktreeIndex,
			SessionName: sessionName,
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) submitAddDialog() (tea.Model, tea.Cmd) {
	dialog := m.AddDialog
	rawName := dialog.Input
	sanitized := sanitizeAddName(rawName)
	if sanitized == "" {
		m.AddDialog.Error = "name is required"
		return m, nil
	}

	client := m.TmuxClient
	if client == nil {
		m.AddDialog.Error = "tmux client is not available"
		return m, nil
	}

	switch dialog.Kind {
	case AddKindSession:
		if dialog.RepoIndex < 0 || dialog.RepoIndex >= len(m.Groups) {
			m.AddDialog.Error = "target repo no longer exists"
			return m, nil
		}
		group := m.Groups[dialog.RepoIndex]
		if dialog.WorktreeIdx < 0 || dialog.WorktreeIdx >= len(group.Worktrees) {
			m.AddDialog.Error = "target worktree no longer exists"
			return m, nil
		}
		worktreePath := group.Worktrees[dialog.WorktreeIdx].Path
		candidate := ensureSessionPrefix(sanitized)
		if candidate == "cb_" {
			m.AddDialog.Error = "name is required"
			return m, nil
		}

		m.AddDialog = AddDialogState{}
		m.StatusMsg = fmt.Sprintf("Creating session %s...", candidate)
		return m, func() tea.Msg {
			sessions, err := client.ListSessions()
			if err != nil {
				return addResultMsg{Kind: AddKindSession, Err: err}
			}

			existing := make(map[string]struct{}, len(sessions))
			for _, s := range sessions {
				existing[s.Name] = struct{}{}
			}
			finalName := uniquifyName(candidate, func(name string) bool {
				_, ok := existing[name]
				return ok
			})

			if err := client.CreateSession(finalName, worktreePath); err != nil {
				return addResultMsg{Kind: AddKindSession, Name: finalName, Target: worktreePath, Err: err}
			}

			canonicalPath, err := config.CanonicalPath(worktreePath)
			if err != nil {
				return addResultMsg{Kind: AddKindSession, Name: finalName, Target: worktreePath, Err: err}
			}
			if err := client.SetSessionOption(finalName, tmux.SessionOptionHomePath, canonicalPath); err != nil {
				return addResultMsg{Kind: AddKindSession, Name: finalName, Target: worktreePath, Err: err}
			}

			return addResultMsg{Kind: AddKindSession, Name: finalName, Target: worktreePath}
		}
	case AddKindWindow:
		sessionName := dialog.SessionName
		if sessionName == "" {
			m.AddDialog.Error = "target session no longer exists"
			return m, nil
		}

		// Best effort dedupe from the current model snapshot.
		existing := make(map[string]struct{})
		if dialog.RepoIndex >= 0 && dialog.RepoIndex < len(m.Groups) &&
			dialog.WorktreeIdx >= 0 && dialog.WorktreeIdx < len(m.Groups[dialog.RepoIndex].Worktrees) {
			worktree := m.Groups[dialog.RepoIndex].Worktrees[dialog.WorktreeIdx]
			for _, session := range worktree.Sessions {
				if session.Name != sessionName {
					continue
				}
				for _, w := range session.Windows {
					existing[w.Name] = struct{}{}
				}
				break
			}
		}
		windowName := uniquifyName(sanitized, func(name string) bool {
			_, ok := existing[name]
			return ok
		})

		m.AddDialog = AddDialogState{}
		m.StatusMsg = fmt.Sprintf("Creating window %s...", windowName)
		return m, func() tea.Msg {
			err := client.CreateWindow(sessionName, windowName, "")
			return addResultMsg{
				Kind:   AddKindWindow,
				Name:   windowName,
				Target: sessionName,
				Err:    err,
			}
		}
	default:
		m.AddDialog.Error = "invalid add action"
		return m, nil
	}
}

func sanitizeAddName(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/':
			b.WriteRune(r)
		}
	}

	sanitized := b.String()
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}
	return strings.Trim(sanitized, "-/")
}

func ensureSessionPrefix(name string) string {
	if strings.HasPrefix(name, "cb_") {
		return name
	}
	return "cb_" + name
}

func uniquifyName(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !exists(candidate) {
			return candidate
		}
	}
}
