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
  cb claude                   # Creates default session
  cb claude --name research   # Named session`,
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
		// We're inside tmux, get current session name
		output, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
		if err == nil {
			currentSession := strings.TrimSpace(string(output))
			if strings.HasPrefix(currentSession, "cb_") {
				sessionName = currentSession
			}
		}
	}

	// If not in a cb_ session, try to find one matching this directory
	if sessionName == "" {
		sessions, err := tmuxClient.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		// Worktree paths follow: <project>-<ticket>-<title>
		// Session names follow: cb_<ticket>-<title>
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
	if err := tmuxClient.CreateWindowWithShell(sessionName, windowName, "claude"); err != nil {
		return fmt.Errorf("failed to create Claude window: %w", err)
	}

	// Switch to the new window
	selectCmd := exec.Command("tmux", "select-window", "-t", sessionName+":"+windowName)
	if err := selectCmd.Run(); err != nil {
		return fmt.Errorf("failed to select window: %w", err)
	}
	return nil
}
