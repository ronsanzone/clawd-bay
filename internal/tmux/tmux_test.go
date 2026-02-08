package tmux

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSessionList(t *testing.T) {
	output := `cb_proj-123-auth: 3 windows (created Tue Feb  4 10:30:00 2025)
cb_proj-456-bug: 1 windows (created Tue Feb  4 11:00:00 2025)
other-session: 2 windows (created Tue Feb  4 09:00:00 2025)`

	sessions := ParseSessionList(output)

	// Should only include cb_ prefixed sessions
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}

	if sessions[0].Name != "cb_proj-123-auth" {
		t.Errorf("first session = %q, want %q", sessions[0].Name, "cb_proj-123-auth")
	}
}

func TestClient_ListSessions_Success(t *testing.T) {
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return []byte(`cb_test-session: 1 windows (created Tue Feb  4 10:30:00 2025)
other-session: 2 windows (created Tue Feb  4 09:00:00 2025)`), nil
		},
	}

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Name != "cb_test-session" {
		t.Errorf("session name = %q, want %q", sessions[0].Name, "cb_test-session")
	}
}

func TestClient_ListSessions_NoTmux(t *testing.T) {
	// Test graceful handling when tmux not running
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return nil, &mockError{msg: "no server running"}
		},
	}

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v, want nil", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestParseWindowList(t *testing.T) {
	// Format from: tmux list-windows -F "#{window_index}:#{window_name}:#{window_active}"
	output := `0:shell:1
1:claude:default:0
2:claude:research:0`

	windows := ParseWindowList(output)

	if len(windows) != 3 {
		t.Fatalf("got %d windows, want 3", len(windows))
	}

	if windows[0].Name != "shell" {
		t.Errorf("window 0 name = %q, want %q", windows[0].Name, "shell")
	}
	if !windows[0].Active {
		t.Error("window 0 should be active")
	}
	if windows[1].Name != "claude:default" {
		t.Errorf("window 1 name = %q, want %q", windows[1].Name, "claude:default")
	}
}

func TestWindow_IsClaudeSession(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"shell", false},
		{"claude:default", true},
		{"claude:research", true},
		{"vim", false},
	}

	for _, tt := range tests {
		w := Window{Name: tt.name}
		if got := w.IsClaudeSession(); got != tt.want {
			t.Errorf("Window{%q}.IsClaudeSession() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestClient_GetPaneStatus(t *testing.T) {
	tests := []struct {
		name        string
		cmdOutput   string
		cmdErr      error
		paneContent string
		expected    Status
	}{
		// WORKING states
		{"claude working with spinner", "claude", nil, "⠹ Thinking...\n", StatusWorking},
		{"claude working with braille", "claude", nil, "⠋ Reading file\n", StatusWorking},
		{"claude working with interrupt", "claude", nil, "Some output\nctrl+c to interrupt\n", StatusWorking},
		{"claude working with esc interrupt", "claude", nil, "Output\nesc to interrupt\n", StatusWorking},
		{"claude working with asterisk", "claude", nil, "✳ Generating...\n", StatusWorking},
		// WAITING states
		{"claude waiting permission", "claude", nil, "Yes, allow once\nNo, deny\n", StatusWaiting},
		{"claude waiting prompt", "claude", nil, "Some output\n> ", StatusWaiting},
		{"claude waiting chevron", "claude", nil, "Ready\n❯ ", StatusWaiting},
		{"claude waiting continue", "claude", nil, "Continue? (Y/n)", StatusWaiting},
		// IDLE state
		{"claude idle no indicators", "claude", nil, "Just some text\nnothing special", StatusIdle},
		// DONE states
		{"shell running", "zsh", nil, "", StatusDone},
		{"bash running", "bash", nil, "", StatusDone},
		{"error", "", errors.New("error"), "", StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				execCommand: func(name string, args ...string) ([]byte, error) {
					// Route based on the tmux subcommand
					if len(args) > 0 && args[0] == "capture-pane" {
						return []byte(tt.paneContent), nil
					}
					return []byte(tt.cmdOutput), tt.cmdErr
				},
			}
			status := client.GetPaneStatus("session", "window")
			if status != tt.expected {
				t.Errorf("GetPaneStatus() = %v, want %v", status, tt.expected)
			}
		})
	}
}

func TestContainsSpinnerChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"braille spinner dot", "⠋ Thinking...", true},
		{"braille spinner bar", "⠹ Reading file.go", true},
		{"multiple braille chars", "⠸⠼⠴", true},
		{"asterisk spinner", "✳ Generating...", true},
		{"star spinner", "✶ Building...", true},
		{"normal text", "Hello world", false},
		{"empty string", "", false},
		{"prompt only", "> ", false},
		{"empty braille U+2800", "\u2800", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsSpinnerChars(tt.input)
			if got != tt.want {
				t.Errorf("containsSpinnerChars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasBusyIndicator(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		// Interrupt messages
		{"ctrl+c interrupt", "Working... ctrl+c to interrupt", true},
		{"ctrl+c case insensitive", "CTRL+C TO INTERRUPT", true},
		{"esc interrupt", "Processing esc to interrupt", true},
		// Spinner characters
		{"braille spinner", "⠹ Thinking...", true},
		{"asterisk spinner", "✳ Generating code...", true},
		{"star spinner", "✶ Building...", true},
		// Negative cases
		{"plain prompt", "> ", false},
		{"just text", "Hello world", false},
		{"empty", "", false},
		{"permission prompt only", "Yes, allow once", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasBusyIndicator(tt.content)
			if got != tt.want {
				t.Errorf("hasBusyIndicator(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestHasPromptIndicator(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		// Permission prompts
		{"allow once", "Yes, allow once\nNo, deny", true},
		{"allow always", "Yes, allow always", true},
		{"tell claude", "No, and tell Claude what to do", true},
		// Confirmation prompts
		{"continue prompt", "Continue? (Y/n)", true},
		{"proceed prompt", "Proceed?", true},
		{"yes no prompt", "[yes/no]", true},
		// Input prompts
		{"arrow prompt", "Some output\n> ", true},
		{"chevron prompt", "Ready\n❯ ", true},
		{"prompt with text before", "What next?\n> ", true},
		// Negative cases
		{"prompt mid-line", "prefix > suffix\nmore text", false},
		{"working output", "ctrl+c to interrupt", false},
		{"plain text", "Just some text", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPromptIndicator(tt.content)
			if got != tt.want {
				t.Errorf("hasPromptIndicator(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestDetectionPriority(t *testing.T) {
	// Verify busy takes precedence over prompt
	tests := []struct {
		name    string
		content string
		busy    bool
		prompt  bool
	}{
		{"busy wins over prompt", "ctrl+c to interrupt\n> ", true, true},
		{"prompt alone", "Yes, allow once\n> ", false, true},
		{"neither", "Just output text", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBusy := hasBusyIndicator(tt.content)
			gotPrompt := hasPromptIndicator(tt.content)
			if gotBusy != tt.busy {
				t.Errorf("hasBusyIndicator() = %v, want %v", gotBusy, tt.busy)
			}
			if gotPrompt != tt.prompt {
				t.Errorf("hasPromptIndicator() = %v, want %v", gotPrompt, tt.prompt)
			}
		})
	}
}

func TestClient_CreateSession(t *testing.T) {
	var capturedArgs []string
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			capturedArgs = args
			return nil, nil
		},
	}

	err := client.CreateSession("cb_proj-123-test", "/path/to/worktree")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Should create detached session with working dir
	expectedArgs := []string{"new-session", "-d", "-s", "cb_proj-123-test", "-c", "/path/to/worktree"}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, expectedArgs)
	}
	for i, arg := range expectedArgs {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, capturedArgs[i], arg)
		}
	}
}

func TestClient_CreateSession_Error(t *testing.T) {
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("tmux error")
		},
	}

	err := client.CreateSession("test", "/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("error = %q, want to contain 'failed to create session'", err)
	}
}

func TestClient_CreateWindow(t *testing.T) {
	var capturedArgs []string
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			capturedArgs = args
			return nil, nil
		},
	}

	err := client.CreateWindow("cb_test", "claude:default", "claude")
	if err != nil {
		t.Fatalf("CreateWindow() error = %v", err)
	}

	expected := []string{"new-window", "-t", "cb_test", "-n", "claude:default", "claude"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("args = %v, want %v", capturedArgs, expected)
	}
	for i, arg := range expected {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, capturedArgs[i], arg)
		}
	}
}

func TestClient_CreateWindow_Error(t *testing.T) {
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("tmux error")
		},
	}

	err := client.CreateWindow("cb_test", "window", "cmd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create window") {
		t.Errorf("error = %q, want to contain 'failed to create window'", err)
	}
}

func TestClient_CreateWindowWithShell(t *testing.T) {
	var calls [][]string
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			return nil, nil
		},
	}

	err := client.CreateWindowWithShell("cb_test", "claude", "claude")
	if err != nil {
		t.Fatalf("CreateWindowWithShell() error = %v", err)
	}

	// Should make two calls: new-window (no command), then send-keys
	if len(calls) != 2 {
		t.Fatalf("got %d tmux calls, want 2", len(calls))
	}

	// First call: create window without command
	newWindowArgs := calls[0]
	expectedNewWindow := []string{"tmux", "new-window", "-t", "cb_test", "-n", "claude"}
	if len(newWindowArgs) != len(expectedNewWindow) {
		t.Fatalf("new-window args = %v, want %v", newWindowArgs, expectedNewWindow)
	}
	for i, arg := range expectedNewWindow {
		if newWindowArgs[i] != arg {
			t.Errorf("new-window arg[%d] = %q, want %q", i, newWindowArgs[i], arg)
		}
	}

	// Second call: send-keys with command
	sendKeysArgs := calls[1]
	expectedSendKeys := []string{"tmux", "send-keys", "-t", "cb_test:claude", "claude", "Enter"}
	if len(sendKeysArgs) != len(expectedSendKeys) {
		t.Fatalf("send-keys args = %v, want %v", sendKeysArgs, expectedSendKeys)
	}
	for i, arg := range expectedSendKeys {
		if sendKeysArgs[i] != arg {
			t.Errorf("send-keys arg[%d] = %q, want %q", i, sendKeysArgs[i], arg)
		}
	}
}

func TestClient_AttachSession_Error(t *testing.T) {
	client := &Client{
		execInteractive: func(name string, args ...string) error {
			return errors.New("tmux error")
		},
	}

	err := client.AttachSession("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to attach to session") {
		t.Errorf("error = %q, want to contain 'failed to attach to session'", err)
	}
}

func TestClient_SwitchClient_Error(t *testing.T) {
	client := &Client{
		execInteractive: func(name string, args ...string) error {
			return errors.New("tmux error")
		},
	}

	err := client.SwitchClient("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to switch to session") {
		t.Errorf("error = %q, want to contain 'failed to switch to session'", err)
	}
}

func TestClient_GetPaneWorkingDir(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		err      error
		expected string
	}{
		{
			name:     "valid path",
			output:   "/Users/ron/code/project/.worktrees/project-feat-auth\n",
			err:      nil,
			expected: "/Users/ron/code/project/.worktrees/project-feat-auth",
		},
		{
			name:     "error returns empty",
			output:   "",
			err:      errors.New("no pane"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				execCommand: func(name string, args ...string) ([]byte, error) {
					return []byte(tt.output), tt.err
				},
			}
			result := client.GetPaneWorkingDir(tt.name)
			if result != tt.expected {
				t.Errorf("GetPaneWorkingDir() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestClient_GetRepoName(t *testing.T) {
	tests := []struct {
		name     string
		outputs  map[string]string
		expected string
	}{
		{
			name: "derives repo from pane path",
			outputs: map[string]string{
				"tmux": "/Users/ron/code/my-project/.worktrees/my-project-feat",
				"git":  "/Users/ron/code/my-project\n",
			},
			expected: "my-project",
		},
		{
			name:     "tmux error returns unknown",
			outputs:  map[string]string{},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "tmux" {
						if out, ok := tt.outputs["tmux"]; ok {
							return []byte(out), nil
						}
						return nil, errors.New("tmux error")
					}
					if name == "git" {
						if out, ok := tt.outputs["git"]; ok {
							return []byte(out), nil
						}
						return nil, errors.New("git error")
					}
					return nil, errors.New("unknown command")
				},
			}
			result := client.GetRepoName("cb_test")
			if result != tt.expected {
				t.Errorf("GetRepoName() = %q, want %q", result, tt.expected)
			}
		})
	}
}
