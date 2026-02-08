package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines all colors for the TUI.
type Theme struct {
	Bg      lipgloss.Color
	BgDark  lipgloss.Color
	BgLight lipgloss.Color
	Border  lipgloss.Color

	Fg      lipgloss.Color
	FgDim   lipgloss.Color
	FgMuted lipgloss.Color

	Accent    lipgloss.Color
	Highlight lipgloss.Color
	Info      lipgloss.Color

	Working lipgloss.Color
	Waiting lipgloss.Color
	Idle    lipgloss.Color
	Done    lipgloss.Color
}

// KanagawaClaw is the default theme inspired by Kanagawa.nvim.
var KanagawaClaw = Theme{
	Bg:      lipgloss.Color("#1F1F28"),
	BgDark:  lipgloss.Color("#16161D"),
	BgLight: lipgloss.Color("#2A2A37"),
	Border:  lipgloss.Color("#363646"),

	Fg:      lipgloss.Color("#DCD7BA"),
	FgDim:   lipgloss.Color("#C8C093"),
	FgMuted: lipgloss.Color("#727169"),

	Accent:    lipgloss.Color("#957FB8"),
	Highlight: lipgloss.Color("#D27E99"),
	Info:      lipgloss.Color("#7E9CD8"),

	Working: lipgloss.Color("#98BB6C"),
	Waiting: lipgloss.Color("#E6C384"),
	Idle:    lipgloss.Color("#FF9E3B"),
	Done:    lipgloss.Color("#54546D"),
}

// Styles holds all pre-built lipgloss styles derived from a Theme.
type Styles struct {
	// Frame
	Title lipgloss.Style
	Frame lipgloss.Style

	// Tree nodes
	Repo     lipgloss.Style
	Session  lipgloss.Style
	Window   lipgloss.Style
	Selected lipgloss.Style

	// Status badges
	StatusWorking lipgloss.Style
	StatusWaiting lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusDone    lipgloss.Style

	// UI chrome
	Footer    lipgloss.Style
	StatusBar lipgloss.Style
}

// NewStyles builds all styles from the given theme.
func NewStyles(t Theme) Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent),

		Frame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),

		Repo: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent),

		Session: lipgloss.NewStyle().
			Foreground(t.FgDim),

		Window: lipgloss.NewStyle().
			Foreground(t.FgMuted),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Highlight).
			Background(t.BgLight),

		StatusWorking: lipgloss.NewStyle().
			Foreground(t.Working),

		StatusWaiting: lipgloss.NewStyle().
			Foreground(t.Waiting),

		StatusIdle: lipgloss.NewStyle().
			Foreground(t.Idle),

		StatusDone: lipgloss.NewStyle().
			Foreground(t.Done),

		Footer: lipgloss.NewStyle().
			Foreground(t.FgMuted),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.FgMuted),
	}
}
