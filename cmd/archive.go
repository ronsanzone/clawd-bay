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
			if !strings.HasPrefix(sessionName, "cb_") {
				sessionName = "cb_" + sessionName
			}

			// Try to find worktree path from session's pane
			tmuxClient := tmux.NewClient()
			paneDir := tmuxClient.GetPaneWorkingDir(sessionName)
			if paneDir != "" {
				worktreePath = paneDir
			}
		} else {
			// Detect session from current directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			worktreePath = cwd

			tmuxClient := tmux.NewClient()
			sessions, err := tmuxClient.ListSessions()
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			dirName := filepath.Base(cwd)
			for _, s := range sessions {
				sessionSuffix := strings.TrimPrefix(s.Name, "cb_")
				if strings.Contains(dirName, sessionSuffix) {
					sessionName = s.Name
					break
				}
			}

			if sessionName == "" {
				return fmt.Errorf("no cb_ session found for directory %s", dirName)
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
		_ = killCmd.Run() // Ignore error if session doesn't exist

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
