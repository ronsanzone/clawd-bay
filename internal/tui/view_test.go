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
		{tmux.StatusWorking, "•"},
		{tmux.StatusIdle, "◦"},
		{tmux.StatusDone, "·"},
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

	lines := m.buildDisplayLines(m.Nodes)

	// Expect: repo-a, s1, shell, <blank>, repo-b, s2 = 6 lines
	if len(lines) != 6 {
		t.Errorf("got %d display lines, want 6", len(lines))
	}

	// The blank separator between repo groups
	if len(lines) > 3 && strings.TrimSpace(lines[3]) != "" {
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

	if !strings.Contains(view, "ClawdBay") {
		t.Error("view missing ClawdBay title")
	}

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") {
		t.Error("view missing rounded border characters")
	}

	if !strings.Contains(view, "quit") {
		t.Error("view missing footer keybindings")
	}
}

func TestViewEmptyState(t *testing.T) {
	m := Model{
		Groups:         []RepoGroup{},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)

	view := m.View()

	if !strings.Contains(view, "No active sessions") {
		t.Error("empty view missing 'No active sessions' message")
	}
}

func TestViewFilterModeHint(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Sessions: []WorktreeSession{
					{Name: "cb_test", Expanded: true, Windows: []tmux.Window{{Name: "shell"}}},
				},
			},
		},
		FilterMode:     true,
		FilterQuery:    "missing",
		FilteredNodes:  []TreeNode{},
		WindowStatuses: make(map[string]tmux.Status),
		Styles:         NewStyles(KanagawaClaw),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildNodes(m.Groups)

	view := m.View()
	if !strings.Contains(view, `filter: "missing"`) {
		t.Fatalf("view missing filter hint: %q", view)
	}
	if !strings.Contains(view, "No matches") {
		t.Fatalf("view missing no matches message: %q", view)
	}
}

func TestRenderNodeLineWindowIncludesAgentTag(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Sessions: []WorktreeSession{
					{
						Name:     "cb_demo",
						Expanded: true,
						Windows: []tmux.Window{
							{Index: 3, Name: "workbench"},
						},
					},
				},
			},
		},
		WindowStatuses: map[string]tmux.Status{
			"cb_demo:workbench": tmux.StatusWorking,
		},
		WindowAgentTypes: map[string]tmux.AgentType{
			"cb_demo:workbench": tmux.AgentCodex,
		},
		Styles: NewStyles(KanagawaClaw),
		Width:  80,
		Cursor: 2,
	}
	m.Nodes = BuildNodes(m.Groups)

	line := m.renderNodeLine(m.Nodes[2], 2)
	if !strings.Contains(line, "[CODEX]") {
		t.Fatalf("window line missing [CODEX] tag: %q", line)
	}
	if !strings.Contains(line, "•") {
		t.Fatalf("window line missing status badge: %q", line)
	}
}

func TestRenderNodeLineWindowNoAgentTagForNone(t *testing.T) {
	m := Model{
		Groups: []RepoGroup{
			{
				Name:     "repo",
				Expanded: true,
				Sessions: []WorktreeSession{
					{
						Name:     "cb_demo",
						Expanded: true,
						Windows: []tmux.Window{
							{Index: 1, Name: "shell"},
						},
					},
				},
			},
		},
		WindowStatuses: map[string]tmux.Status{
			"cb_demo:shell": tmux.StatusDone,
		},
		WindowAgentTypes: map[string]tmux.AgentType{
			"cb_demo:shell": tmux.AgentNone,
		},
		Styles: NewStyles(KanagawaClaw),
		Width:  80,
		Cursor: 2,
	}
	m.Nodes = BuildNodes(m.Groups)

	line := m.renderNodeLine(m.Nodes[2], 2)
	if strings.Contains(line, "[CLAUDE]") || strings.Contains(line, "[CODEX]") || strings.Contains(line, "[OPEN]") {
		t.Fatalf("window line should not contain agent tag: %q", line)
	}
	if !strings.Contains(line, "·") {
		t.Fatalf("window line missing status badge for done: %q", line)
	}
}

func TestViewAgentsModeEmptyState(t *testing.T) {
	m := Model{
		Mode:           DashboardModeAgents,
		AgentRows:      []AgentWindowRow{},
		Styles:         NewStyles(KanagawaClaw),
		WindowStatuses: make(map[string]tmux.Status),
		Width:          80,
		Height:         24,
	}
	m.Nodes = BuildAgentNodes(m.AgentRows)

	view := m.View()
	if !strings.Contains(view, "No detected agent windows") {
		t.Fatalf("agents empty view missing message: %q", view)
	}
}

func TestRenderNodeLineAgentRowIncludesMetadata(t *testing.T) {
	m := Model{
		Mode: DashboardModeAgents,
		AgentRows: []AgentWindowRow{
			{
				SessionName: "cb_demo",
				WindowName:  "workbench",
				WindowIndex: 4,
				RepoName:    "demo-repo",
				AgentType:   tmux.AgentCodex,
				Status:      tmux.StatusWorking,
			},
		},
		Styles: NewStyles(KanagawaClaw),
		Width:  80,
		Cursor: 0,
	}
	m.Nodes = BuildAgentNodes(m.AgentRows)

	line := m.renderNodeLine(m.Nodes[0], 0)
	if !strings.Contains(line, "[CODEX]") {
		t.Fatalf("agent row missing [CODEX] tag: %q", line)
	}
	if !strings.Contains(line, "•") {
		t.Fatalf("agent row missing status badge: %q", line)
	}
	if !strings.Contains(line, "cb_demo:4") {
		t.Fatalf("agent row missing session target: %q", line)
	}
	if !strings.Contains(line, "repo=demo-repo") {
		t.Fatalf("agent row missing repo metadata: %q", line)
	}
	if strings.Contains(line, "managed") || strings.Contains(line, "unmanaged") {
		t.Fatalf("agent row should not include managed label: %q", line)
	}
}

func TestRenderFooterAgentsMode(t *testing.T) {
	m := Model{
		Mode: DashboardModeAgents,
		AgentRows: []AgentWindowRow{
			{
				SessionName: "cb_demo",
				WindowName:  "claude",
				WindowIndex: 1,
				RepoName:    "repo",
				AgentType:   tmux.AgentClaude,
				Status:      tmux.StatusIdle,
			},
		},
		Styles: NewStyles(KanagawaClaw),
		Width:  80,
		Height: 24,
	}
	m.Nodes = BuildAgentNodes(m.AgentRows)

	footer := m.renderFooter()
	if !strings.Contains(footer, "m mode") {
		t.Fatalf("agents footer missing mode toggle: %q", footer)
	}
	if strings.Contains(footer, "c claude") {
		t.Fatalf("agents footer should not contain create key: %q", footer)
	}
}
