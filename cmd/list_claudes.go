package cmd

import (
	"fmt"

	"github.com/ronsanzone/clawd-bay/internal/tmux"
	"github.com/spf13/cobra"
)

type listClaudesOut struct {
	repoName    string
	isWorktree  bool
	windowName  string
	agentType   tmux.AgentType
	isAgent     bool
	agentStatus tmux.Status
}

func (l listClaudesOut) toString() string {
	var repoName = l.repoName
	if l.isWorktree {
		repoName = repoName + " (wt)"
	}

	if l.isAgent {
		agentStatus := "agentType: " + string(l.agentType) + " status: " + string(l.agentStatus)
		return fmt.Sprintf("%s %s (%s)\n", l.windowName, repoName, agentStatus)
	}
	return fmt.Sprintf("%s %s (DETECTED AGENT: NONE)\n", l.windowName, repoName)
}

var listClaudesCmd = &cobra.Command{
	Use:   "clist",
	Short: "List tmux windows and detected coding agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmuxClient := tmux.NewClient()
		rows, err := tmuxClient.ListSessionWindowInfo()
		if err != nil {
			return err
		}

		if len(rows) == 0 {
			fmt.Println("No active sessions. Start one with: cb start <branch-name>")
			return nil
		}

		var output []listClaudesOut
		for _, row := range rows {
			output = append(output, listClaudesOut{
				repoName:    row.RepoName,
				isWorktree:  row.Managed,
				windowName:  row.Window.Name,
				agentType:   row.AgentInfo.Type,
				isAgent:     row.AgentInfo.Detected,
				agentStatus: row.AgentInfo.Status,
			})
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
