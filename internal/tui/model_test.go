package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

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
		{"single working", []tmux.Status{tmux.StatusWorking}, tmux.StatusWorking},
		{"single waiting", []tmux.Status{tmux.StatusWaiting}, tmux.StatusWaiting},
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

func TestGroupByRepo(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "cb_feat-auth"},
		{Name: "cb_refactor"},
		{Name: "cb_fix-login"},
	}

	repoNames := map[string]string{
		"cb_feat-auth": "my-project",
		"cb_refactor":  "my-project",
		"cb_fix-login": "other-project",
	}

	windows := map[string][]tmux.Window{
		"cb_feat-auth": {
			{Index: 0, Name: "shell", Active: true},
			{Index: 1, Name: "claude", Active: false},
			{Index: 2, Name: "claude:research", Active: false},
		},
		"cb_refactor": {
			{Index: 0, Name: "shell", Active: true},
		},
		"cb_fix-login": {
			{Index: 0, Name: "shell", Active: true},
			{Index: 1, Name: "claude", Active: false},
		},
	}

	statuses := map[string]tmux.Status{
		"cb_feat-auth:claude":          tmux.StatusWorking,
		"cb_feat-auth:claude:research": tmux.StatusIdle,
		"cb_fix-login:claude":          tmux.StatusDone,
	}

	groups := GroupByRepo(sessions, repoNames, windows, statuses)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}

	// Verify ordering preserved
	if groups[0].Name != "my-project" {
		t.Errorf("first group = %q, want %q", groups[0].Name, "my-project")
	}
	if groups[1].Name != "other-project" {
		t.Errorf("second group = %q, want %q", groups[1].Name, "other-project")
	}

	// my-project should have 2 sessions
	if len(groups[0].Sessions) != 2 {
		t.Errorf("my-project has %d sessions, want 2", len(groups[0].Sessions))
	}

	// Check status rollup: feat-auth has WORKING and IDLE, should roll up to WORKING
	for _, s := range groups[0].Sessions {
		if s.Name == "cb_feat-auth" {
			if s.Status != tmux.StatusWorking {
				t.Errorf("cb_feat-auth status = %q, want %q", s.Status, tmux.StatusWorking)
			}
		}
	}

	// other-project has 1 session with DONE status
	if len(groups[1].Sessions) != 1 {
		t.Errorf("other-project has %d sessions, want 1", len(groups[1].Sessions))
	}
	if groups[1].Sessions[0].Status != tmux.StatusDone {
		t.Errorf("fix-login status = %q, want %q", groups[1].Sessions[0].Status, tmux.StatusDone)
	}
}

func TestGroupByRepo_UnknownRepo(t *testing.T) {
	sessions := []tmux.Session{{Name: "cb_orphan"}}
	repoNames := map[string]string{} // empty — no repo detected
	windows := map[string][]tmux.Window{}
	statuses := map[string]tmux.Status{}

	groups := GroupByRepo(sessions, repoNames, windows, statuses)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if groups[0].Name != "Unknown" {
		t.Errorf("group name = %q, want %q", groups[0].Name, "Unknown")
	}
}

func TestBuildNodes(t *testing.T) {
	groups := []RepoGroup{
		{
			Name:     "my-project",
			Expanded: true,
			Sessions: []WorktreeSession{
				{
					Name:     "cb_feat-auth",
					Status:   tmux.StatusWorking,
					Expanded: true,
					Windows: []tmux.Window{
						{Index: 0, Name: "shell"},
						{Index: 1, Name: "claude"},
					},
				},
				{
					Name:     "cb_refactor",
					Status:   tmux.StatusIdle,
					Expanded: false,
					Windows:  []tmux.Window{{Index: 0, Name: "shell"}},
				},
			},
		},
		{
			Name:     "other-project",
			Expanded: false,
			Sessions: nil,
		},
	}

	nodes := BuildNodes(groups)

	// Expected:
	// 0: Repo "my-project" (expanded)
	// 1: Session "cb_feat-auth" (expanded)
	// 2: Window "shell"
	// 3: Window "claude"
	// 4: Session "cb_refactor" (collapsed — no window children)
	// 5: Repo "other-project" (collapsed — no session children)
	if len(nodes) != 6 {
		t.Fatalf("got %d nodes, want 6", len(nodes))
	}

	if nodes[0].Type != NodeRepo {
		t.Errorf("node 0 type = %v, want NodeRepo", nodes[0].Type)
	}
	if nodes[1].Type != NodeSession {
		t.Errorf("node 1 type = %v, want NodeSession", nodes[1].Type)
	}
	if nodes[2].Type != NodeWindow {
		t.Errorf("node 2 type = %v, want NodeWindow", nodes[2].Type)
	}
	if nodes[3].Type != NodeWindow {
		t.Errorf("node 3 type = %v, want NodeWindow", nodes[3].Type)
	}
	if nodes[4].Type != NodeSession {
		t.Errorf("node 4 type = %v, want NodeSession", nodes[4].Type)
	}
	if nodes[5].Type != NodeRepo {
		t.Errorf("node 5 type = %v, want NodeRepo", nodes[5].Type)
	}
}

func TestBuildNodes_AllCollapsed(t *testing.T) {
	groups := []RepoGroup{
		{Name: "repo-a", Expanded: false},
		{Name: "repo-b", Expanded: false},
	}

	nodes := BuildNodes(groups)

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
}

func TestStatusRollup(t *testing.T) {
	tests := []struct {
		name     string
		statuses []tmux.Status
		expected tmux.Status
	}{
		{"all idle", []tmux.Status{tmux.StatusIdle, tmux.StatusIdle}, tmux.StatusIdle},
		{"one working", []tmux.Status{tmux.StatusIdle, tmux.StatusWorking}, tmux.StatusWorking},
		{"all done", []tmux.Status{tmux.StatusDone, tmux.StatusDone}, tmux.StatusDone},
		{"mixed", []tmux.Status{tmux.StatusDone, tmux.StatusIdle, tmux.StatusWorking}, tmux.StatusWorking},
		{"empty", []tmux.Status{}, tmux.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RollupStatus(tt.statuses)
			if result != tt.expected {
				t.Errorf("RollupStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSessionCounts(t *testing.T) {
	tests := []struct {
		name                                       string
		groups                                     []RepoGroup
		wantTotal, wantWork, wantWait, wantIdle int
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
			wantTotal: 3, wantWork: 1, wantWait: 0, wantIdle: 1,
		},
		{
			name: "with waiting",
			groups: []RepoGroup{
				{
					Name: "repo-a",
					Sessions: []WorktreeSession{
						{Name: "s1", Status: tmux.StatusWaiting},
						{Name: "s2", Status: tmux.StatusWorking},
					},
				},
			},
			wantTotal: 2, wantWork: 1, wantWait: 1, wantIdle: 0,
		},
		{
			name:      "empty",
			groups:    []RepoGroup{},
			wantTotal: 0, wantWork: 0, wantWait: 0, wantIdle: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{Groups: tt.groups}
			total, working, waiting, idle := m.SessionCounts()
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if working != tt.wantWork {
				t.Errorf("working = %d, want %d", working, tt.wantWork)
			}
			if waiting != tt.wantWait {
				t.Errorf("waiting = %d, want %d", waiting, tt.wantWait)
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
			name: "fits in view",
			lineCount: 5, viewHeight: 10, cursorLine: 2, scrollOff: 0,
			wantStart: 0, wantEnd: 5, wantScroll: 0,
		},
		{
			name: "cursor below viewport scrolls down",
			lineCount: 20, viewHeight: 10, cursorLine: 12, scrollOff: 0,
			wantStart: 3, wantEnd: 13, wantScroll: 3,
		},
		{
			name: "cursor above viewport scrolls up",
			lineCount: 20, viewHeight: 10, cursorLine: 2, scrollOff: 5,
			wantStart: 2, wantEnd: 12, wantScroll: 2,
		},
		{
			name: "cursor at end",
			lineCount: 20, viewHeight: 10, cursorLine: 19, scrollOff: 0,
			wantStart: 10, wantEnd: 20, wantScroll: 10,
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

func TestUpdate_CursorMovement(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo", Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "s1", Expanded: true, Windows: []tmux.Window{
						{Index: 0, Name: "shell"},
						{Index: 1, Name: "claude"},
					}},
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

	// Verify initial state: 4 nodes (repo, session, shell, claude)
	if len(m.Nodes) != 4 {
		t.Fatalf("got %d nodes, want 4", len(m.Nodes))
	}

	// Press "j" → cursor moves from 0 to 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.Cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.Cursor)
	}

	// Press "down" → cursor moves from 1 to 2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.Cursor != 2 {
		t.Errorf("after down: cursor = %d, want 2", m.Cursor)
	}

	// Press "k" → cursor moves from 2 to 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.Cursor != 1 {
		t.Errorf("after k: cursor = %d, want 1", m.Cursor)
	}

	// Press "up" → cursor moves from 1 to 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.Cursor != 0 {
		t.Errorf("after up: cursor = %d, want 0", m.Cursor)
	}

	// Press "up" at top → cursor stays at 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.Cursor != 0 {
		t.Errorf("after up at top: cursor = %d, want 0", m.Cursor)
	}
}

func TestUpdate_ExpandCollapse(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name: "repo", Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "s1", Expanded: true, Windows: []tmux.Window{
						{Index: 0, Name: "shell"},
					}},
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

	// Initial: 3 nodes (repo expanded, session expanded, window)
	if len(m.Nodes) != 3 {
		t.Fatalf("initial nodes = %d, want 3", len(m.Nodes))
	}

	// Press "h" on repo → collapse repo
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if len(m.Nodes) != 1 {
		t.Errorf("after h: nodes = %d, want 1 (repo collapsed)", len(m.Nodes))
	}
	if m.Groups[0].Expanded {
		t.Error("after h: repo should be collapsed")
	}

	// Press "l" on collapsed repo → expand repo
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if len(m.Nodes) != 3 {
		t.Errorf("after l: nodes = %d, want 3 (repo expanded)", len(m.Nodes))
	}
	if !m.Groups[0].Expanded {
		t.Error("after l: repo should be expanded")
	}

	// Press "enter" on repo → toggle (collapse)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(m.Nodes) != 1 {
		t.Errorf("after enter: nodes = %d, want 1 (repo collapsed)", len(m.Nodes))
	}
}

func TestUpdate_QuitKey(t *testing.T) {
	m := Model{
		Groups:         []RepoGroup{},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	result := updated.(Model)
	if !result.Quitting {
		t.Error("after q: Quitting should be true")
	}
	if cmd == nil {
		t.Error("after q: cmd should not be nil (should be tea.Quit)")
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

	tests := []struct {
		cursor   int
		wantLine int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 5}, // repo-b after blank line
		{5, 6},
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
