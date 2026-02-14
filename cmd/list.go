package cmd

import (
	"fmt"

	"github.com/rsanzone/clawdbay/internal/discovery"
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
		result, err := discovery.NewService(tmuxClient).Discover()
		if err != nil {
			return err
		}

		if result.ConfigMissing {
			fmt.Println("No project config found. Add one with: cb project add <path>")
			return nil
		}

		if len(result.Projects) == 0 {
			fmt.Println("No configured projects. Add one with: cb project add <path>")
			return nil
		}

		for _, project := range result.Projects {
			fmt.Println(project.Name)
			if project.InvalidError != "" {
				fmt.Printf("  [INVALID] %s\n", project.InvalidError)
			}

			for _, wt := range project.Worktrees {
				fmt.Printf("  %s\n", wt.Name)
				if len(wt.Sessions) == 0 {
					fmt.Println("    (no active session)")
					continue
				}

				for _, s := range wt.Sessions {
					windowCount := len(s.Windows)
					windowWord := "windows"
					if windowCount == 1 {
						windowWord = "window"
					}
					fmt.Printf("    %-30s %d %s  (%s)\n", s.Name, windowCount, windowWord, s.Status)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
