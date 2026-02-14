package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsanzone/clawdbay/internal/tmux"
	"github.com/rsanzone/clawdbay/internal/tui"
	"github.com/spf13/cobra"
)

type dashTmuxClient interface {
	SelectWindow(session string, windowIndex int) error
	AttachOrSwitchToSession(name string, inTmux bool) error
}

func attachDashboardSelection(tmuxClient dashTmuxClient, model tui.Model, inTmux bool) error {
	if model.SelectedName == "" {
		return nil
	}

	if model.SelectedWindowIndex >= 0 {
		if err := tmuxClient.SelectWindow(model.SelectedName, model.SelectedWindowIndex); err != nil {
			return fmt.Errorf(
				"failed to select window index %d for session %s: %w",
				model.SelectedWindowIndex,
				model.SelectedName,
				err,
			)
		}
	}

	if err := tmuxClient.AttachOrSwitchToSession(model.SelectedName, inTmux); err != nil {
		return fmt.Errorf("failed to attach/switch to session %s: %w", model.SelectedName, err)
	}
	return nil
}

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
			fmt.Printf("Attaching to %s...\n", m.SelectedName)
			return attachDashboardSelection(tmuxClient, m, os.Getenv("TMUX") != "")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashCmd)
}
