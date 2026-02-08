package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}
	expected := filepath.Join(home, ".config", "cb")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if cfg.ConfigDir != expected {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, expected)
	}
}

func TestEnsureDirs(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()

	cfg := &Config{
		ConfigDir: filepath.Join(tmpDir, ".config", "cb"),
	}

	err := cfg.EnsureDirs()
	if err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	// Check directory exists
	if _, err := os.Stat(cfg.ConfigDir); os.IsNotExist(err) {
		t.Error("ConfigDir was not created")
	}

	// Test idempotency - calling again should succeed
	err = cfg.EnsureDirs()
	if err != nil {
		t.Fatalf("EnsureDirs() second call error = %v", err)
	}
}
