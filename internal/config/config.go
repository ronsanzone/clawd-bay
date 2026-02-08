package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds ClawdBay configuration paths.
type Config struct {
	ConfigDir string
}

// New creates a Config with default paths.
func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(home, ".config", "cb")

	return &Config{
		ConfigDir: configDir,
	}, nil
}

// EnsureDirs creates the config directory if it doesn't exist.
func (c *Config) EnsureDirs() error {
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}
