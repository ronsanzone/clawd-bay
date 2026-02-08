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
