package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "Feature-Branch", "feature-branch"},
		{"spaces to dashes", "my feature branch", "my-feature-branch"},
		{"special chars removed", "feat@#$%ure!", "feature"},
		{"slashes preserved", "feature/add-login", "feature/add-login"},
		{"underscores preserved", "feat_123_auth", "feat_123_auth"},
		{"multiple dashes collapsed", "feat---branch", "feat-branch"},
		{"leading trailing dashes trimmed", "-branch-", "branch"},
		{"digits preserved", "proj-123-auth", "proj-123-auth"},
		{"empty after sanitize", "@#$%", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureGitignoreEntry(t *testing.T) {
	t.Run("creates gitignore if missing", func(t *testing.T) {
		dir := t.TempDir()
		ensureGitignoreEntry(dir, ".worktrees/")

		content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if err != nil {
			t.Fatalf("failed to read .gitignore: %v", err)
		}
		if string(content) != ".worktrees/\n" {
			t.Errorf("got %q, want %q", content, ".worktrees/\n")
		}
	})

	t.Run("appends to existing gitignore", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644); err != nil {
			t.Fatalf("failed to seed .gitignore: %v", err)
		}

		ensureGitignoreEntry(dir, ".worktrees/")

		content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if err != nil {
			t.Fatalf("failed to read .gitignore: %v", err)
		}
		want := "node_modules/\n.worktrees/\n"
		if string(content) != want {
			t.Errorf("got %q, want %q", content, want)
		}
	})

	t.Run("does not duplicate existing entry", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n"), 0644); err != nil {
			t.Fatalf("failed to seed .gitignore: %v", err)
		}

		ensureGitignoreEntry(dir, ".worktrees/")

		content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if err != nil {
			t.Fatalf("failed to read .gitignore: %v", err)
		}
		if string(content) != ".worktrees/\n" {
			t.Errorf("got %q, want duplicate-free %q", content, ".worktrees/\n")
		}
	})

	t.Run("adds newline before entry if file lacks trailing newline", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/"), 0644); err != nil {
			t.Fatalf("failed to seed .gitignore: %v", err)
		}

		ensureGitignoreEntry(dir, ".worktrees/")

		content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if err != nil {
			t.Fatalf("failed to read .gitignore: %v", err)
		}
		want := "node_modules/\n.worktrees/\n"
		if string(content) != want {
			t.Errorf("got %q, want %q", content, want)
		}
	})
}

func TestRunStart_RejectsEmptySanitizedBranch(t *testing.T) {
	err := runStart(startCmd, []string{"@#$%"})
	if err == nil {
		t.Fatal("expected error for empty sanitized branch, got nil")
	}
	if !strings.Contains(err.Error(), "invalid after sanitization") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "invalid after sanitization")
	}
}
