package tmux

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Session represents a tmux session.
type Session struct {
	Name string
}

// Window represents a tmux window with its index, name, and active state.
type Window struct {
	Index  int
	Name   string
	Active bool
}

// Status represents a Claude session's current state.
type Status string

const (
	// StatusWorking indicates Claude is actively processing a task.
	StatusWorking Status = "WORKING"
	// StatusWaiting indicates Claude needs user input (permission prompt, etc).
	StatusWaiting Status = "WAITING"
	// StatusIdle indicates Claude is running but not actively working.
	StatusIdle Status = "IDLE"
	// StatusDone indicates Claude has exited or the session is complete.
	StatusDone Status = "DONE"
)

// Client provides tmux operations.
type Client struct {
	execCommand     func(name string, args ...string) ([]byte, error)
	execInteractive func(name string, args ...string) error
}

// NewClient creates a Client that executes real tmux commands.
func NewClient() *Client {
	return &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
		execInteractive: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			// Interactive commands need terminal access, not output capture
			return cmd.Run()
		},
	}
}

// ListSessions returns all ClawdBay tmux sessions.
func (c *Client) ListSessions() ([]Session, error) {
	output, err := c.execCommand("tmux", "list-sessions")
	if err != nil {
		// tmux not running or no sessions is expected, return empty list
		errMsg := err.Error()
		if strings.Contains(errMsg, "no server running") ||
			strings.Contains(errMsg, "no sessions") {
			return []Session{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}
	return ParseSessionList(string(output)), nil
}

// ListWindows returns all windows in the given session.
func (c *Client) ListWindows(session string) ([]Window, error) {
	output, err := c.execCommand("tmux", "list-windows", "-t", session, "-F", "#{window_index}:#{window_name}:#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("failed to list windows for %s: %w", session, err)
	}
	return ParseWindowList(string(output)), nil
}

// ParseSessionList parses tmux list-sessions output and returns only cb_ prefixed sessions.
func ParseSessionList(output string) []Session {
	var sessions []Session
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		// Only include cb_ prefixed sessions
		if !strings.HasPrefix(line, "cb_") {
			continue
		}

		// Parse: "cb_proj-123-auth: 3 windows (created ...)"
		// Session name is everything before the colon-space pattern " N windows"
		colonSpace := strings.Index(line, ": ")
		if colonSpace == -1 {
			continue
		}
		name := line[:colonSpace]

		sessions = append(sessions, Session{
			Name: name,
		})
	}

	return sessions
}

// IsClaudeSession returns true if this window is a Claude session.
func (w *Window) IsClaudeSession() bool {
	return strings.HasPrefix(w.Name, "claude:")
}

// ParseWindowList parses output from:
// tmux list-windows -F "#{window_index}:#{window_name}:#{window_active}"
// Format: "0:shell:1" or "1:claude:default:0"
func ParseWindowList(output string) []Window {
	var windows []Window
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Split from the end to handle window names with colons (like "claude:default")
		// Format: index:name:active where active is 0 or 1
		lastColon := strings.LastIndex(line, ":")
		if lastColon == -1 {
			continue
		}

		activeStr := line[lastColon+1:]
		rest := line[:lastColon]

		firstColon := strings.Index(rest, ":")
		if firstColon == -1 {
			continue
		}

		idxStr := rest[:firstColon]
		name := rest[firstColon+1:]

		idx := 0
		_, _ = fmt.Sscanf(idxStr, "%d", &idx)

		windows = append(windows, Window{
			Index:  idx,
			Name:   name,
			Active: activeStr == "1",
		})
	}

	return windows
}

// GetPaneStatus detects if a Claude session is IDLE, WORKING, or DONE
// by checking the pane's current command and inspecting pane content.
func (c *Client) GetPaneStatus(session, window string) Status {
	target := session + ":" + window
	output, err := c.execCommand("tmux", "display-message", "-t", target, "-p", "#{pane_current_command}")
	if err != nil {
		return StatusDone
	}

	cmd := strings.TrimSpace(string(output))

	// If the pane is running a shell, Claude has exited
	if cmd == "zsh" || cmd == "bash" || cmd == "sh" {
		return StatusDone
	}

	// If claude is running, inspect pane content to distinguish IDLE vs WORKING
	if cmd == "claude" || strings.Contains(cmd, "claude") {
		return c.detectClaudeActivity(target)
	}

	return StatusDone
}

// detectClaudeActivity inspects the last few lines of a pane to determine
// Claude Code's current state: actively working, waiting for input, or idle.
//
// Detection priority (matches agent-deck approach):
//  1. Busy indicators (spinners, interrupt messages) → WORKING
//  2. Prompt indicators (permission dialogs, input prompts) → WAITING
//  3. Default → IDLE
func (c *Client) detectClaudeActivity(target string) Status {
	output, err := c.execCommand("tmux", "capture-pane", "-t", target, "-p", "-l", "10")
	if err != nil {
		return StatusIdle
	}

	content := string(output)

	// Priority 1: Check busy indicators
	if hasBusyIndicator(content) {
		return StatusWorking
	}

	// Priority 2: Check prompt indicators
	if hasPromptIndicator(content) {
		return StatusWaiting
	}

	return StatusIdle
}

// busyStrings are text patterns that indicate Claude is actively working.
var busyStrings = []string{
	"ctrl+c to interrupt",
	"esc to interrupt",
}

// spinnerChars includes both Braille and asterisk spinner characters.
var spinnerChars = []rune{
	// Braille spinners
	'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏',
	// Asterisk spinners (Claude 2.1.25+)
	'✳', '✽', '✶', '✢',
}

// hasBusyIndicator reports whether content contains indicators that Claude
// is actively working: interrupt messages or spinner characters.
func hasBusyIndicator(content string) bool {
	lower := strings.ToLower(content)

	// Check interrupt messages
	for _, s := range busyStrings {
		if strings.Contains(lower, s) {
			return true
		}
	}

	// Check spinner characters
	return containsSpinnerChars(content)
}

// containsSpinnerChars checks for any spinner character in the content.
func containsSpinnerChars(s string) bool {
	for _, r := range s {
		for _, spinner := range spinnerChars {
			if r == spinner {
				return true
			}
		}
		// Also check Braille range for backwards compatibility
		if r > 0x2800 && r <= 0x28FF {
			return true
		}
	}
	return false
}

// promptStrings are permission dialog patterns.
var promptStrings = []string{
	"yes, allow once",
	"yes, allow always",
	"no, and tell claude",
}

// confirmationPatterns are patterns for confirmation prompts.
var confirmationPatterns = []string{
	"continue?",
	"proceed?",
	"(y/n)",
	"[yes/no]",
}

// hasPromptIndicator reports whether content contains indicators that Claude
// is waiting for user input: permission dialogs or input prompts.
func hasPromptIndicator(content string) bool {
	lower := strings.ToLower(content)

	// Check permission prompts
	for _, s := range promptStrings {
		if strings.Contains(lower, s) {
			return true
		}
	}

	// Check confirmation prompts
	for _, p := range confirmationPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	// Check for input prompt (last non-empty line ends with > or ❯)
	lines := strings.Split(content, "\n")
	lastLine := getLastNonEmptyLine(lines)
	trimmed := strings.TrimSpace(lastLine)
	if strings.HasSuffix(trimmed, ">") || strings.HasSuffix(trimmed, "❯") {
		return true
	}

	return false
}

// getLastNonEmptyLine returns the last line that contains non-whitespace.
func getLastNonEmptyLine(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}

// CreateSession creates a new detached tmux session with the given name and working directory.
func (c *Client) CreateSession(name, workdir string) error {
	_, err := c.execCommand("tmux", "new-session", "-d", "-s", name, "-c", workdir)
	if err != nil {
		return fmt.Errorf("failed to create session %s: %w", name, err)
	}
	return nil
}

// CreateWindow creates a new window in the given session.
// If command is non-empty, it is run directly as the window's process.
// Note: commands run this way use a non-login shell, so profile env vars
// may not be available. Use CreateWindowWithShell for commands that need
// the user's full environment.
func (c *Client) CreateWindow(session, name string, command string) error {
	args := []string{"new-window", "-t", session, "-n", name}
	if command != "" {
		args = append(args, command)
	}
	_, err := c.execCommand("tmux", args...)
	if err != nil {
		return fmt.Errorf("failed to create window %s in %s: %w", name, session, err)
	}
	return nil
}

// CreateWindowWithShell creates a new window with an interactive login shell,
// then sends the command via send-keys. This ensures the user's profile files
// (.zshrc, .zprofile, .bashrc) are sourced and env vars are available.
func (c *Client) CreateWindowWithShell(session, name, command string) error {
	// Create window with default shell (interactive login shell)
	_, err := c.execCommand("tmux", "new-window", "-t", session, "-n", name)
	if err != nil {
		return fmt.Errorf("failed to create window %s in %s: %w", name, session, err)
	}

	// Send the command to the new window's shell
	if command != "" {
		target := session + ":" + name
		_, err = c.execCommand("tmux", "send-keys", "-t", target, command, "Enter")
		if err != nil {
			return fmt.Errorf("failed to send command to %s:%s: %w", session, name, err)
		}
	}
	return nil
}

// AttachSession attaches to the given tmux session.
// This is an interactive command that takes over the terminal.
func (c *Client) AttachSession(name string) error {
	if err := c.execInteractive("tmux", "attach-session", "-t", name); err != nil {
		return fmt.Errorf("failed to attach to session %s: %w", name, err)
	}
	return nil
}

// SwitchClient switches the tmux client to the given session.
// This is an interactive command that manipulates the terminal.
func (c *Client) SwitchClient(name string) error {
	if err := c.execInteractive("tmux", "switch-client", "-t", name); err != nil {
		return fmt.Errorf("failed to switch to session %s: %w", name, err)
	}
	return nil
}

// GetPaneWorkingDir returns the working directory of the first pane in a session.
// Returns empty string on error.
func (c *Client) GetPaneWorkingDir(session string) string {
	target := session + ":0"
	output, err := c.execCommand("tmux", "display-message", "-t", target, "-p", "#{pane_current_path}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetRepoName returns the repository name for a session by querying the
// pane's working directory and deriving the git toplevel.
// Returns "Unknown" if the repo cannot be determined.
func (c *Client) GetRepoName(session string) string {
	paneDir := c.GetPaneWorkingDir(session)
	if paneDir == "" {
		return "Unknown"
	}

	output, err := c.execCommand("git", "-C", paneDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "Unknown"
	}

	repoRoot := strings.TrimSpace(string(output))
	return filepath.Base(repoRoot)
}
