package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/spf13/cobra"
)

func TestRunProjectAdd_Success(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	projectAddName = "my repo"
	projectRemoveByName = ""
	cmd, _ := testProjectCmd()

	if err := runProjectAdd(cmd, []string{repo}); err != nil {
		t.Fatalf("runProjectAdd() error = %v", err)
	}

	cfg, err := config.LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig() error = %v", err)
	}
	canonicalRepo, err := config.CanonicalPath(repo)
	if err != nil {
		t.Fatalf("CanonicalPath() error = %v", err)
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("len(cfg.Projects) = %d, want 1", len(cfg.Projects))
	}
	if cfg.Projects[0].Path != canonicalRepo {
		t.Fatalf("stored path = %q, want %q", cfg.Projects[0].Path, canonicalRepo)
	}
	if cfg.Projects[0].Name != "my repo" {
		t.Fatalf("stored name = %q, want my repo", cfg.Projects[0].Name)
	}
}

func TestRunProjectAdd_InvalidPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectAddName = ""

	cmd, _ := testProjectCmd()
	err := runProjectAdd(cmd, []string{"/path/that/does/not/exist"})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestRunProjectRemove_ByCanonicalPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	alias := filepath.Join(home, "repo-link")
	if err := os.Symlink(repo, alias); err != nil {
		t.Skipf("symlink unsupported in this environment: %v", err)
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version:  config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{{Path: repo, Name: "repo"}},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	projectRemoveByName = ""
	cmd, _ := testProjectCmd()
	if err := runProjectRemove(cmd, []string{alias}); err != nil {
		t.Fatalf("runProjectRemove() error = %v", err)
	}

	cfg, err := config.LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig() error = %v", err)
	}
	if len(cfg.Projects) != 0 {
		t.Fatalf("len(cfg.Projects) = %d, want 0", len(cfg.Projects))
	}
}

func TestRunProjectRemove_ByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo1 := filepath.Join(home, "repo-1")
	repo2 := filepath.Join(home, "repo-2")
	for _, p := range []string{repo1, repo2} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version: config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{
			{Path: repo1, Name: "keep"},
			{Path: repo2, Name: "drop"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	projectRemoveByName = "drop"
	cmd, _ := testProjectCmd()
	if err := runProjectRemove(cmd, nil); err != nil {
		t.Fatalf("runProjectRemove() error = %v", err)
	}

	cfg, err := config.LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig() error = %v", err)
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("len(cfg.Projects) = %d, want 1", len(cfg.Projects))
	}
	if cfg.Projects[0].Name != "keep" {
		t.Fatalf("remaining project name = %q, want keep", cfg.Projects[0].Name)
	}
}

func TestRunProjectRemove_ByNameErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo1 := filepath.Join(home, "repo-1")
	repo2 := filepath.Join(home, "repo-2")
	for _, p := range []string{repo1, repo2} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version: config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{
			{Path: repo1, Name: "dup"},
			{Path: repo2, Name: "dup"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	projectRemoveByName = "missing"
	cmd, _ := testProjectCmd()
	err := runProjectRemove(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "no configured project matched") {
		t.Fatalf("unexpected error for missing name: %v", err)
	}

	projectRemoveByName = "dup"
	err = runProjectRemove(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("unexpected error for ambiguous name: %v", err)
	}
}

func TestRunProjectList_EmptyAndInvalid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd, out := testProjectCmd()
	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("runProjectList() missing config error = %v", err)
	}
	if !strings.Contains(out.String(), "No project config found") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	out.Reset()
	if err := config.SaveUserConfig(config.UserConfig{Version: config.SupportedConfigVersion}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}
	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("runProjectList() empty config error = %v", err)
	}
	if !strings.Contains(out.String(), "No configured projects") {
		t.Fatalf("unexpected output: %q", out.String())
	}

	cfgDir := filepath.Join(home, ".config", "cb")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfgDir: %v", err)
	}
	invalidPath := filepath.Join(home, "missing")
	content := strings.Join([]string{
		"version = 1",
		"",
		"[[projects]]",
		fmt.Sprintf("path = %q", invalidPath),
		"name = \"broken\"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatalf("write manual config: %v", err)
	}

	out.Reset()
	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("runProjectList() invalid config runtime path error = %v", err)
	}
	if !strings.Contains(out.String(), "INVALID") {
		t.Fatalf("expected INVALID status in output, got: %q", out.String())
	}
}

func testProjectCmd() (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	return cmd, &out
}
