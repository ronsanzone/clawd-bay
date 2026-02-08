package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const debugLogPath = "/tmp/cb-debug.log"

// Setup configures the default slog logger.
// When debug is true, logs at Debug level to /tmp/cb-debug.log.
// Otherwise defaults to Warn level on stderr.
func Setup(debug bool) {
	level := slog.LevelWarn
	output := os.Stderr

	if debug {
		level = slog.LevelDebug
		f, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open debug log %s: %v\n", debugLogPath, err)
		} else {
			output = f
			fmt.Fprintf(os.Stderr, "debug logs: %s\n", filepath.Clean(debugLogPath))
		}
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
