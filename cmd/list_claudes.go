package cmd

import (
	"fmt"

	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/spf13/cobra"
)

type listClaudesOut struct {
	repoName        string
	isWorktree      bool
	windowName      string
	isClaudeSession bool
	claudeStatus    tmux.Status
}

func (l listClaudesOut) toString() string {
	var repoName = l.repoName
	if l.isWorktree {
		repoName = repoName + " (wt)"
	}

	var claudeStatus = ""
	if l.isClaudeSession {
		claudeStatus = "claudeStatus: " + string(l.claudeStatus)
	} else {
		claudeStatus = "DETECTED AGENT: NONE"
	}

	return fmt.Sprintf("%s %s (%s)\n", l.windowName, repoName, claudeStatus)
}

var listClaudesCmd = &cobra.Command{
	Use:   "clist",
	Short: "List all active Claude Code sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxClient := tmux.NewClient()
		sessions, err := tmuxClient.ListAllSessions()
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions. Start one with: cb start <branch-name>")
			return nil
		}

		var output []listClaudesOut // Group by repo
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
			for _, s := range repoSessions[repoName] {
				wins, winErr := tmuxClient.ListWindows(s.Name)
				if winErr == nil {
					for _, w := range wins {
						out := listClaudesOut{
							repoName:        repoName,
							isWorktree:      false,
							windowName:      w.Name,
							isClaudeSession: tmuxClient.DetectAgentProcess(s.Name, w.Name),
							claudeStatus:    tmuxClient.GetPaneStatus(s.Name, w.Name),
						}
						output = append(output, out)
					}
				}
			}
		}

		for _, o := range output {
			fmt.Print(o.toString())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listClaudesCmd)
}
