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
		model := tui.InitialModel(tmuxClient)

		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Handle selection (attach to session after TUI exits)
		if m, ok := finalModel.(tui.Model); ok && m.SelectedName != "" {
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
