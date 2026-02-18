package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ronsanzone/clawd-bay/internal/logging"
	"github.com/spf13/cobra"
)

// Version is the current version of ClawdBay.
var Version = "0.2.0"

var debug bool

var rootCmd = &cobra.Command{
	Use:     "cb",
	Short:   "ClawdBay - A harbor for your coding-agent sessions",
	Version: Version,
	Long: `ClawdBay helps you manage and switch between coding-agent sessions in tmux.

Create isolated git worktree workflows and track session status
from an interactive dashboard.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logging.Setup(debug)
		slog.Debug("cb starting", "command", cmd.Name(), "debug", debug)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Default to dashboard
		if err := dashCmd.RunE(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
