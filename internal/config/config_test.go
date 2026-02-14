package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	tmpDir := t.TempDir()
	cfg := &Config{ConfigDir: filepath.Join(tmpDir, ".config", "cb")}

	if err := cfg.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	if _, err := os.Stat(cfg.ConfigDir); os.IsNotExist(err) {
		t.Error("ConfigDir was not created")
	}
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() second call error = %v", err)
	}
}

func TestSaveAndLoadUserConfig_RoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoA := filepath.Join(home, "code", "repo-a")
	repoB := filepath.Join(home, "code", "repo-b")
	if err := os.MkdirAll(repoA, 0755); err != nil {
		t.Fatalf("mkdir repoA: %v", err)
	}
	if err := os.MkdirAll(repoB, 0755); err != nil {
		t.Fatalf("mkdir repoB: %v", err)
	}

	input := UserConfig{
		Version: SupportedConfigVersion,
		Projects: []ProjectConfig{
			{Path: repoB},
			{Path: repoA, Name: "alpha"},
		},
	}

	if err := SaveUserConfig(input); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	loaded, exists, err := LoadUserConfigWithMeta()
	if err != nil {
		t.Fatalf("LoadUserConfigWithMeta() error = %v", err)
	}
	if !exists {
		t.Fatal("expected config file to exist")
	}
	if loaded.Version != SupportedConfigVersion {
		t.Fatalf("loaded.Version = %d, want %d", loaded.Version, SupportedConfigVersion)
	}
	if len(loaded.Projects) != 2 {
		t.Fatalf("len(loaded.Projects) = %d, want 2", len(loaded.Projects))
	}
	if loaded.Projects[0].Name != "alpha" {
		t.Fatalf("projects[0].Name = %q, want alpha", loaded.Projects[0].Name)
	}

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	info, err := os.Stat(cfg.ConfigFilePath())
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config mode = %#o, want %#o", info.Mode().Perm(), os.FileMode(0600))
	}
}

func TestLoadUserConfig_MissingFileIsValid(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, exists, err := LoadUserConfigWithMeta()
	if err != nil {
		t.Fatalf("LoadUserConfigWithMeta() error = %v", err)
	}
	if exists {
		t.Fatal("exists = true, want false")
	}
	if cfg.Version != SupportedConfigVersion {
		t.Fatalf("cfg.Version = %d, want %d", cfg.Version, SupportedConfigVersion)
	}
	if len(cfg.Projects) != 0 {
		t.Fatalf("len(cfg.Projects) = %d, want 0", len(cfg.Projects))
	}
}

func TestSaveUserConfig_RejectsDuplicateCanonicalPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "code", "repo")
	if err := os.MkdirAll(filepath.Join(home, "code"), 0755); err != nil {
		t.Fatalf("mkdir code dir: %v", err)
	}
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	alias := filepath.Join(home, "repo-link")
	if err := os.Symlink(repo, alias); err != nil {
		t.Skipf("symlink unsupported in test environment: %v", err)
	}

	err := SaveUserConfig(UserConfig{
		Version: SupportedConfigVersion,
		Projects: []ProjectConfig{
			{Path: repo},
			{Path: alias},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate canonical path error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate canonical project path") {
		t.Fatalf("error = %q, want duplicate canonical path error", err)
	}
}

func TestSaveUserConfig_CanonicalizesSymlinkPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "code", "repo")
	if err := os.MkdirAll(filepath.Join(home, "code"), 0755); err != nil {
		t.Fatalf("mkdir code dir: %v", err)
	}
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	alias := filepath.Join(home, "repo-link")
	if err := os.Symlink(repo, alias); err != nil {
		t.Skipf("symlink unsupported in test environment: %v", err)
	}

	if err := SaveUserConfig(UserConfig{
		Version:  SupportedConfigVersion,
		Projects: []ProjectConfig{{Path: alias}},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	loaded, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig() error = %v", err)
	}
	canonicalRepo, err := CanonicalPath(repo)
	if err != nil {
		t.Fatalf("CanonicalPath(repo) error = %v", err)
	}
	if len(loaded.Projects) != 1 {
		t.Fatalf("len(loaded.Projects) = %d, want 1", len(loaded.Projects))
	}
	if loaded.Projects[0].Path != canonicalRepo {
		t.Fatalf("stored path = %q, want canonical %q", loaded.Projects[0].Path, canonicalRepo)
	}
}

func TestLoadUserConfig_UnsupportedVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "cb")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfgDir: %v", err)
	}
	content := "version = 99\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, _, err := LoadUserConfigWithMeta()
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("error = %q, want unsupported version", err)
	}
}

func TestSaveUserConfig_RejectsEmptyName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	err := SaveUserConfig(UserConfig{
		Version:  SupportedConfigVersion,
		Projects: []ProjectConfig{{Path: repo, Name: "   "}},
	})
	if err == nil {
		t.Fatal("expected empty name validation error")
	}
}

func TestSaveUserConfig_DeterministicOrdering(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	alphaPath := filepath.Join(home, "z")
	betaPath := filepath.Join(home, "a")
	if err := os.MkdirAll(alphaPath, 0755); err != nil {
		t.Fatalf("mkdir alphaPath: %v", err)
	}
	if err := os.MkdirAll(betaPath, 0755); err != nil {
		t.Fatalf("mkdir betaPath: %v", err)
	}

	if err := SaveUserConfig(UserConfig{
		Version: SupportedConfigVersion,
		Projects: []ProjectConfig{
			{Path: alphaPath},
			{Path: betaPath, Name: "aa"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	content, err := os.ReadFile(cfg.ConfigFilePath())
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	body := string(content)
	firstPathIdx := strings.Index(body, betaPath)
	secondPathIdx := strings.Index(body, alphaPath)
	if firstPathIdx == -1 || secondPathIdx == -1 {
		t.Fatalf("paths missing from file content: %q", body)
	}
	if firstPathIdx > secondPathIdx {
		t.Fatalf("projects not ordered deterministically by display name: %q", body)
	}
}

func TestCanonicalPath(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not stable on windows CI environments")
	}

	alias := filepath.Join(base, "repo-link")
	if err := os.Symlink(repo, alias); err != nil {
		t.Skipf("symlink unsupported in test environment: %v", err)
	}

	got, err := CanonicalPath(alias)
	if err != nil {
		t.Fatalf("CanonicalPath() error = %v", err)
	}
	want, err := CanonicalPath(repo)
	if err != nil {
		t.Fatalf("CanonicalPath(repo) error = %v", err)
	}
	if got != want {
		t.Fatalf("CanonicalPath() = %q, want %q", got, want)
	}
}
