package cmd

import (
	"errors"
	"testing"

	"github.com/rsanzone/clawdbay/internal/tui"
)

type fakeDashTmuxClient struct {
	calls               []string
	selectedSession     string
	selectedWindowIndex int
	attachedSession     string
	inTmux              bool
	selectErr           error
	attachErr           error
}

func (f *fakeDashTmuxClient) SelectWindow(session string, windowIndex int) error {
	f.calls = append(f.calls, "select")
	f.selectedSession = session
	f.selectedWindowIndex = windowIndex
	return f.selectErr
}

func (f *fakeDashTmuxClient) AttachOrSwitchToSession(name string, inTmux bool) error {
	f.calls = append(f.calls, "attach")
	f.attachedSession = name
	f.inTmux = inTmux
	return f.attachErr
}

func TestAttachDashboardSelection_SessionOnly(t *testing.T) {
	client := &fakeDashTmuxClient{}
	model := tui.Model{
		SelectedName:        "cb_demo",
		SelectedWindowIndex: -1,
	}

	err := attachDashboardSelection(client, model, true)
	if err != nil {
		t.Fatalf("attachDashboardSelection() error = %v", err)
	}
	if len(client.calls) != 1 || client.calls[0] != "attach" {
		t.Fatalf("calls = %v, want [attach]", client.calls)
	}
	if client.attachedSession != "cb_demo" {
		t.Fatalf("attachedSession = %q, want %q", client.attachedSession, "cb_demo")
	}
	if !client.inTmux {
		t.Fatal("inTmux should be true")
	}
}

func TestAttachDashboardSelection_WindowSelectionOrder(t *testing.T) {
	client := &fakeDashTmuxClient{}
	model := tui.Model{
		SelectedName:        "cb_demo",
		SelectedWindowIndex: 2,
	}

	err := attachDashboardSelection(client, model, false)
	if err != nil {
		t.Fatalf("attachDashboardSelection() error = %v", err)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %v, want 2 calls", client.calls)
	}
	if client.calls[0] != "select" || client.calls[1] != "attach" {
		t.Fatalf("calls = %v, want [select attach]", client.calls)
	}
	if client.selectedSession != "cb_demo" || client.selectedWindowIndex != 2 {
		t.Fatalf("select args = (%q, %d), want (%q, %d)", client.selectedSession, client.selectedWindowIndex, "cb_demo", 2)
	}
	if client.attachedSession != "cb_demo" {
		t.Fatalf("attachedSession = %q, want %q", client.attachedSession, "cb_demo")
	}
	if client.inTmux {
		t.Fatal("inTmux should be false")
	}
}

func TestAttachDashboardSelection_SelectError(t *testing.T) {
	client := &fakeDashTmuxClient{selectErr: errors.New("select failed")}
	model := tui.Model{
		SelectedName:        "cb_demo",
		SelectedWindowIndex: 1,
	}

	err := attachDashboardSelection(client, model, false)
	if err == nil {
		t.Fatal("attachDashboardSelection() expected error, got nil")
	}
	if len(client.calls) != 1 || client.calls[0] != "select" {
		t.Fatalf("calls = %v, want only select", client.calls)
	}
}

func TestAttachDashboardSelection_AttachError(t *testing.T) {
	client := &fakeDashTmuxClient{attachErr: errors.New("attach failed")}
	model := tui.Model{
		SelectedName:        "cb_demo",
		SelectedWindowIndex: -1,
	}

	err := attachDashboardSelection(client, model, false)
	if err == nil {
		t.Fatal("attachDashboardSelection() expected error, got nil")
	}
	if len(client.calls) != 1 || client.calls[0] != "attach" {
		t.Fatalf("calls = %v, want only attach", client.calls)
	}
}
