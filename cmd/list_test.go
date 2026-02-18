package cmd

import (
	"strings"
	"testing"

	"github.com/ronsanzone/clawd-bay/internal/discovery"
	"github.com/ronsanzone/clawd-bay/internal/tmux"
)

type fakeListAgentDetector struct {
	infoByWindow map[string]tmux.AgentInfo
}

func (f fakeListAgentDetector) DetectAgentInfo(session, window string) tmux.AgentInfo {
	if info, ok := f.infoByWindow[session+":"+window]; ok {
		return info
	}
	return tmux.AgentInfo{Type: tmux.AgentNone, Detected: false, Status: tmux.StatusDone}
}

func TestSessionStatusFromWindows_IgnoresNonAgents(t *testing.T) {
	detector := fakeListAgentDetector{
		infoByWindow: map[string]tmux.AgentInfo{
			"cb_demo:shell":      {Type: tmux.AgentNone, Detected: false, Status: tmux.StatusDone},
			"cb_demo:notes":      {Type: tmux.AgentNone, Detected: false, Status: tmux.StatusDone},
			"cb_demo:random-win": {Type: tmux.AgentCodex, Detected: true, Status: tmux.StatusWorking},
		},
	}

	wins := []tmux.Window{
		{Name: "shell"},
		{Name: "notes"},
		{Name: "random-win"},
	}

	got := sessionStatusFromWindows(detector, "cb_demo", wins)
	if got != tmux.StatusWorking {
		t.Fatalf("sessionStatusFromWindows() = %q, want %q", got, tmux.StatusWorking)
	}
}

func TestSessionStatusFromWindows_MixedDetectedAgents(t *testing.T) {
	detector := fakeListAgentDetector{
		infoByWindow: map[string]tmux.AgentInfo{
			"cb_demo:codex-main": {Type: tmux.AgentCodex, Detected: true, Status: tmux.StatusWaiting},
			"cb_demo:open-run":   {Type: tmux.AgentOpenCode, Detected: true, Status: tmux.StatusIdle},
		},
	}

	wins := []tmux.Window{
		{Name: "codex-main"},
		{Name: "open-run"},
	}

	got := sessionStatusFromWindows(detector, "cb_demo", wins)
	if got != tmux.StatusWaiting {
		t.Fatalf("sessionStatusFromWindows() = %q, want %q", got, tmux.StatusWaiting)
	}
}

func TestSessionStatusFromWindows_NoDetectedAgents(t *testing.T) {
	detector := fakeListAgentDetector{
		infoByWindow: map[string]tmux.AgentInfo{
			"cb_demo:shell": {Type: tmux.AgentNone, Detected: false, Status: tmux.StatusDone},
		},
	}

	wins := []tmux.Window{
		{Name: "shell"},
	}

	got := sessionStatusFromWindows(detector, "cb_demo", wins)
	if got != tmux.StatusDone {
		t.Fatalf("sessionStatusFromWindows() = %q, want %q", got, tmux.StatusDone)
	}
}

func TestFormatListSessionLine(t *testing.T) {
	t.Run("formats status and plural windows", func(t *testing.T) {
		line := formatListSessionLine(discovery.SessionNode{
			Name:    "cb_demo",
			Status:  tmux.StatusWorking,
			Windows: []tmux.Window{{Name: "a"}, {Name: "b"}},
		})
		if !strings.Contains(line, "(WORKING)") {
			t.Fatalf("line = %q, want status", line)
		}
		if !strings.Contains(line, "2 windows") {
			t.Fatalf("line = %q, want plural windows", line)
		}
	})

	t.Run("singular window wording", func(t *testing.T) {
		line := formatListSessionLine(discovery.SessionNode{
			Name:    "cb_demo",
			Status:  tmux.StatusIdle,
			Windows: []tmux.Window{{Name: "a"}},
		})
		if !strings.Contains(line, "1 window") {
			t.Fatalf("line = %q, want singular window", line)
		}
	})
}
