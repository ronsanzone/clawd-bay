package tui

import (
	"testing"
)

func TestKanagawaClawThemeHasAllColors(t *testing.T) {
	theme := KanagawaClaw

	colors := map[string]string{
		"Bg":        string(theme.Bg),
		"BgDark":    string(theme.BgDark),
		"BgLight":   string(theme.BgLight),
		"Border":    string(theme.Border),
		"Fg":        string(theme.Fg),
		"FgDim":     string(theme.FgDim),
		"FgMuted":   string(theme.FgMuted),
		"Accent":    string(theme.Accent),
		"Highlight": string(theme.Highlight),
		"Info":      string(theme.Info),
		"Working":   string(theme.Working),
		"Idle":      string(theme.Idle),
		"Done":      string(theme.Done),
	}

	for name, val := range colors {
		if val == "" {
			t.Errorf("theme color %s is empty", name)
		}
	}
}

func TestBuildStylesFromTheme(t *testing.T) {
	styles := NewStyles(KanagawaClaw)

	if styles.Title.Render("test") == "" {
		t.Error("Title style renders empty")
	}
	if styles.Repo.Render("test") == "" {
		t.Error("Repo style renders empty")
	}
	if styles.Session.Render("test") == "" {
		t.Error("Session style renders empty")
	}
	if styles.Window.Render("test") == "" {
		t.Error("Window style renders empty")
	}
	if styles.StatusWorking.Render("test") == "" {
		t.Error("StatusWorking style renders empty")
	}
}
