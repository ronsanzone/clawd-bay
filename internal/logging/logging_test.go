package logging

import (
	"log/slog"
	"testing"
)

func TestSetup_DebugMode(t *testing.T) {
	Setup(true)

	logger := slog.Default()
	if !logger.Enabled(nil, slog.LevelDebug) {
		t.Error("debug mode should enable debug-level logging")
	}
}

func TestSetup_DefaultMode(t *testing.T) {
	Setup(false)

	logger := slog.Default()
	if logger.Enabled(nil, slog.LevelInfo) {
		t.Error("default mode should not enable info-level logging")
	}
	if !logger.Enabled(nil, slog.LevelWarn) {
		t.Error("default mode should enable warn-level logging")
	}
}
