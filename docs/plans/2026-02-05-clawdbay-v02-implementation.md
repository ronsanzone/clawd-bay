# ClawdBay v0.2 Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Simplify ClawdBay to focus on cross-repo session orchestration, removing the prompt/template system and rebuilding the TUI with a three-level hierarchy.

**Architecture:** Stateless tmux-derived state. Five commands: `cb dash`, `cb start`, `cb claude`, `cb list`, `cb archive`. TUI shows Repo > Session > Window tree with per-window status and auto-refresh.

**Tech Stack:** Go 1.25, Cobra CLI, Bubbletea TUI, Lipgloss styling, tmux IPC

**Design doc:** `docs/plans/2026-02-05-clawdbay-simplification-design.md`

**Verification commands:**
```bash
go test ./...                    # Unit tests
golangci-lint run                # Lint
make verify                      # Both
go test -tags=integration ./...  # E2E (requires tmux)
```

---

## Phase 1: Remove Dead Code

### Task 1: Delete prompt/template system files

**Files:**
- Delete: `cmd/prompt.go`
- Delete: `cmd/init.go`
- Delete: `internal/prompt/prompt.go`
- Delete: `internal/prompt/prompt_test.go`
- Delete: `templates/embed.go`
- Delete: `templates/prompts/research.md`
- Delete: `templates/prompts/plan.md`
- Delete: `templates/prompts/implement.md`
- Delete: `templates/prompts/verify.md`

**Step 1: Delete the files**

```bash
rm cmd/prompt.go cmd/init.go
rm -rf internal/prompt/
rm -rf templates/
```

**Step 2: Run tests to see what breaks**

Run: `go build ./...`
Expected: FAIL — `cmd/init.go` referenced `templates.FS` and `internal/config`, `cmd/prompt.go` referenced `internal/prompt`

Since the `init()` functions in the deleted files registered subcommands on `rootCmd`, and those files are now gone, the commands simply won't register anymore. No code in other files references `promptCmd`, `initCmd`, or the `prompt`/`templates` packages — except the integration test.

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: PASS (the deleted files' `init()` functions were self-contained)

**Step 4: Run full test suite**

Run: `go test ./...`
Expected: PASS (no remaining code imports `internal/prompt` or `templates`)

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove prompt/template system

Remove cb prompt, cb init, templates/, and internal/prompt/.
Skills handle prompting now — this code is dead weight."
```

---

### Task 2: Delete version subcommand, add --version flag

**Files:**
- Delete: `cmd/version.go`
- Modify: `cmd/root.go`

**Step 1: Delete version command file**

```bash
rm cmd/version.go
```

**Step 2: Move version to root command as a flag**

Edit `cmd/root.go` — add a `--version` flag to the root command:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the current version of ClawdBay.
var Version = "0.2.0"

var rootCmd = &cobra.Command{
	Use:     "cb",
	Short:   "ClawdBay - A harbor for your Claude sessions",
	Long:    `ClawdBay manages multi-session Claude Code workflows.

Start workflows with git worktrees, manage multiple Claude sessions
per worktree, and track session status from an interactive dashboard.`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to dashboard
		if err := dashCmd.RunE(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Note: Setting `Version` on the cobra command automatically adds a `--version` flag.

**Step 3: Run tests**

Run: `go build ./... && go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: replace version subcommand with --version flag"
```

---

### Task 3: Simplify config package (remove PromptsDir)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Step 1: Simplify Config struct**

The `Config` struct currently has `PromptsDir` which is only used by the deleted prompt commands. Remove it:

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds ClawdBay configuration paths.
type Config struct {
	ConfigDir string
}

// New creates a Config with default paths.
func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(home, ".config", "cb")

	return &Config{
		ConfigDir: configDir,
	}, nil
}

// EnsureDirs creates the config directory if it doesn't exist.
func (c *Config) EnsureDirs() error {
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}
```

**Step 2: Update config_test.go if it references PromptsDir**

Read the test file and remove any PromptsDir assertions.

**Step 3: Run tests**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/config/
git commit -m "refactor: remove PromptsDir from config (prompt system deleted)"
```

---

### Task 4: Update integration test (remove prompt/init references)

**Files:**
- Modify: `integration_test.go`

**Step 1: Update TestCLI_Help to remove prompt from expected subcommands**

In `integration_test.go:34`, the help test expects `"prompt"` in the output. Remove it:

```go
expected := []string{"start", "claude", "list", "archive", "dash"}
```

**Step 2: Run integration test compile check**

Run: `go vet ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: update integration test for removed prompt command"
```

---

## Phase 2: tmux Layer Enhancements

### Task 5: Add GetPaneWorkingDir to tmux client

**Files:**
- Modify: `internal/tmux/tmux.go`
- Modify: `internal/tmux/tmux_test.go`

**Step 1: Write the failing test**

Add to `internal/tmux/tmux_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/ -run TestClient_GetPaneWorkingDir -v`
Expected: FAIL — `GetPaneWorkingDir` not defined

**Step 3: Write the implementation**

Add to `internal/tmux/tmux.go`:

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/ -run TestClient_GetPaneWorkingDir -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add GetPaneWorkingDir to tmux client"
```

---

### Task 6: Add GetRepoName helper to tmux client

This derives a repo name from a session by querying the pane's working directory and running `git rev-parse --show-toplevel`.

**Files:**
- Modify: `internal/tmux/tmux.go`
- Modify: `internal/tmux/tmux_test.go`

**Step 1: Write the failing test**

Add to `internal/tmux/tmux_test.go`:

```go
func TestClient_GetRepoName(t *testing.T) {
	tests := []struct {
		name     string
		outputs  map[string]string
		expected string
	}{
		{
			name: "derives repo from pane path",
			outputs: map[string]string{
				"tmux":  "/Users/ron/code/my-project/.worktrees/my-project-feat",
				"git":   "/Users/ron/code/my-project\n",
			},
			expected: "my-project",
		},
		{
			name: "tmux error returns unknown",
			outputs: map[string]string{},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			client := &Client{
				execCommand: func(name string, args ...string) ([]byte, error) {
					callCount++
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tmux/ -run TestClient_GetRepoName -v`
Expected: FAIL — `GetRepoName` not defined

**Step 3: Write the implementation**

Add to `internal/tmux/tmux.go`:

```go
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
```

Also add `"path/filepath"` to the imports in `tmux.go`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tmux/ -run TestClient_GetRepoName -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tmux/
git commit -m "feat: add GetRepoName for cross-repo session discovery"
```

---

## Phase 3: Update CLI Commands

### Task 7: Update `cb start` — worktree path and Claude window

**Files:**
- Modify: `cmd/start.go`

**Step 1: Update worktree directory calculation**

Change `cmd/start.go` line 48 from:
```go
worktreeDir := filepath.Join(filepath.Dir(cwd), projectName+"-"+branchName)
```
to:
```go
worktreeDir := filepath.Join(cwd, ".worktrees", projectName+"-"+branchName)
```

**Step 2: Add .worktrees/ to .gitignore if not present**

After the worktree directory check (line 53), add gitignore management:

```go
// Ensure .worktrees/ is in .gitignore
ensureGitignoreEntry(cwd, ".worktrees/")
```

Add this helper function to the file:

```go
// ensureGitignoreEntry adds an entry to .gitignore if not already present.
func ensureGitignoreEntry(repoDir, entry string) {
	gitignorePath := filepath.Join(repoDir, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err == nil {
		// Check if entry already exists
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == entry {
				return
			}
		}
	}

	// Append entry
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Best effort — don't fail the whole command
	}
	defer func() { _ = f.Close() }()

	// Add newline before entry if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(entry + "\n")
}
```

**Step 3: Ensure .worktrees directory exists**

Before the worktree creation, add:

```go
// Ensure .worktrees directory exists
worktreesDir := filepath.Join(cwd, ".worktrees")
if err := os.MkdirAll(worktreesDir, 0755); err != nil {
	return fmt.Errorf("failed to create .worktrees directory: %w", err)
}
```

**Step 4: Add Claude window creation after session creation**

After the `CreateSession` call (line 84), add:

```go
// Create a Claude window in the session
if err := tmuxClient.CreateWindow(sessionName, "claude", "claude"); err != nil {
	fmt.Fprintf(os.Stderr, "Warning: failed to create Claude window: %v\n", err)
	// Non-fatal — session still usable without Claude window
}
```

**Step 5: Run build**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/start.go
git commit -m "feat: update cb start for .worktrees/ path and auto Claude window"
```

---

### Task 8: Update `cb claude` — remove --prompt flag

**Files:**
- Modify: `cmd/claude.go`

**Step 1: Remove the --prompt flag and related code**

Remove the `claudePrompt` variable, the flag registration, and the prompt-file handling logic. The simplified `runClaude` function:

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

var claudeName string

var claudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Add a Claude session to current worktree",
	Long: `Creates a new tmux window with a Claude session.

Example:
  cb claude                  # Creates default session
  cb claude --name research  # Named session`,
	RunE: runClaude,
}

func init() {
	claudeCmd.Flags().StringVarP(&claudeName, "name", "n", "default", "Name for the Claude session")
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
		output, execErr := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
		if execErr == nil {
			currentSession := strings.TrimSpace(string(output))
			if strings.HasPrefix(currentSession, "cb_") {
				sessionName = currentSession
			}
		}
	}

	// If not in a cb_ session, try to find one matching this directory
	if sessionName == "" {
		sessions, listErr := tmuxClient.ListSessions()
		if listErr != nil {
			return fmt.Errorf("failed to list sessions: %w", listErr)
		}

		dirName := filepath.Base(cwd)
		for _, s := range sessions {
			sessionSuffix := strings.TrimPrefix(s.Name, "cb_")
			if strings.Contains(dirName, sessionSuffix) {
				sessionName = s.Name
				break
			}
		}
	}

	if sessionName == "" {
		return fmt.Errorf("no cb_ session found for this directory. Run 'cb start' first")
	}

	// Create window name
	windowName := "claude:" + claudeName

	// Create window with claude
	fmt.Printf("Creating Claude session: %s in %s\n", windowName, sessionName)
	if err := tmuxClient.CreateWindow(sessionName, windowName, "claude"); err != nil {
		return fmt.Errorf("failed to create Claude window: %w", err)
	}

	// Switch to the new window
	selectCmd := exec.Command("tmux", "select-window", "-t", sessionName+":"+windowName)
	if err := selectCmd.Run(); err != nil {
		return fmt.Errorf("failed to select window: %w", err)
	}
	return nil
}
```

Note: also uses `tmuxClient.CreateWindow` instead of raw `exec.Command` for the window creation.

**Step 2: Run tests**

Run: `go build ./... && go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/claude.go
git commit -m "refactor: remove --prompt flag from cb claude"
```

---

### Task 9: Update `cb archive` — worktree path detection for .worktrees/

**Files:**
- Modify: `cmd/archive.go`

**Step 1: Update worktree path detection**

The current archive command detects worktree by checking `filepath.Base(cwd)` against session suffixes. With `.worktrees/` inside the repo, the logic needs to also handle the case where the cwd is inside `.worktrees/`.

The key change: when a session name is provided as an argument (without auto-detect from cwd), we need to find the worktree path. We can query tmux for the pane's working directory.

Update the archive command to use `tmuxClient.GetPaneWorkingDir()` when a session name is explicitly provided:

In the `len(args) > 0` branch, after resolving `sessionName`, add:

```go
// Try to find worktree path from session's pane
tmuxClient := tmux.NewClient()
paneDir := tmuxClient.GetPaneWorkingDir(sessionName)
if paneDir != "" {
	worktreePath = paneDir
}
```

This ensures archive can find and remove the worktree even when not running from within it.

**Step 2: Run build**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/archive.go
git commit -m "feat: archive uses tmux pane dir for worktree discovery"
```

---

## Phase 4: TUI Data Model Rewrite

### Task 10: Define the new three-level data model

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Write the new types**

Replace the existing types in `model.go` with the three-level hierarchy:

```go
package tui

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

// NodeType represents what kind of tree node the cursor is on.
type NodeType int

const (
	NodeRepo    NodeType = iota
	NodeSession
	NodeWindow
)

// RepoGroup represents a repository with its worktree sessions.
type RepoGroup struct {
	Name     string
	Path     string
	Sessions []WorktreeSession
	Expanded bool
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
	Type         NodeType
	RepoIndex    int
	SessionIndex int
	WindowIndex  int
}

// Model is the Bubbletea model for the dashboard.
type Model struct {
	Groups      []RepoGroup
	Cursor      int
	Nodes       []TreeNode // Flattened tree for cursor navigation
	Quitting    bool
	TmuxClient  *tmux.Client
	SelectedName string // Name of what was selected (for attach after quit)
	SelectedWindow string // Window name if a window was selected
}

// tickMsg triggers periodic refresh.
type tickMsg time.Time

// refreshMsg carries new data from a refresh.
type refreshMsg struct {
	Groups []RepoGroup
}
```

**Step 2: Run build to check types compile**

Run: `go build ./internal/tui/...`
Expected: Likely FAIL because the old `GroupByWorktree` and other functions reference old types. That's fine — we'll fix those next.

**Step 3: Don't commit yet — continue to Task 11**

---

### Task 11: Implement BuildTree and GroupByRepo

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`

**Step 1: Write the failing test for GroupByRepo**

Replace `internal/tui/model_test.go`:

```go
package tui

import (
	"testing"

	"github.com/rsanzone/clawdbay/internal/tmux"
)

func TestGroupByRepo(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "cb_feat-auth"},
		{Name: "cb_refactor"},
		{Name: "cb_fix-login"},
	}

	// Mock: first two sessions are in "my-project", third is in "other-project"
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

	// Find my-project group
	var myProject *RepoGroup
	for i := range groups {
		if groups[i].Name == "my-project" {
			myProject = &groups[i]
			break
		}
	}
	if myProject == nil {
		t.Fatal("my-project group not found")
	}

	if len(myProject.Sessions) != 2 {
		t.Errorf("my-project has %d sessions, want 2", len(myProject.Sessions))
	}

	// Check status rollup: feat-auth has WORKING and IDLE, should roll up to WORKING
	for _, s := range myProject.Sessions {
		if s.Name == "cb_feat-auth" {
			if s.Status != tmux.StatusWorking {
				t.Errorf("cb_feat-auth status = %q, want %q", s.Status, tmux.StatusWorking)
			}
		}
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

	// Expected nodes:
	// 0: Repo "my-project" (expanded)
	// 1: Session "cb_feat-auth" (expanded)
	// 2: Window "shell"
	// 3: Window "claude"
	// 4: Session "cb_refactor" (collapsed — no children)
	// 5: Repo "other-project" (collapsed — no children)
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
	if nodes[5].Type != NodeRepo {
		t.Errorf("node 5 type = %v, want NodeRepo", nodes[5].Type)
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v`
Expected: FAIL — `GroupByRepo`, `BuildNodes`, `RollupStatus` not defined

**Step 3: Write the implementation**

Add to `internal/tui/model.go`:

```go
// RollupStatus returns the most active status from a slice.
// Priority: WORKING > IDLE > DONE
func RollupStatus(statuses []tmux.Status) tmux.Status {
	hasIdle := false
	for _, s := range statuses {
		if s == tmux.StatusWorking {
			return tmux.StatusWorking
		}
		if s == tmux.StatusIdle {
			hasIdle = true
		}
	}
	if hasIdle {
		return tmux.StatusIdle
	}
	return tmux.StatusDone
}

// GroupByRepo groups sessions by their repository name.
// repoNames maps session name -> repo name.
// windows maps session name -> window list.
// statuses maps "session:window" -> status.
func GroupByRepo(
	sessions []tmux.Session,
	repoNames map[string]string,
	windows map[string][]tmux.Window,
	statuses map[string]tmux.Status,
) []RepoGroup {
	repoMap := make(map[string]*RepoGroup)
	var repoOrder []string

	for _, session := range sessions {
		repoName := repoNames[session.Name]
		if repoName == "" {
			repoName = "Unknown"
		}

		if _, exists := repoMap[repoName]; !exists {
			repoMap[repoName] = &RepoGroup{
				Name:     repoName,
				Expanded: true,
			}
			repoOrder = append(repoOrder, repoName)
		}

		wins := windows[session.Name]
		var windowStatuses []tmux.Status
		for i := range wins {
			key := session.Name + ":" + wins[i].Name
			if status, ok := statuses[key]; ok {
				wins[i].Active = false // Reset — we use status instead
				windowStatuses = append(windowStatuses, status)
			}
		}

		ws := WorktreeSession{
			Name:     session.Name,
			Status:   RollupStatus(windowStatuses),
			Windows:  wins,
			Expanded: true,
		}

		repoMap[repoName].Sessions = append(repoMap[repoName].Sessions, ws)
	}

	var groups []RepoGroup
	for _, name := range repoOrder {
		groups = append(groups, *repoMap[name])
	}
	return groups
}

// BuildNodes flattens the tree into a list of navigable nodes.
func BuildNodes(groups []RepoGroup) []TreeNode {
	var nodes []TreeNode

	for ri, repo := range groups {
		nodes = append(nodes, TreeNode{Type: NodeRepo, RepoIndex: ri})

		if !repo.Expanded {
			continue
		}

		for si, session := range repo.Sessions {
			nodes = append(nodes, TreeNode{Type: NodeSession, RepoIndex: ri, SessionIndex: si})

			if !session.Expanded {
				continue
			}

			for wi := range session.Windows {
				nodes = append(nodes, TreeNode{Type: NodeWindow, RepoIndex: ri, SessionIndex: si, WindowIndex: wi})
			}
		}
	}

	return nodes
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: three-level data model with GroupByRepo and tree navigation"
```

---

### Task 12: Rewrite TUI Update (navigation + actions)

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Replace the Update function**

Replace the `InitialModel`, `Init`, and `Update` functions:

```go
const refreshInterval = 3 * time.Second

// InitialModel creates the initial dashboard model.
func InitialModel(tmuxClient *tmux.Client) Model {
	return Model{
		Groups:     []RepoGroup{},
		TmuxClient: tmuxClient,
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
		groups := fetchGroups(m.TmuxClient)
		return refreshMsg{Groups: groups}
	}
}

// fetchGroups queries tmux for all session data and builds repo groups.
func fetchGroups(tmuxClient *tmux.Client) []RepoGroup {
	if tmuxClient == nil {
		return nil
	}

	sessions, err := tmuxClient.ListSessions()
	if err != nil {
		return nil
	}

	repoNames := make(map[string]string)
	windowMap := make(map[string][]tmux.Window)
	statusMap := make(map[string]tmux.Status)

	for _, s := range sessions {
		repoNames[s.Name] = tmuxClient.GetRepoName(s.Name)

		wins, winErr := tmuxClient.ListWindows(s.Name)
		if winErr != nil {
			continue
		}
		windowMap[s.Name] = wins

		for _, w := range wins {
			if strings.HasPrefix(w.Name, "claude") {
				statusMap[s.Name+":"+w.Name] = tmuxClient.GetPaneStatus(s.Name, w.Name)
			}
		}
	}

	return GroupByRepo(sessions, repoNames, windowMap, statusMap)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		m.Groups = mergeExpandState(m.Groups, msg.Groups)
		m.Nodes = BuildNodes(m.Groups)
		if m.Cursor >= len(m.Nodes) {
			m.Cursor = max(0, len(m.Nodes)-1)
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.refreshCmd(), m.tickCmd())

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
			if m.Cursor < len(m.Nodes)-1 {
				m.Cursor++
			}

		case "enter":
			return m.handleEnter()

		case "l", "right":
			return m.handleExpand()

		case "h", "left":
			return m.handleCollapse()

		case "r":
			return m, m.refreshCmd()
		}
	}
	return m, nil
}

// mergeExpandState preserves expand/collapse state across refreshes.
func mergeExpandState(old, updated []RepoGroup) []RepoGroup {
	oldState := make(map[string]bool)
	oldSessionState := make(map[string]bool)

	for _, g := range old {
		oldState[g.Name] = g.Expanded
		for _, s := range g.Sessions {
			oldSessionState[s.Name] = s.Expanded
		}
	}

	for i := range updated {
		if expanded, ok := oldState[updated[i].Name]; ok {
			updated[i].Expanded = expanded
		}
		for j := range updated[i].Sessions {
			if expanded, ok := oldSessionState[updated[i].Sessions[j].Name]; ok {
				updated[i].Sessions[j].Expanded = expanded
			}
		}
	}
	return updated
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.Cursor >= len(m.Nodes) {
		return m, nil
	}
	node := m.Nodes[m.Cursor]

	switch node.Type {
	case NodeRepo:
		m.Groups[node.RepoIndex].Expanded = !m.Groups[node.RepoIndex].Expanded
		m.Nodes = BuildNodes(m.Groups)
	case NodeSession:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		m.SelectedName = session.Name
		return m, tea.Quit
	case NodeWindow:
		session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
		window := session.Windows[node.WindowIndex]
		m.SelectedName = session.Name
		m.SelectedWindow = window.Name
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
	case NodeSession:
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = true
		m.Nodes = BuildNodes(m.Groups)
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
	case NodeSession:
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
	case NodeWindow:
		// Collapse parent session
		m.Groups[node.RepoIndex].Sessions[node.SessionIndex].Expanded = false
		m.Nodes = BuildNodes(m.Groups)
	}
	return m, nil
}
```

**Step 2: Run build**

Run: `go build ./internal/tui/...`
Expected: PASS

**Step 3: Run tests**

Run: `go test ./internal/tui/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat: TUI tree navigation with expand/collapse and auto-refresh"
```

---

### Task 13: Rewrite TUI View (three-level rendering)

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Replace the View function**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	repoStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141"))

	sessionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	windowStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	idleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	workingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	doneStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("─ ClawdBay ") + strings.Repeat("─", 50) + "\n\n")

	if len(m.Groups) == 0 {
		b.WriteString("  No active sessions.\n")
		b.WriteString("  Start one with: cb start <branch-name>\n")
	} else {
		for i, node := range m.Nodes {
			isSelected := i == m.Cursor
			line := m.renderNode(node, isSelected)
			b.WriteString(line + "\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderNode(node TreeNode, selected bool) string {
	switch node.Type {
	case NodeRepo:
		return m.renderRepoNode(node, selected)
	case NodeSession:
		return m.renderSessionNode(node, selected)
	case NodeWindow:
		return m.renderWindowNode(node, selected)
	}
	return ""
}

func (m Model) renderRepoNode(node TreeNode, selected bool) string {
	repo := m.Groups[node.RepoIndex]
	icon := "▸"
	if repo.Expanded {
		icon = "▼"
	}

	line := fmt.Sprintf("%s %s", icon, repo.Name)
	if selected {
		return selectedStyle.Render(line)
	}
	return repoStyle.Render(line)
}

func (m Model) renderSessionNode(node TreeNode, selected bool) string {
	session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]

	icon := "▸"
	if session.Expanded {
		icon = "▼"
	}

	statusStr := renderStatus(session.Status)
	line := fmt.Sprintf("  %s %s", icon, session.Name)
	padding := 40 - len(line)
	if padding < 2 {
		padding = 2
	}

	if selected {
		return selectedStyle.Render(line) + strings.Repeat(" ", padding) + statusStr
	}
	return sessionStyle.Render(line) + strings.Repeat(" ", padding) + statusStr
}

func (m Model) renderWindowNode(node TreeNode, selected bool) string {
	session := m.Groups[node.RepoIndex].Sessions[node.SessionIndex]
	window := session.Windows[node.WindowIndex]

	line := fmt.Sprintf("      %s", window.Name)

	// Show status for claude windows
	if strings.HasPrefix(window.Name, "claude") {
		// Look up per-window status from the session's perspective
		statusStr := ""
		if m.TmuxClient != nil {
			status := m.TmuxClient.GetPaneStatus(session.Name, window.Name)
			statusStr = renderStatus(status)
		}
		padding := 40 - len(line)
		if padding < 2 {
			padding = 2
		}
		if selected {
			return selectedStyle.Render(line) + strings.Repeat(" ", padding) + statusStr
		}
		return windowStyle.Render(line) + strings.Repeat(" ", padding) + statusStr
	}

	if selected {
		return selectedStyle.Render(line)
	}
	return windowStyle.Render(line)
}

func renderStatus(status tmux.Status) string {
	switch status {
	case tmux.StatusWorking:
		return workingStyle.Render("● WORKING")
	case tmux.StatusIdle:
		return idleStyle.Render("○ IDLE")
	case tmux.StatusDone:
		return doneStyle.Render("◌ DONE")
	default:
		return doneStyle.Render("◌ DONE")
	}
}

func (m Model) renderFooter() string {
	if len(m.Nodes) == 0 {
		return footerStyle.Render("  [n] new  [q] quit")
	}

	if m.Cursor >= len(m.Nodes) {
		return footerStyle.Render("  [q] quit")
	}

	node := m.Nodes[m.Cursor]
	switch node.Type {
	case NodeRepo:
		return footerStyle.Render("  [enter] expand  [n] new  [q] quit")
	case NodeSession:
		return footerStyle.Render("  [enter] attach  [c] add claude  [x] archive  [r] refresh  [q] quit")
	case NodeWindow:
		return footerStyle.Render("  [enter] attach  [r] refresh  [q] quit")
	}
	return footerStyle.Render("  [q] quit")
}
```

**Step 2: Run build**

Run: `go build ./internal/tui/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat: three-level TUI rendering with per-window status"
```

---

### Task 14: Update dash command to use new TUI model

**Files:**
- Modify: `cmd/dash.go`

**Step 1: Rewrite dash.go**

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"

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

		// Build model — data loads via Init() refresh
		model := tui.InitialModel(tmuxClient)

		// Run TUI
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Handle selection (attach to session after TUI exits)
		if m, ok := finalModel.(tui.Model); ok && m.SelectedName != "" {
			// Select specific window if one was chosen
			if m.SelectedWindow != "" {
				selectCmd := exec.Command("tmux", "select-window", "-t", m.SelectedName+":"+m.SelectedWindow)
				_ = selectCmd.Run()
			}

			fmt.Printf("Attaching to %s...\n", m.SelectedName)
			if os.Getenv("TMUX") != "" {
				return tmuxClient.SwitchClient(m.SelectedName)
			}
			return tmuxClient.AttachSession(m.SelectedName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashCmd)
}
```

**Step 2: Run build**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/dash.go
git commit -m "feat: update dash command for new TUI model with auto-refresh"
```

---

## Phase 5: Update cb list with repo grouping

### Task 15: Add repo grouping to cb list

**Files:**
- Modify: `cmd/list.go`

**Step 1: Rewrite list command**

```go
package cmd

import (
	"fmt"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active ClawdBay sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxClient := tmux.NewClient()
		sessions, err := tmuxClient.ListSessions()
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions. Start one with: cb start <branch-name>")
			return nil
		}

		// Group by repo
		repoSessions := make(map[string][]tmux.Session)
		var repoOrder []string

		for _, s := range sessions {
			repoName := tmuxClient.GetRepoName(s.Name)
			if _, exists := repoSessions[repoName]; !exists {
				repoOrder = append(repoOrder, repoName)
			}
			repoSessions[repoName] = append(repoSessions[repoName], s)
		}

		for _, repoName := range repoOrder {
			fmt.Println(repoName)
			for _, s := range repoSessions[repoName] {
				wins, winErr := tmuxClient.ListWindows(s.Name)
				windowCount := 0
				if winErr == nil {
					windowCount = len(wins)
				}

				// Get rolled-up status
				var statuses []tmux.Status
				if winErr == nil {
					for _, w := range wins {
						if w.IsClaudeSession() {
							statuses = append(statuses, tmuxClient.GetPaneStatus(s.Name, w.Name))
						}
					}
				}

				status := tmux.StatusDone
				if len(statuses) > 0 {
					for _, st := range statuses {
						if st == tmux.StatusWorking {
							status = tmux.StatusWorking
							break
						}
						if st == tmux.StatusIdle {
							status = tmux.StatusIdle
						}
					}
				}

				windowWord := "windows"
				if windowCount == 1 {
					windowWord = "window"
				}
				fmt.Printf("  %-30s %d %s  (%s)\n", s.Name, windowCount, windowWord, status)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Run build**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/list.go
git commit -m "feat: cb list shows sessions grouped by repo with status"
```

---

## Phase 6: Integration Test Update

### Task 16: Update integration tests for v0.2 changes

**Files:**
- Modify: `integration_test.go`

**Step 1: Update TestCLI_StartWorkflow for .worktrees/ path**

Change the worktree path calculation from:
```go
worktreePath := filepath.Join(filepath.Dir(cwd), projectName+"-"+branchName)
```
to:
```go
worktreePath := filepath.Join(cwd, ".worktrees", projectName+"-"+branchName)
```

**Step 2: Add verification that Claude window was created**

After verifying the session exists, add:

```go
// Verify Claude window was created
windowCmd := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
windowOutput, err := windowCmd.Output()
if err != nil {
	t.Errorf("failed to list windows: %v", err)
} else {
	windowNames := string(windowOutput)
	if !strings.Contains(windowNames, "claude") {
		t.Errorf("claude window not created. Windows: %s", windowNames)
	}
}
```

**Step 3: Update cleanup to use new worktree path**

The cleanup in `t.Cleanup` already uses `worktreePath` variable, so updating the calculation in Step 1 handles this.

**Step 4: Update TestCLI_Help for removed prompt command**

Already done in Task 4.

**Step 5: Run vet**

Run: `go vet ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add integration_test.go
git commit -m "test: update integration tests for .worktrees/ and claude window"
```

---

## Phase 7: Final Verification

### Task 17: Full test suite and lint

**Step 1: Run unit tests**

Run: `go test ./...`
Expected: PASS

**Step 2: Run linter**

Run: `golangci-lint run`
Expected: PASS

**Step 3: Run go vet**

Run: `go vet ./...`
Expected: PASS

**Step 4: Build binary**

Run: `go build -o cb main.go`
Expected: PASS

**Step 5: Run make verify**

Run: `make verify`
Expected: `All checks passed`

**Step 6: Commit any remaining fixes and tag**

```bash
git tag v0.2.0
```

---

## Summary

| Phase | Tasks | What it does |
|-------|-------|-------------|
| 1: Remove Dead Code | 1-4 | Delete prompt system, version cmd, simplify config |
| 2: tmux Enhancements | 5-6 | Add GetPaneWorkingDir, GetRepoName |
| 3: CLI Updates | 7-9 | Update start, claude, archive commands |
| 4: TUI Rewrite | 10-14 | New data model, navigation, rendering, dash command |
| 5: List Update | 15 | Repo-grouped text output |
| 6: Integration Tests | 16 | Update E2E tests for new paths |
| 7: Final Verification | 17 | Full test + lint pass |
