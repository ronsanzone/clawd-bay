//go:build integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLI_Version(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "--version")
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

	expected := []string{"start", "claude", "list", "archive", "dash"}
	for _, sub := range expected {
		if !strings.Contains(string(output), sub) {
			t.Errorf("help missing subcommand: %s", sub)
		}
	}
}

// buildTestBinary builds the cb binary to a unique temp location for testing.
// Returns the path to the binary. Caller should defer removal.
func buildTestBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cb")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, output)
	}
	return binaryPath
}


// TestCLI_StartWorkflow tests the full cb start workflow end-to-end.
// This test:
// 1. Creates a worktree and tmux session via `cb start`
// 2. Verifies the tmux session exists with correct naming (cb_<branch>)
// 3. Verifies the worktree directory was created
// 4. Cleans up all resources
func TestCLI_StartWorkflow(t *testing.T) {
	// Skip if tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping integration test")
	}

	binaryPath := buildTestBinary(t)

	// Use a unique branch name to avoid conflicts
	branchName := fmt.Sprintf("e2e-test-%d", time.Now().UnixNano())
	sessionName := "cb_" + branchName

	// Get current directory info for worktree path calculation
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectName := filepath.Base(cwd)
	worktreePath := filepath.Join(cwd, ".worktrees", projectName+"-"+branchName)

	// Register cleanup to run even if test fails
	t.Cleanup(func() {
		// Kill tmux session (ignore errors - may not exist)
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()

		// Remove worktree (ignore errors - may not exist)
		exec.Command("git", "worktree", "remove", "--force", worktreePath).Run()

		// Delete the test branch (ignore errors - may not exist)
		exec.Command("git", "branch", "-D", branchName).Run()
	})

	// Run cb start with --detach to avoid interactive attach/switch
	// (tmux attach/switch require TTY which isn't available in go test)
	startCmd := exec.Command(binaryPath, "start", "--detach", branchName)
	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cb start failed: %v\nOutput: %s", err, startOutput)
	}

	// Verify tmux session was created with correct name
	hasSessionCmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if err := hasSessionCmd.Run(); err != nil {
		t.Errorf("tmux session %q was not created", sessionName)

		// Debug: list all sessions
		listCmd := exec.Command("tmux", "list-sessions")
		if listOutput, err := listCmd.Output(); err == nil {
			t.Logf("Available sessions:\n%s", listOutput)
		}
	}

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

	// Verify worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", worktreePath)
	}

	// Verify worktree is on correct branch
	branchCmd := exec.Command("git", "-C", worktreePath, "branch", "--show-current")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		t.Errorf("failed to get branch in worktree: %v", err)
	} else if strings.TrimSpace(string(branchOutput)) != branchName {
		t.Errorf("worktree on wrong branch: got %q, want %q",
			strings.TrimSpace(string(branchOutput)), branchName)
	}

	// Regression test: Verify session name doesn't contain colons
	// (colons break tmux targeting since : is the session:window separator)
	listCmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	listOutput, err := listCmd.Output()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for _, line := range strings.Split(string(listOutput), "\n") {
		if strings.HasPrefix(line, "cb") && strings.Contains(line, ":") {
			t.Errorf("session name contains colon (breaks tmux targeting): %q", line)
		}
	}
}

