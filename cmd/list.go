package cmd

import (
	"fmt"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

type listAgentDetector interface {
	DetectAgentInfo(session, window string) tmux.AgentInfo
}

func rollupStatuses(statuses []tmux.Status) tmux.Status {
	hasWaiting := false
	hasIdle := false
	for _, s := range statuses {
		switch s {
		case tmux.StatusWorking:
			return tmux.StatusWorking
		case tmux.StatusWaiting:
			hasWaiting = true
		case tmux.StatusIdle:
			hasIdle = true
		}
	}
	if hasWaiting {
		return tmux.StatusWaiting
	}
	if hasIdle {
		return tmux.StatusIdle
	}
	return tmux.StatusDone
}

func sessionStatusFromWindows(detector listAgentDetector, session string, wins []tmux.Window) tmux.Status {
	var statuses []tmux.Status
	for _, w := range wins {
		info := detector.DetectAgentInfo(session, w.Name)
		if info.Detected {
			statuses = append(statuses, info.Status)
		}
	}
	return rollupStatuses(statuses)
}

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
				status := tmux.StatusDone
				if winErr == nil {
					status = sessionStatusFromWindows(tmuxClient, s.Name, wins)
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
