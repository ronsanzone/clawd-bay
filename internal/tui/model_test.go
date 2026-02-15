package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/discovery"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

type stubDiscoverer struct {
	result discovery.Result
	err    error
}

func (s stubDiscoverer) Discover() (discovery.Result, error) {
	return s.result, s.err
}

func TestRollupStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []tmux.Status
		want     tmux.Status
	}{
		{"working wins all", []tmux.Status{tmux.StatusDone, tmux.StatusWorking, tmux.StatusWaiting}, tmux.StatusWorking},
		{"waiting over idle", []tmux.Status{tmux.StatusIdle, tmux.StatusWaiting, tmux.StatusDone}, tmux.StatusWaiting},
		{"idle over done", []tmux.Status{tmux.StatusDone, tmux.StatusIdle}, tmux.StatusIdle},
		{"all done", []tmux.Status{tmux.StatusDone, tmux.StatusDone}, tmux.StatusDone},
		{"empty returns done", []tmux.Status{}, tmux.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RollupStatus(tt.statuses)
			if got != tt.want {
				t.Errorf("RollupStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildNodes_FourLevelHierarchy(t *testing.T) {
	groups := []RepoGroup{
		{
			Name:     "repo-a",
			Expanded: true,
			Worktrees: []WorktreeGroup{
				{
					Name:     "(main repo)",
					Expanded: true,
					Sessions: []WorktreeSession{
						{
							Name:     "cb_feat-auth",
							Expanded: true,
							Windows: []tmux.Window{
								{Index: 0, Name: "shell"},
								{Index: 1, Name: "claude"},
							},
						},
					},
				},
			},
		},
		{
			Name:     "repo-b",
			Expanded: false,
			Worktrees: []WorktreeGroup{
				{Name: "(main repo)", Expanded: false},
			},
		},
	}

	nodes := BuildNodes(groups)
	if len(nodes) != 6 {
		t.Fatalf("got %d nodes, want 6", len(nodes))
	}
	if nodes[0].Type != NodeRepo {
		t.Fatalf("node 0 = %v, want NodeRepo", nodes[0].Type)
	}
	if nodes[1].Type != NodeWorktree {
		t.Fatalf("node 1 = %v, want NodeWorktree", nodes[1].Type)
	}
	if nodes[2].Type != NodeSession {
		t.Fatalf("node 2 = %v, want NodeSession", nodes[2].Type)
	}
	if nodes[3].Type != NodeWindow || nodes[4].Type != NodeWindow {
		t.Fatalf("nodes 3/4 should be windows")
	}
	if nodes[5].Type != NodeRepo {
		t.Fatalf("node 5 = %v, want NodeRepo", nodes[5].Type)
	}
}

func TestSessionCounts(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo-a",
				Worktrees: []WorktreeGroup{
					{Sessions: []WorktreeSession{{Name: "s1", Status: tmux.StatusWorking}}},
					{Sessions: []WorktreeSession{{Name: "s2", Status: tmux.StatusWaiting}}},
				},
			},
			{
				Name: "repo-b",
				Worktrees: []WorktreeGroup{
					{Sessions: []WorktreeSession{{Name: "s3", Status: tmux.StatusIdle}, {Name: "s4", Status: tmux.StatusDone}}},
				},
			},
		},
	}

	total, working, waiting, idle := m.SessionCounts()
	if total != 4 || working != 1 || waiting != 1 || idle != 1 {
		t.Fatalf("counts = (%d,%d,%d,%d), want (4,1,1,1)", total, working, waiting, idle)
	}
}

func TestVisibleRange(t *testing.T) {
	start, end, offset := VisibleRange(20, 10, 12, 0)
	if start != 3 || end != 13 || offset != 3 {
		t.Fatalf("VisibleRange() = (%d,%d,%d), want (3,13,3)", start, end, offset)
	}
}

func TestCursorToLine(t *testing.T) {
	nodes := []TreeNode{
		{Type: NodeRepo},
		{Type: NodeWorktree},
		{Type: NodeSession},
		{Type: NodeWindow},
		{Type: NodeRepo},
		{Type: NodeWorktree},
	}
	if got := CursorToLine(nodes, 4); got != 5 {
		t.Fatalf("CursorToLine() = %d, want 5", got)
	}
}

func TestUpdate_ExpandCollapseProjectAndWorktree(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Worktrees: []WorktreeGroup{
					{
						Name:     "(main repo)",
						Expanded: true,
						Sessions: []WorktreeSession{{Name: "cb_one", Expanded: false}},
					},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if len(m.Nodes) != 1 {
		t.Fatalf("after collapsing repo nodes = %d, want 1", len(m.Nodes))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if len(m.Nodes) != 3 {
		t.Fatalf("after expanding repo nodes = %d, want 3", len(m.Nodes))
	}

	// move to worktree node and collapse it
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if len(m.Nodes) != 2 {
		t.Fatalf("after collapsing worktree nodes = %d, want 2", len(m.Nodes))
	}
}

func TestHandleEnter_WindowSetsSelectedWindowIndex(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Worktrees: []WorktreeGroup{
					{
						Name:     "(main repo)",
						Expanded: true,
						Sessions: []WorktreeSession{
							{
								Name:     "cb_test",
								Expanded: true,
								Windows: []tmux.Window{
									{Index: 0, Name: "shell"},
									{Index: 5, Name: "claude"},
								},
							},
						},
					},
				},
			},
		},
		Styles:              NewStyles(KanagawaClaw),
		WindowStatuses:      make(map[string]tmux.Status),
		SelectedWindowIndex: -1,
		Cursor:              4,
	}
	m.Nodes = BuildNodes(m.Groups)

	updated, cmd := m.handleEnter()
	result := updated.(Model)
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd when selecting window")
	}
	if result.SelectedName != "cb_test" {
		t.Fatalf("SelectedName = %q, want cb_test", result.SelectedName)
	}
	if result.SelectedWindowIndex != 5 {
		t.Fatalf("SelectedWindowIndex = %d, want 5", result.SelectedWindowIndex)
	}
}

func TestFilterModeMatchesWorktreeNames(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Worktrees: []WorktreeGroup{
					{Name: "(main repo)", Expanded: true},
					{Name: ".worktrees/repo-feature", Expanded: true},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("feature")})
	m = updated.(Model)

	if len(m.FilteredNodes) != 1 {
		t.Fatalf("len(FilteredNodes) = %d, want 1", len(m.FilteredNodes))
	}
	if m.FilteredNodes[0].Type != NodeWorktree {
		t.Fatalf("filtered node type = %v, want NodeWorktree", m.FilteredNodes[0].Type)
	}
}

func TestUpdateRefreshMsgSetsWindowAgentTypes(t *testing.T) {
	m := Model{
		Styles:              NewStyles(KanagawaClaw),
		WindowStatuses:      make(map[string]tmux.Status),
		WindowAgentTypes:    make(map[string]tmux.AgentType),
		SelectedWindowIndex: -1,
		Width:               80,
		Height:              24,
	}

	msg := refreshMsg{
		Groups: []RepoGroup{{
			Name:     "repo",
			Expanded: true,
			Worktrees: []WorktreeGroup{{
				Name:     "(main repo)",
				Expanded: true,
				Sessions: []WorktreeSession{{
					Name:     "cb_demo",
					Status:   tmux.StatusWorking,
					Expanded: true,
					Windows:  []tmux.Window{{Index: 1, Name: "custom-window"}},
				}},
			}},
		}},
		WindowStatuses: map[string]tmux.Status{"cb_demo:custom-window": tmux.StatusWorking},
		WindowAgents:   map[string]tmux.AgentType{"cb_demo:custom-window": tmux.AgentCodex},
	}

	updated, _ := m.Update(msg)
	out := updated.(Model)

	if got := out.WindowAgentTypes["cb_demo:custom-window"]; got != tmux.AgentCodex {
		t.Fatalf("WindowAgentTypes[...] = %q, want %q", got, tmux.AgentCodex)
	}
	if got := out.WindowStatuses["cb_demo:custom-window"]; got != tmux.StatusWorking {
		t.Fatalf("WindowStatuses[...] = %q, want %q", got, tmux.StatusWorking)
	}
}

func TestCursorToLine_Table(t *testing.T) {
	nodes := []TreeNode{
		{Type: NodeRepo},
		{Type: NodeWorktree},
		{Type: NodeRepo},
	}
	cases := []struct {
		cursor int
		want   int
	}{
		{0, 0},
		{1, 1},
		{2, 3},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("cursor_%d", tc.cursor), func(t *testing.T) {
			if got := CursorToLine(nodes, tc.cursor); got != tc.want {
				t.Fatalf("CursorToLine(%d) = %d, want %d", tc.cursor, got, tc.want)
			}
		})
	}
}

func TestBuildAgentNodes(t *testing.T) {
	rows := []AgentWindowRow{
		{SessionName: "cb_demo", WindowName: "claude", WindowIndex: 1},
		{SessionName: "other", WindowName: "codex", WindowIndex: 3},
	}

	nodes := BuildAgentNodes(rows)
	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want %d", len(nodes), 2)
	}
	if nodes[0].Type != NodeAgentWindow || nodes[0].AgentIndex != 0 {
		t.Fatalf("node[0] = %+v, want NodeAgentWindow index 0", nodes[0])
	}
	if nodes[1].Type != NodeAgentWindow || nodes[1].AgentIndex != 1 {
		t.Fatalf("node[1] = %+v, want NodeAgentWindow index 1", nodes[1])
	}
}

func TestAgentsModeFilterAndEnterSelectsWindowByIndex(t *testing.T) {
	m := Model{
		Mode: DashboardModeAgents,
		AgentRows: []AgentWindowRow{
			{
				SessionName: "cb_demo",
				WindowName:  "codex-main",
				WindowIndex: 9,
				RepoName:    "my-repo",
				AgentType:   tmux.AgentCodex,
				Status:      tmux.StatusWaiting,
				Managed:     true,
			},
			{
				SessionName: "other",
				WindowName:  "claude-main",
				WindowIndex: 2,
				RepoName:    "other-repo",
				AgentType:   tmux.AgentClaude,
				Status:      tmux.StatusWorking,
				Managed:     false,
			},
		},
		Styles:              NewStyles(KanagawaClaw),
		WindowStatuses:      make(map[string]tmux.Status),
		WindowAgentTypes:    make(map[string]tmux.AgentType),
		Width:               80,
		Height:              24,
		SelectedWindowIndex: -1,
	}
	m.Nodes = BuildAgentNodes(m.AgentRows)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("my-repo codex waiting")})
	m = updated.(Model)

	if len(m.FilteredNodes) != 1 {
		t.Fatalf("FilteredNodes len = %d, want %d", len(m.FilteredNodes), 1)
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updatedModel.(Model)
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd from enter on filtered agent row")
	}
	if result.SelectedName != "cb_demo" {
		t.Fatalf("SelectedName = %q, want %q", result.SelectedName, "cb_demo")
	}
	if result.SelectedWindowIndex != 9 {
		t.Fatalf("SelectedWindowIndex = %d, want %d", result.SelectedWindowIndex, 9)
	}
}

func TestToggleModeResetsFilterAndCursor(t *testing.T) {
	m := Model{
		Mode:           DashboardModeWorktree,
		Groups:         []RepoGroup{{Name: "repo", Expanded: true}},
		Nodes:          []TreeNode{{Type: NodeRepo, RepoIndex: 0}},
		Cursor:         3,
		FilterMode:     false,
		FilterQuery:    "abc",
		FilteredNodes:  []TreeNode{{Type: NodeRepo, RepoIndex: 0}},
		FilteredCursor: 1,
		ScrollOffset:   8,
		Styles:         NewStyles(KanagawaClaw),
		Width:          80,
		Height:         24,
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	updated := updatedModel.(Model)

	if cmd == nil {
		t.Fatal("expected refresh cmd when toggling mode")
	}
	if updated.Mode != DashboardModeAgents {
		t.Fatalf("Mode = %q, want %q", updated.Mode, DashboardModeAgents)
	}
	if updated.FilterMode {
		t.Fatal("FilterMode should be false after mode toggle")
	}
	if updated.FilterQuery != "" {
		t.Fatalf("FilterQuery = %q, want empty", updated.FilterQuery)
	}
	if updated.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0", updated.Cursor)
	}
	if updated.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d, want 0", updated.ScrollOffset)
	}
}

func TestAgentsModeIgnoresTreeAndCreateKeys(t *testing.T) {
	m := Model{
		Mode: DashboardModeAgents,
		AgentRows: []AgentWindowRow{
			{
				SessionName: "cb_demo",
				WindowName:  "codex-main",
				WindowIndex: 1,
				RepoName:    "repo",
				AgentType:   tmux.AgentCodex,
				Status:      tmux.StatusWorking,
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildAgentNodes(m.AgentRows)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected nil cmd for expand in agents mode")
	}
	if len(m.Nodes) != 1 || m.Nodes[0].Type != NodeAgentWindow {
		t.Fatalf("nodes changed unexpectedly: %+v", m.Nodes)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected nil cmd for add in agents mode")
	}
	if m.StatusMsg != "" {
		t.Fatalf("StatusMsg should remain empty, got %q", m.StatusMsg)
	}
}

func TestOpenAddDialogForNodeTypes(t *testing.T) {
	tests := []struct {
		name        string
		selectNode  func([]TreeNode) int
		wantKind    AddKind
		wantSession string
		wantWT      int
	}{
		{
			name: "repo opens main-worktree session dialog",
			selectNode: func(nodes []TreeNode) int {
				for i, n := range nodes {
					if n.Type == NodeRepo {
						return i
					}
				}
				return -1
			},
			wantKind: AddKindSession,
			wantWT:   0,
		},
		{
			name: "worktree opens session dialog for selected worktree",
			selectNode: func(nodes []TreeNode) int {
				for i, n := range nodes {
					if n.Type == NodeWorktree && n.WorktreeIndex == 1 {
						return i
					}
				}
				return -1
			},
			wantKind: AddKindSession,
			wantWT:   1,
		},
		{
			name: "session opens window dialog",
			selectNode: func(nodes []TreeNode) int {
				for i, n := range nodes {
					if n.Type == NodeSession && n.WorktreeIndex == 1 {
						return i
					}
				}
				return -1
			},
			wantKind:    AddKindWindow,
			wantSession: "cb_feat",
		},
		{
			name: "window opens parent session window dialog",
			selectNode: func(nodes []TreeNode) int {
				for i, n := range nodes {
					if n.Type == NodeWindow && n.WorktreeIndex == 0 {
						return i
					}
				}
				return -1
			},
			wantKind:    AddKindWindow,
			wantSession: "cb_main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := addDialogTestModel()
			idx := tt.selectNode(m.Nodes)
			if idx < 0 {
				t.Fatal("test node not found")
			}
			m.Cursor = idx

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
			got := updated.(Model)
			if cmd != nil {
				t.Fatal("expected nil command when opening add dialog")
			}
			if !got.AddDialog.Active {
				t.Fatal("expected active add dialog")
			}
			if got.AddDialog.Kind != tt.wantKind {
				t.Fatalf("AddDialog.Kind = %v, want %v", got.AddDialog.Kind, tt.wantKind)
			}
			if tt.wantSession != "" && got.AddDialog.SessionName != tt.wantSession {
				t.Fatalf("AddDialog.SessionName = %q, want %q", got.AddDialog.SessionName, tt.wantSession)
			}
			if tt.wantKind == AddKindSession && got.AddDialog.WorktreeIdx != tt.wantWT {
				t.Fatalf("AddDialog.WorktreeIdx = %d, want %d", got.AddDialog.WorktreeIdx, tt.wantWT)
			}
		})
	}
}

func TestAddDialogInputHandling(t *testing.T) {
	m := addDialogTestModel()
	m.AddDialog = AddDialogState{
		Active:      true,
		Kind:        AddKindSession,
		RepoIndex:   0,
		WorktreeIdx: 0,
		Input:       "ab",
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(Model)
	if m.AddDialog.Input != "abc" {
		t.Fatalf("input after rune = %q, want %q", m.AddDialog.Input, "abc")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	if m.AddDialog.Input != "ab" {
		t.Fatalf("input after backspace = %q, want %q", m.AddDialog.Input, "ab")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.AddDialog.Active {
		t.Fatal("dialog should be inactive after esc")
	}
}

func TestSubmitAddDialogEmptySanitizedInputShowsError(t *testing.T) {
	m := addDialogTestModel()
	m.AddDialog = AddDialogState{
		Active:      true,
		Kind:        AddKindWindow,
		SessionName: "cb_main",
		Input:       "!!!",
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if cmd != nil {
		t.Fatal("expected nil command on validation failure")
	}
	if !got.AddDialog.Active {
		t.Fatal("dialog should remain open on validation failure")
	}
	if got.AddDialog.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestSanitizeAddName(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "lower and trim", raw: " Feature Branch ", want: "feature-branch"},
		{name: "keep slash and underscore", raw: "API_v2/Review", want: "api_v2/review"},
		{name: "collapse dashes", raw: "alpha   beta---gamma", want: "alpha-beta-gamma"},
		{name: "trim edge separators", raw: "/demo-path/-", want: "demo-path"},
		{name: "drop invalid", raw: "###", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeAddName(tt.raw); got != tt.want {
				t.Fatalf("sanitizeAddName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestEnsureSessionPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "cb_demo", want: "cb_demo"},
		{in: "demo", want: "cb_demo"},
		{in: "", want: "cb_"},
	}

	for _, tt := range tests {
		if got := ensureSessionPrefix(tt.in); got != tt.want {
			t.Fatalf("ensureSessionPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUniquifyName(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		existing map[string]struct{}
		want     string
	}{
		{name: "unused base", base: "demo", existing: map[string]struct{}{}, want: "demo"},
		{name: "first suffix", base: "demo", existing: map[string]struct{}{"demo": {}}, want: "demo-2"},
		{name: "next suffix", base: "demo", existing: map[string]struct{}{"demo": {}, "demo-2": {}}, want: "demo-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniquifyName(tt.base, func(name string) bool {
				_, ok := tt.existing[name]
				return ok
			})
			if got != tt.want {
				t.Fatalf("uniquifyName(%q) = %q, want %q", tt.base, got, tt.want)
			}
		})
	}
}

func TestFetchGroups_MapsSessionFields(t *testing.T) {
	discoverer := stubDiscoverer{
		result: discovery.Result{
			Projects: []discovery.ProjectNode{{
				Name: "repo",
				Path: "/tmp/repo",
				Worktrees: []discovery.WorktreeNode{{
					Name:       "(main repo)",
					Path:       "/tmp/repo",
					IsMainRepo: true,
					Sessions: []discovery.SessionNode{{
						Name:    "cb_demo",
						Status:  tmux.StatusWaiting,
						Windows: []tmux.Window{{Index: 1, Name: "claude"}},
					}},
				}},
			}},
		},
	}

	groups, _, _, _, err := fetchGroups(discoverer)
	if err != nil {
		t.Fatalf("fetchGroups() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].Worktrees) != 1 || len(groups[0].Worktrees[0].Sessions) != 1 {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	session := groups[0].Worktrees[0].Sessions[0]
	if session.Name != "cb_demo" {
		t.Fatalf("Name = %q, want cb_demo", session.Name)
	}
	if session.Status != tmux.StatusWaiting {
		t.Fatalf("Status = %q, want %q", session.Status, tmux.StatusWaiting)
	}
}

func TestUpdate_EscQuitsOutsideFilterMode(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Worktrees: []WorktreeGroup{
					{Name: "(main repo)", Expanded: true},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := updatedModel.(Model)

	if cmd == nil {
		t.Fatal("expected tea.Quit cmd on esc outside filter mode")
	}
	if !updated.Quitting {
		t.Fatal("expected Quitting=true on esc outside filter mode")
	}
}

func TestUpdate_EscClearsFilterModeWithoutQuit(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Worktrees: []WorktreeGroup{
					{Name: "(main repo)", Expanded: true},
				},
			},
		},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		FilterMode:     true,
		FilterQuery:    "repo",
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)
	m.FilteredNodes = append([]TreeNode(nil), m.Nodes...)

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := updatedModel.(Model)

	if cmd != nil {
		t.Fatal("expected nil cmd on esc in filter mode")
	}
	if updated.Quitting {
		t.Fatal("expected Quitting=false on esc in filter mode")
	}
	if updated.FilterMode {
		t.Fatal("expected filter mode to be cleared on esc")
	}
	if updated.FilterQuery != "" {
		t.Fatalf("FilterQuery = %q, want empty", updated.FilterQuery)
	}
}

func addDialogTestModel() Model {
	groups := []RepoGroup{
		{
			Name:     "repo",
			Expanded: true,
			Worktrees: []WorktreeGroup{
				{
					Name:       "(main repo)",
					Path:       "/tmp/repo",
					IsMainRepo: true,
					Expanded:   true,
					Sessions: []WorktreeSession{
						{
							Name:     "cb_main",
							Expanded: true,
							Windows:  []tmux.Window{{Index: 0, Name: "shell"}},
						},
					},
				},
				{
					Name:       ".worktrees/repo-feat",
					Path:       "/tmp/repo/.worktrees/repo-feat",
					IsMainRepo: false,
					Expanded:   true,
					Sessions: []WorktreeSession{
						{
							Name:     "cb_feat",
							Expanded: true,
							Windows:  []tmux.Window{{Index: 1, Name: "work"}},
						},
					},
				},
			},
		},
	}

	m := Model{
		Groups:           groups,
		Styles:           NewStyles(KanagawaClaw),
		WindowStatuses:   make(map[string]tmux.Status),
		WindowAgentTypes: make(map[string]tmux.AgentType),
		Width:            80,
		Height:           24,
	}
	m.Nodes = BuildNodes(m.Groups)
	return m
}
