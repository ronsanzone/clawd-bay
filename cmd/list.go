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
