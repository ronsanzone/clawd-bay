# ClawdBay Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

## Progress

- [x] **Task 0:** Verify Environment and Dependencies - `06c7fcb`
- [x] **Task 1:** Initialize Go Module - `bdd1252`
- [x] **Task 2:** Add Cobra CLI Framework - `2abba97`
- [x] **Task 3:** Add Version Command - `70f9b20`
- [x] **Task 4:** Create Config Package - `7969956`
- [x] **Task 5:** Add Config Initialization - `d20021a`
- [x] **Task 6:** Create tmux Package - Session Detection - `f9748a3`
- [x] **Task 7:** Add tmux Command Execution - `057778e`
- [x] **Task 8:** Add Window Listing and Status Detection - `20e5e67`
- [x] **Task 9:** Add Session Creation - `b084089`
- [x] **Task 10:** Add `cb start` Command - `794b7c6`
- [x] **Task 11:** Add `cb claude` Command - `589f143`
- [x] **Task 12:** Add `cb prompt` Commands - `2439d29`
- [x] **Task 13:** Add `cb list` Command - `c06b76c`
- [x] **Task 14:** Add `cb archive` Command - `d08d469`
- [x] **Task 15:** Add Bubbletea Dependencies - `49cf7c3`
- [x] **Task 16:** Create Dashboard Model - `a867b7e`
- [x] **Task 17:** Implement Bubbletea Update/View - `c4ab16d`
- [x] **Task 18:** Add `cb dash` Command - `7cc85f6`
- [x] **Task 19:** Add Integration Test - `4ec6fce`
- [x] **Task 20:** Create Default Prompt Templates - `fea6a45`
- [x] **Task 21:** Add Install Command for Templates - `72641a3`
- [x] **Task 22:** Final Build and Test - `cd6591f`

---

**Goal:** Build `cb`, a CLI + TUI tool for managing multi-session Claude Code workflows from idea to completion.

**Architecture:** Go CLI using Cobra for commands, Bubbletea for TUI dashboard. Integrates with tmux for session management and manages prompt templates. State derived from tmux (no separate state files).

**Tech Stack:** Go 1.21+, Cobra (CLI), Bubbletea (TUI), Lipgloss (styling), tmux (external)

**Design Doc:** `docs/plans/2025-02-04-clawdbay-design.md`

---

## Phase 0: Prerequisites Verification

### Task 0: Verify Environment and Dependencies

**Purpose:** Ensure external dependencies work as expected before building.

**Step 1: Verify tmux is available**

```bash
tmux -V
```
Expected: `tmux 3.x` or similar

**Step 2: Create verification commands**

Create `Makefile`:

```makefile
.PHONY: build test lint verify install clean

build:
	go build -o cb main.go

test:
	go test ./...

lint:
	golangci-lint run

verify: test lint
	@echo "All checks passed"

install: build
	cp cb ~/bin/cb

clean:
	rm -f cb
```

**Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile with build/test/lint targets"
```

---

## Phase 1: Project Foundation

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `main.go`

**Step 1: Initialize Go module**

```bash
cd /Users/ron.sanzone/code/claude-essentials/.worktrees/clawdbay
go mod init github.com/rsanzone/clawdbay
```

**Step 2: Create minimal main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("ClawdBay - A harbor for your Claude sessions")
}
```

**Step 3: Verify it runs**

Run: `go run main.go`
Expected: `ClawdBay - A harbor for your Claude sessions`

**Step 4: Commit**

```bash
git add go.mod main.go
git commit -m "feat: initialize ClawdBay Go project"
```

---

### Task 2: Add Cobra CLI Framework

**Files:**
- Modify: `main.go`
- Create: `cmd/root.go`
- Modify: `go.mod` (auto via go get)

**Step 1: Add Cobra dependency**

```bash
go get github.com/spf13/cobra@latest
```

**Step 2: Create root command**

Create `cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cb",
	Short: "ClawdBay - A harbor for your Claude sessions",
	Long: `ClawdBay manages multi-session Claude Code workflows.

Start workflows with git worktrees, manage multiple Claude sessions
per worktree, and track session status from an interactive dashboard.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to dashboard when no subcommand
		fmt.Println("ClawdBay dashboard would open here")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Update main.go**

```go
package main

import "github.com/rsanzone/clawdbay/cmd"

func main() {
	cmd.Execute()
}
```

**Step 4: Verify CLI works**

Run: `go run main.go --help`
Expected: Shows usage with "ClawdBay - A harbor for your Claude sessions"

**Step 5: Commit**

```bash
git add go.mod go.sum main.go cmd/
git commit -m "feat: add Cobra CLI framework with root command"
```

---

### Task 3: Add Version Command

**Files:**
- Create: `cmd/version.go`
- Modify: `cmd/root.go`

**Step 1: Create version command**

Create `cmd/version.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print ClawdBay version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ClawdBay v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

**Step 2: Verify version command**

Run: `go run main.go version`
Expected: `ClawdBay v0.1.0`

**Step 3: Commit**

```bash
git add cmd/version.go
git commit -m "feat: add version command"
```

**Phase 1 Checkpoint:**
```bash
make verify  # Must pass before continuing
```

---

## Phase 2: Configuration Management

### Task 4: Create Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing test for config paths**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "cb")

	cfg := New()
	if cfg.ConfigDir != expected {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, expected)
	}
}

func TestPromptsDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "cb", "prompts")

	cfg := New()
	if cfg.PromptsDir != expected {
		t.Errorf("PromptsDir = %q, want %q", cfg.PromptsDir, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ConfigDir  string
	PromptsDir string
}

func New() *Config {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "cb")

	return &Config{
		ConfigDir:  configDir,
		PromptsDir: filepath.Join(configDir, "prompts"),
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/
git commit -m "feat: add config package with default paths"
```

---

### Task 5: Add Config Initialization

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Step 1: Write failing test for EnsureDirs**

Add to `internal/config/config_test.go`:

```go
func TestEnsureDirs(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()

	cfg := &Config{
		ConfigDir:  filepath.Join(tmpDir, ".config", "cb"),
		PromptsDir: filepath.Join(tmpDir, ".config", "cb", "prompts"),
	}

	err := cfg.EnsureDirs()
	if err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	// Check directories exist
	if _, err := os.Stat(cfg.ConfigDir); os.IsNotExist(err) {
		t.Error("ConfigDir was not created")
	}
	if _, err := os.Stat(cfg.PromptsDir); os.IsNotExist(err) {
		t.Error("PromptsDir was not created")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL - EnsureDirs not defined

**Step 3: Write minimal implementation**

Add to `internal/config/config.go`:

```go
func (c *Config) EnsureDirs() error {
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(c.PromptsDir, 0755); err != nil {
		return err
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add EnsureDirs to create config directories"
```

**Phase 2 Checkpoint:**
```bash
make verify  # Must pass before continuing
```

---

## Phase 3: tmux Integration

### Task 6: Create tmux Package - Session Detection

**Files:**
- Create: `internal/tmux/tmux.go`
- Create: `internal/tmux/tmux_test.go`

**Step 1: Write test for parsing session list**

Create `internal/tmux/tmux_test.go`:

```go
package tmux

import (
	"testing"
)

func TestParseSessionList(t *testing.T) {
	output := `cb:proj-123-auth: 3 windows (created Tue Feb  4 10:30:00 2025)
cb:proj-456-bug: 1 windows (created Tue Feb  4 11:00:00 2025)
other-session: 2 windows (created Tue Feb  4 09:00:00 2025)`

	sessions := ParseSessionList(output)

	// Should only include cb: prefixed sessions
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}

	if sessions[0].Name != "cb:proj-123-auth" {
		t.Errorf("first session = %q, want %q", sessions[0].Name, "cb:proj-123-auth")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/...`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

Create `internal/tmux/tmux.go`:

```go
package tmux

import (
	"strings"
)

type Session struct {
	Name        string
	WindowCount int
}

func ParseSessionList(output string) []Session {
	var sessions []Session
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		// Only include cb: prefixed sessions
		if !strings.HasPrefix(line, "cb:") {
			continue
		}

		// Parse: "cb:proj-123-auth: 3 windows (created ...)"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		name := "cb:" + strings.TrimSpace(strings.Split(parts[1], ":")[0])

		sessions = append(sessions, Session{
			Name: name,
		})
	}

	return sessions
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add tmux package with session parsing"
```

---

### Task 7: Add tmux Command Execution

**Files:**
- Modify: `internal/tmux/tmux.go`
- Modify: `internal/tmux/tmux_test.go`

**Step 1: Write test for ListSessions (integration-style, will mock)**

Add to `internal/tmux/tmux_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/...`
Expected: FAIL - Client not defined

**Step 3: Write implementation**

Add to `internal/tmux/tmux.go`:

```go
import (
	"os/exec"
	"strings"
)

type Client struct {
	execCommand func(name string, args ...string) ([]byte, error)
}

func NewClient() *Client {
	return &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
	}
}

func (c *Client) ListSessions() ([]Session, error) {
	output, err := c.execCommand("tmux", "list-sessions")
	if err != nil {
		// tmux not running is not an error, just no sessions
		return []Session{}, nil
	}
	return ParseSessionList(string(output)), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add tmux client with ListSessions"
```

---

### Task 8: Add Window Listing and Status Detection

**Files:**
- Modify: `internal/tmux/tmux.go`
- Modify: `internal/tmux/tmux_test.go`

**Step 1: Add Status type and detection**

Add to `internal/tmux/tmux.go`:

```go
type Status string

const (
	StatusIdle    Status = "IDLE"
	StatusWorking Status = "WORKING"
	StatusDone    Status = "DONE"
)
```

**Step 2: Write test for window parsing with status**

Add to `internal/tmux/tmux_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/...`
Expected: FAIL - Window, ParseWindowList not defined

**Step 3: Write implementation**

Add to `internal/tmux/tmux.go`:

```go
type Window struct {
	Index  int
	Name   string
	Active bool
}

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
		fmt.Sscanf(idxStr, "%d", &idx)

		windows = append(windows, Window{
			Index:  idx,
			Name:   name,
			Active: activeStr == "1",
		})
	}

	return windows
}
```

Also add `"fmt"` to imports.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/...`
Expected: PASS

**Step 5: Add pane status detection**

Add to `internal/tmux/tmux.go`:

```go
// GetPaneStatus detects if a Claude session is IDLE, WORKING, or DONE
// by checking the pane's current command and recent activity.
func (c *Client) GetPaneStatus(session, window string) Status {
	// Get the current command running in the pane
	target := session + ":" + window
	output, err := c.execCommand("tmux", "display-message", "-t", target, "-p", "#{pane_current_command}")
	if err != nil {
		return StatusDone
	}

	cmd := strings.TrimSpace(string(output))

	// If the pane is running a shell (zsh, bash), Claude has exited
	if cmd == "zsh" || cmd == "bash" || cmd == "sh" {
		return StatusDone
	}

	// Check if there's been recent activity (within last 5 seconds)
	// Using pane_last_activity_time vs current time
	activityOutput, err := c.execCommand("tmux", "display-message", "-t", target, "-p",
		"#{pane_activity}:#{session_activity}")
	if err != nil {
		return StatusIdle
	}

	// For simplicity, assume WORKING if claude is running
	// A more sophisticated check would compare timestamps
	if cmd == "claude" || strings.Contains(cmd, "claude") {
		return StatusIdle // Default to IDLE, TUI will show this
	}

	return StatusDone
}
```

**Step 6: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add window listing and Claude session detection"
```

---

### Task 9: Add Session Creation

**Files:**
- Modify: `internal/tmux/tmux.go`
- Modify: `internal/tmux/tmux_test.go`

**Step 1: Write test for CreateSession**

Add to `internal/tmux/tmux_test.go`:

```go
func TestClient_CreateSession(t *testing.T) {
	var capturedArgs []string
	client := &Client{
		execCommand: func(name string, args ...string) ([]byte, error) {
			capturedArgs = args
			return nil, nil
		},
	}

	err := client.CreateSession("cb:proj-123-test", "/path/to/worktree")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Should create detached session with working dir
	expectedArgs := []string{"new-session", "-d", "-s", "cb:proj-123-test", "-c", "/path/to/worktree"}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, expectedArgs)
	}
	for i, arg := range expectedArgs {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, capturedArgs[i], arg)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/...`
Expected: FAIL - CreateSession not defined

**Step 3: Write implementation**

Add to `internal/tmux/tmux.go`:

```go
func (c *Client) CreateSession(name, workdir string) error {
	_, err := c.execCommand("tmux", "new-session", "-d", "-s", name, "-c", workdir)
	return err
}

func (c *Client) CreateWindow(session, name string, command string) error {
	args := []string{"new-window", "-t", session, "-n", name}
	if command != "" {
		args = append(args, command)
	}
	_, err := c.execCommand("tmux", args...)
	return err
}

func (c *Client) AttachSession(name string) error {
	_, err := c.execCommand("tmux", "attach-session", "-t", name)
	return err
}

func (c *Client) SwitchClient(name string) error {
	_, err := c.execCommand("tmux", "switch-client", "-t", name)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add session and window creation"
```

**Phase 3 Checkpoint:**
```bash
make verify  # Must pass before continuing
```

---

## Phase 4: Core Commands

### Task 10: Add `cb start` Command

**Files:**
- Create: `cmd/start.go`

**Step 1: Create start command**

Create `cmd/start.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <branch-name>",
	Short: "Start a new workflow with a git worktree and tmux session",
	Long: `Creates a git worktree and tmux session for the given branch name.

Example:
  cb start proj-123-auth-feature
  cb start feature/add-login`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	branchName := sanitizeBranchName(args[0])

	// Verify we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		return fmt.Errorf("not in a git repository")
	}

	// Get current directory info
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(cwd)
	worktreeDir := filepath.Join(filepath.Dir(cwd), projectName+"-"+branchName)

	// Check if worktree directory already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	// Check if branch already exists
	checkBranch := exec.Command("git", "rev-parse", "--verify", branchName)
	if checkBranch.Run() == nil {
		// Branch exists, create worktree without -b flag
		fmt.Printf("Branch %s exists, creating worktree...\n", branchName)
		gitCmd := exec.Command("git", "worktree", "add", worktreeDir, branchName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		// Create new branch and worktree
		fmt.Printf("Creating worktree: %s\n", worktreeDir)
		gitCmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", branchName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Create tmux session
	sessionName := "cb:" + branchName
	tmuxClient := tmux.NewClient()

	fmt.Printf("Creating tmux session: %s\n", sessionName)
	if err := tmuxClient.CreateSession(sessionName, worktreeDir); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Switch to the session
	if os.Getenv("TMUX") != "" {
		return tmuxClient.SwitchClient(sessionName)
	}
	return tmuxClient.AttachSession(sessionName)
}

// sanitizeBranchName converts a string to a valid git branch name
func sanitizeBranchName(name string) string {
	// Replace spaces and special chars with dashes
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	// Remove characters not allowed in branch names
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}

	// Clean up multiple dashes
	cleaned := result.String()
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}

	return strings.Trim(cleaned, "-")
}
```

**Step 2: Verify command shows in help**

Run: `go run main.go start --help`
Expected: Shows start command help

**Step 3: Commit**

```bash
git add cmd/start.go
git commit -m "feat: add cb start command"
```

---

### Task 11: Add `cb claude` Command

**Files:**
- Create: `cmd/claude.go`

**Step 1: Create claude command**

Create `cmd/claude.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	claudeName   string
	claudePrompt string
)

var claudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Add a Claude session to current worktree",
	Long: `Creates a new tmux window with a Claude session.

Example:
  cb claude                           # Creates default session
  cb claude --name research           # Named session
  cb claude --name impl --prompt plan.md  # With prompt file`,
	RunE: runClaude,
}

func init() {
	claudeCmd.Flags().StringVarP(&claudeName, "name", "n", "default", "Name for the Claude session")
	claudeCmd.Flags().StringVarP(&claudePrompt, "prompt", "p", "", "Prompt file to execute")
	rootCmd.AddCommand(claudeCmd)
}

func runClaude(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tmuxClient := tmux.NewClient()

	// First, try to get current session from TMUX environment
	var sessionName string
	if tmuxEnv := os.Getenv("TMUX"); tmuxEnv != "" {
		// We're inside tmux, get current session name
		output, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
		if err == nil {
			currentSession := strings.TrimSpace(string(output))
			if strings.HasPrefix(currentSession, "cb:") {
				sessionName = currentSession
			}
		}
	}

	// If not in a cb: session, try to find one matching this directory
	if sessionName == "" {
		sessions, err := tmuxClient.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		// Worktree paths follow: <project>-<ticket>-<title>
		// Session names follow: cb:<ticket>-<title>
		dirName := filepath.Base(cwd)
		for _, s := range sessions {
			sessionSuffix := strings.TrimPrefix(s.Name, "cb:")
			if strings.Contains(dirName, sessionSuffix) {
				sessionName = s.Name
				break
			}
		}
	}

	if sessionName == "" {
		return fmt.Errorf("no cb: session found for this directory. Run 'cb start' first")
	}

	// Create window name
	windowName := "claude:" + claudeName

	// Build claude command
	claudeCommand := "claude"
	if claudePrompt != "" {
		promptPath := filepath.Join(cwd, ".prompts", claudePrompt)
		if _, err := os.Stat(promptPath); err == nil {
			claudeCommand = fmt.Sprintf("claude < %s", promptPath)
		} else {
			return fmt.Errorf("prompt file not found: %s", promptPath)
		}
	}

	// Create window with claude
	fmt.Printf("Creating Claude session: %s in %s\n", windowName, sessionName)
	createCmd := exec.Command("tmux", "new-window", "-t", sessionName, "-n", windowName, claudeCommand)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create Claude window: %w", err)
	}

	// Switch to the new window
	selectCmd := exec.Command("tmux", "select-window", "-t", sessionName+":"+windowName)
	return selectCmd.Run()
}
```

**Step 2: Verify command shows in help**

Run: `go run main.go claude --help`
Expected: Shows claude command help with --name and --prompt flags

**Step 3: Commit**

```bash
git add cmd/claude.go
git commit -m "feat: add cb claude command"
```

---

### Task 12: Add `cb prompt` Commands

**Files:**
- Create: `cmd/prompt.go`
- Create: `internal/prompt/prompt.go`
- Create: `internal/prompt/prompt_test.go`

**Step 1: Create prompt package with tests**

Create `internal/prompt/prompt_test.go`:

```go
package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test templates
	os.WriteFile(filepath.Join(tmpDir, "research.md"), []byte("# Research"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "plan.md"), []byte("# Plan"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "not-md.txt"), []byte("ignored"), 0644)

	templates, err := ListTemplates(tmpDir)
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}

	if len(templates) != 2 {
		t.Errorf("got %d templates, want 2", len(templates))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/prompt/...`
Expected: FAIL - package doesn't exist

**Step 3: Write implementation**

Create `internal/prompt/prompt.go`:

```go
package prompt

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ListTemplates(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var templates []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			templates = append(templates, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return templates, nil
}

func CopyTemplate(srcDir, dstDir, name string) error {
	srcPath := filepath.Join(srcDir, name+".md")
	dstPath := filepath.Join(dstDir, name+".md")

	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("template not found: %s", name)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/prompt/...`
Expected: PASS

**Step 5: Create prompt command**

Create `cmd/prompt.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/rsanzone/clawdbay/internal/prompt"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage prompt templates",
}

var promptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available prompt templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.New()
		templates, err := prompt.ListTemplates(cfg.PromptsDir)
		if err != nil {
			return err
		}

		if len(templates) == 0 {
			fmt.Println("No templates found. Create templates in:", cfg.PromptsDir)
			return nil
		}

		fmt.Println("Available templates:")
		for _, t := range templates {
			fmt.Printf("  - %s\n", t)
		}
		return nil
	},
}

var promptAddCmd = &cobra.Command{
	Use:   "add <template-name>",
	Short: "Copy template to .prompts/ and open in editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName := args[0]
		cfg := config.New()

		cwd, _ := os.Getwd()
		dstDir := filepath.Join(cwd, ".prompts")

		// Copy template
		if err := prompt.CopyTemplate(cfg.PromptsDir, dstDir, templateName); err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, templateName+".md")
		fmt.Printf("Created: %s\n", dstPath)

		// Open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim"
		}

		editorCmd := exec.Command(editor, dstPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		return editorCmd.Run()
	},
}

var promptRunCmd = &cobra.Command{
	Use:   "run <prompt-file>",
	Short: "Execute prompt file with Claude",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptFile := args[0]
		cwd, _ := os.Getwd()
		promptPath := filepath.Join(cwd, ".prompts", promptFile)

		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			return fmt.Errorf("prompt not found: %s", promptPath)
		}

		// Execute: claude < prompt.md
		claudeCmd := exec.Command("sh", "-c", fmt.Sprintf("claude < %s", promptPath))
		claudeCmd.Stdin = os.Stdin
		claudeCmd.Stdout = os.Stdout
		claudeCmd.Stderr = os.Stderr
		return claudeCmd.Run()
	},
}

func init() {
	promptCmd.AddCommand(promptListCmd)
	promptCmd.AddCommand(promptAddCmd)
	promptCmd.AddCommand(promptRunCmd)
	rootCmd.AddCommand(promptCmd)
}
```

**Step 6: Verify commands show in help**

Run: `go run main.go prompt --help`
Expected: Shows prompt subcommands (list, add, run)

**Step 7: Commit**

```bash
git add internal/prompt/ cmd/prompt.go
git commit -m "feat: add cb prompt commands (list, add, run)"
```

---

### Task 13: Add `cb list` Command

**Files:**
- Create: `cmd/list.go`

**Step 1: Create list command**

Create `cmd/list.go`:

```go
package cmd

import (
	"fmt"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active ClawdBay workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxClient := tmux.NewClient()
		sessions, err := tmuxClient.ListSessions()
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No active workflows. Start one with: cb start <ticket-id>")
			return nil
		}

		fmt.Println("Active workflows:")
		for _, s := range sessions {
			fmt.Printf("  %s\n", s.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Verify command**

Run: `go run main.go list`
Expected: Shows "No active workflows" or lists cb: sessions

**Step 3: Commit**

```bash
git add cmd/list.go
git commit -m "feat: add cb list command"
```

---

### Task 14: Add `cb archive` Command

**Files:**
- Create: `cmd/archive.go`

**Step 1: Create archive command**

Create `cmd/archive.go`:

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [session-name]",
	Short: "Archive workflow (kill session + remove worktree, keep branch)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var sessionName string
		var worktreePath string

		if len(args) > 0 {
			sessionName = args[0]
			if !strings.HasPrefix(sessionName, "cb:") {
				sessionName = "cb:" + sessionName
			}
		} else {
			// Detect session from current directory
			// Worktree paths follow pattern: <project>-<ticket>-<title>
			// Session names follow pattern: cb:<ticket>-<title>
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			worktreePath = cwd

			// Try to find matching session by checking all cb: sessions
			tmuxClient := tmux.NewClient()
			sessions, err := tmuxClient.ListSessions()
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			dirName := filepath.Base(cwd)
			for _, s := range sessions {
				// Session name is cb:<ticket>-<title>
				// Dir name is <project>-<ticket>-<title>
				// Match by checking if dir contains the session suffix
				sessionSuffix := strings.TrimPrefix(s.Name, "cb:")
				if strings.Contains(dirName, sessionSuffix) {
					sessionName = s.Name
					break
				}
			}

			if sessionName == "" {
				return fmt.Errorf("no cb: session found for directory %s", dirName)
			}
		}

		// Confirm
		fmt.Printf("Archive workflow: %s\n", sessionName)
		if worktreePath != "" {
			fmt.Printf("Worktree: %s\n", worktreePath)
		}
		fmt.Print("This will kill the tmux session and remove the worktree. Continue? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}

		// Kill tmux session
		fmt.Println("Killing tmux session...")
		killCmd := exec.Command("tmux", "kill-session", "-t", sessionName)
		killCmd.Run() // Ignore error if session doesn't exist

		// Remove worktree if we detected it
		if worktreePath != "" {
			fmt.Printf("Removing worktree: %s\n", worktreePath)

			// Change to parent before removing
			parentDir := filepath.Dir(worktreePath)
			if err := os.Chdir(parentDir); err != nil {
				return fmt.Errorf("failed to change to parent directory: %w", err)
			}

			removeCmd := exec.Command("git", "worktree", "remove", worktreePath)
			removeCmd.Stdout = os.Stdout
			removeCmd.Stderr = os.Stderr
			if err := removeCmd.Run(); err != nil {
				return fmt.Errorf("failed to remove worktree: %w", err)
			}
		}

		fmt.Println("Workflow archived. Branch preserved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}
```

**Step 2: Verify command shows in help**

Run: `go run main.go archive --help`
Expected: Shows archive command help

**Step 3: Commit**

```bash
git add cmd/archive.go
git commit -m "feat: add cb archive command"
```

**Phase 4 Checkpoint:**
```bash
make verify  # Must pass before continuing
```

---

## Phase 5: TUI Dashboard

### Task 15: Add Bubbletea Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add Bubbletea dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add Bubbletea TUI dependencies"
```

---

### Task 16: Create Dashboard Model

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/model_test.go`

**Step 1: Write test for model**

Create `internal/tui/model_test.go`:

```go
package tui

import (
	"testing"

	"github.com/rsanzone/clawdbay/internal/tmux"
)

func TestModel_GroupSessionsByWorktree(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "cb:proj-123-auth"},
		{Name: "cb:proj-456-bug"},
	}

	windows := map[string][]tmux.Window{
		"cb:proj-123-auth": {
			{Name: "shell"},
			{Name: "claude:default"},
			{Name: "claude:research"},
		},
		"cb:proj-456-bug": {
			{Name: "shell"},
			{Name: "claude:default"},
		},
	}

	// Pass nil for tmuxClient in tests (status detection skipped)
	groups := GroupByWorktree(sessions, windows, nil)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}

	if groups[0].SessionCount != 2 {
		t.Errorf("first group sessions = %d, want 2", groups[0].SessionCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/...`
Expected: FAIL - package doesn't exist

**Step 3: Write implementation**

Create `internal/tui/model.go`:

```go
package tui

import (
	"strings"
	"time"

	"github.com/rsanzone/clawdbay/internal/tmux"
)

type WorktreeGroup struct {
	Name         string
	SessionCount int
	Sessions     []ClaudeSession
	Expanded     bool
}

type ClaudeSession struct {
	Name       string
	Status     string // IDLE, WORKING, DONE
	LastActive time.Time
}

type Model struct {
	Groups   []WorktreeGroup
	Cursor   int
	Quitting bool
}

// GroupByWorktree groups sessions and their Claude windows.
// The tmuxClient parameter is used to detect window status.
func GroupByWorktree(sessions []tmux.Session, windows map[string][]tmux.Window, tmuxClient *tmux.Client) []WorktreeGroup {
	var groups []WorktreeGroup

	for _, session := range sessions {
		wins := windows[session.Name]

		var claudeSessions []ClaudeSession
		for _, w := range wins {
			if strings.HasPrefix(w.Name, "claude:") {
				status := string(tmux.StatusIdle)
				if tmuxClient != nil {
					status = string(tmuxClient.GetPaneStatus(session.Name, w.Name))
				}
				claudeSessions = append(claudeSessions, ClaudeSession{
					Name:   w.Name,
					Status: status,
				})
			}
		}

		groups = append(groups, WorktreeGroup{
			Name:         session.Name,
			SessionCount: len(claudeSessions),
			Sessions:     claudeSessions,
			Expanded:     true,
		})
	}

	return groups
}

func InitialModel() Model {
	return Model{
		Groups: []WorktreeGroup{},
		Cursor: 0,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI model with worktree grouping"
```

---

### Task 17: Implement Bubbletea Update/View

**Files:**
- Modify: `internal/tui/model.go`
- Create: `internal/tui/view.go`

**Step 1: Add Bubbletea interfaces**

Add to `internal/tui/model.go`:

```go
import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Groups)-1 {
				m.Cursor++
			}
		case "enter":
			// Would attach to session
			return m, tea.Quit
		}
	}
	return m, nil
}
```

**Step 2: Create view**

Create `internal/tui/view.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("212"))

	idleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	workingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	doneStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("─ ClawdBay ") + strings.Repeat("─", 50) + "\n\n")

	if len(m.Groups) == 0 {
		b.WriteString("  No active workflows.\n")
		b.WriteString("  Start one with: cb start <ticket-id>\n")
	} else {
		for i, group := range m.Groups {
			cursor := "  "
			if i == m.Cursor {
				cursor = "> "
			}

			// Group header
			expandIcon := "▼"
			if !group.Expanded {
				expandIcon = "▶"
			}

			line := fmt.Sprintf("%s%s %s    %d sessions",
				cursor, expandIcon, group.Name, group.SessionCount)

			if i == m.Cursor {
				b.WriteString(selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}

			// Sessions
			if group.Expanded {
				for _, session := range group.Sessions {
					statusIcon := "●"
					var statusStyle lipgloss.Style

					switch session.Status {
					case "IDLE":
						statusStyle = idleStyle
					case "WORKING":
						statusStyle = workingStyle
					case "DONE":
						statusStyle = doneStyle
					}

					sessionLine := fmt.Sprintf("      %s %s  %s",
						statusStyle.Render(statusIcon+" "+session.Status),
						session.Name,
						"")
					b.WriteString(sessionLine + "\n")
				}
			}
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString("  [Enter] Attach  [n] New  [c] Add Claude  [p] Add Prompt  [x] Archive  [q] Quit\n")

	return b.String()
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/tui/
git commit -m "feat: implement Bubbletea Update and View"
```

---

### Task 18: Add `cb dash` Command

**Files:**
- Create: `cmd/dash.go`
- Modify: `cmd/root.go`

**Step 1: Create dash command**

Create `cmd/dash.go`:

```go
package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/rsanzone/clawdbay/internal/tui"
	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Open interactive dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxClient := tmux.NewClient()
		sessions, err := tmuxClient.ListSessions()
		if err != nil {
			return err
		}

		// Get windows for each session
		windows := make(map[string][]tmux.Window)
		for _, s := range sessions {
			wins, err := tmuxClient.ListWindows(s.Name)
			if err == nil {
				windows[s.Name] = wins
			}
		}

		// Build model
		model := tui.InitialModel()
		model.Groups = tui.GroupByWorktree(sessions, windows, tmuxClient)

		// Run TUI
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Handle selection
		if m, ok := finalModel.(tui.Model); ok && !m.Quitting {
			if m.Cursor < len(m.Groups) {
				sessionName := m.Groups[m.Cursor].Name
				fmt.Printf("Attaching to %s...\n", sessionName)
				return tmuxClient.SwitchClient(sessionName)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashCmd)
}
```

**Step 2: Add ListWindows to tmux client**

Add to `internal/tmux/tmux.go`:

```go
func (c *Client) ListWindows(session string) ([]Window, error) {
	output, err := c.execCommand("tmux", "list-windows", "-t", session, "-F", "#{window_index}:#{window_name}:#{window_active}")
	if err != nil {
		return nil, err
	}
	return ParseWindowList(string(output)), nil
}
```

**Step 3: Update root command to default to dash**

Modify `cmd/root.go` Run function:

```go
Run: func(cmd *cobra.Command, args []string) {
	// Default to dashboard
	dashCmd.Run(cmd, args)
},
```

**Step 4: Verify dashboard runs**

Run: `go run main.go dash`
Expected: Shows TUI with "No active workflows" or list of sessions

**Step 5: Commit**

```bash
git add cmd/dash.go cmd/root.go internal/tmux/tmux.go
git commit -m "feat: add cb dash command with TUI"
```

**Phase 5 Checkpoint:**
```bash
make verify  # Must pass before continuing
```

---

## Phase 6: Polish and Testing

### Task 19: Add Integration Test

**Files:**
- Create: `integration_test.go`

**Step 1: Create basic integration test**

Create `integration_test.go`:

```go
//go:build integration

package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	if !strings.Contains(string(output), "ClawdBay") {
		t.Errorf("output = %q, want to contain 'ClawdBay'", output)
	}
}

func TestCLI_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	expected := []string{"start", "claude", "prompt", "list", "archive", "dash"}
	for _, sub := range expected {
		if !strings.Contains(string(output), sub) {
			t.Errorf("help missing subcommand: %s", sub)
		}
	}
}
```

**Step 2: Run integration tests**

Run: `go test -tags=integration ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: add CLI integration tests"
```

---

### Task 20: Create Default Prompt Templates

**Files:**
- Create: `templates/prompts/research.md`
- Create: `templates/prompts/plan.md`
- Create: `templates/prompts/implement.md`
- Create: `templates/prompts/verify.md`

**Note:** Templates are plain markdown - users copy and customize them per-worktree. No variable substitution needed.

**Step 1: Create research template**

Create `templates/prompts/research.md`:

```markdown
# Research

## Context

<!-- Add ticket ID and description here -->

## Instructions

1. Explore the codebase to understand the current implementation
2. Identify key files, components, and patterns involved
3. Document your findings in `docs/plans/YYYY-MM-DD-<topic>-research.md`

## Output

Produce a research report with:
- Current state analysis with file references
- Key components and their interactions
- Potential approaches with trade-offs
- Open questions needing clarification
- Mermaid diagrams where helpful

Use the brainstorming skill if available for structured exploration.
```

**Step 2: Create plan template**

Create `templates/prompts/plan.md`:

```markdown
# Implementation Plan

## Context

<!-- Add ticket ID and description here -->

## Research

Review the research document if available: `docs/plans/*-research.md`

## Instructions

Create a detailed implementation plan following TDD principles:

1. Break work into small, testable tasks
2. Each task: write failing test -> implement -> verify -> commit
3. Include exact file paths and code snippets
4. Document expected test commands and outputs

Save plan to: `docs/plans/YYYY-MM-DD-<topic>-implementation.md`

Use the writing-plans skill format if available.
```

**Step 3: Create implement template**

Create `templates/prompts/implement.md`:

```markdown
# Implementation

## Plan

Follow the implementation plan: `docs/plans/*-implementation.md`

## Instructions

Execute the plan task by task:

1. Read the current task from the plan
2. Write the failing test first
3. Implement minimal code to pass
4. Run tests to verify
5. Commit with clear message
6. Move to next task

Use the executing-plans skill if available.

## Checkpoints

Stop and verify at:
- After each task completes
- When encountering unexpected behavior
- Before starting a new component
```

**Step 4: Create verify template**

Create `templates/prompts/verify.md`:

```markdown
# Verification

## Checklist

Run through this verification checklist:

### Code Quality
- [ ] All tests pass: `go test ./...`
- [ ] Linting clean: `golangci-lint run`
- [ ] No forbidden patterns (time.Sleep, interface{}, etc.)

### Functionality
- [ ] Feature works end-to-end
- [ ] Edge cases handled
- [ ] Error messages are helpful

### Documentation
- [ ] Code is self-documenting
- [ ] Complex logic has comments
- [ ] Public APIs have godoc

### Cleanup
- [ ] No debug code left
- [ ] No TODOs in final code
- [ ] Old code removed (no migration layers)

Report any issues found and fix them before completion.
```

**Step 5: Commit**

```bash
git add templates/
git commit -m "feat: add default prompt templates"
```

---

### Task 21: Add Install Command for Templates

**Files:**
- Create: `templates/prompts/research.md` (from Task 22)
- Create: `templates/prompts/plan.md` (from Task 22)
- Create: `templates/prompts/implement.md` (from Task 22)
- Create: `templates/prompts/verify.md` (from Task 22)
- Create: `templates/embed.go`
- Create: `cmd/init.go`

**Note:** embed.FS paths are relative to the Go file. We create a small package at `templates/` to hold the embed directive.

**Step 1: Create templates embed package**

Create `templates/embed.go`:

```go
package templates

import "embed"

//go:embed prompts/*.md
var FS embed.FS
```

**Step 2: Create init command**

Create `cmd/init.go`:

```go
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/rsanzone/clawdbay/templates"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ClawdBay configuration and templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.New()

		// Create directories
		if err := cfg.EnsureDirs(); err != nil {
			return fmt.Errorf("failed to create config directories: %w", err)
		}
		fmt.Printf("Created: %s\n", cfg.ConfigDir)

		// Copy templates
		err := fs.WalkDir(templates.FS, "prompts", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			content, err := templates.FS.ReadFile(path)
			if err != nil {
				return err
			}

			dstPath := filepath.Join(cfg.PromptsDir, filepath.Base(path))

			// Don't overwrite existing
			if _, err := os.Stat(dstPath); err == nil {
				fmt.Printf("Skipped (exists): %s\n", dstPath)
				return nil
			}

			if err := os.WriteFile(dstPath, content, 0644); err != nil {
				return err
			}
			fmt.Printf("Created: %s\n", dstPath)
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to copy templates: %w", err)
		}

		fmt.Println("\nClawdBay initialized! Run 'cb start <ticket>' to begin.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
```

**Step 2: Verify init command**

Run: `go run main.go init`
Expected: Creates ~/.config/cb/prompts/ with template files

**Step 3: Commit**

```bash
git add cmd/init.go
git commit -m "feat: add cb init command to install templates"
```

---

### Task 22: Final Build and Test

**Step 1: Run all tests**

```bash
go test ./...
```
Expected: All tests pass

**Step 2: Build binary**

```bash
go build -o cb main.go
```
Expected: Creates `cb` binary

**Step 3: Test binary**

```bash
./cb --help
./cb version
./cb init
./cb prompt list
```
Expected: All commands work

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: ClawdBay v0.1.0 complete

Multi-session Claude workflow manager with:
- cb start: Create worktree + tmux session
- cb claude: Add Claude sessions to worktree
- cb prompt: Manage and execute prompt templates
- cb dash: Interactive TUI dashboard
- cb archive: Clean up completed workflows

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

**Total tasks:** 23 (Task 0 through Task 22)
**Phases:** 7 (Phase 0 through Phase 6)

**Key deliverables:**
1. `cb` CLI with commands: start, claude, prompt, list, archive, dash, init, version
2. tmux integration for session management with status detection
3. Bubbletea TUI dashboard
4. Default prompt templates (plain markdown, no variable substitution)

**Testing approach:** TDD throughout - every feature has tests before implementation.

**Verification:** `make verify` checkpoint at end of each phase.

**Next steps after implementation:**
1. Add tmux keybinding for `cb dash`
2. Improve status detection with timestamp-based activity checks
3. Create additional prompt templates
4. Consider adding session coordination features
5. Optional: Add jira-cli integration as a plugin/extension
