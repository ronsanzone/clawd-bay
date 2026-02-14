package cmd

import (
	"path/filepath"
	"testing"

	"github.com/rsanzone/clawdbay/internal/tmux"
)

type fakeSessionResolver struct {
	sessions []tmux.Session
	paths    map[string]string
	err      error
}

func (f fakeSessionResolver) ListSessions() ([]tmux.Session, error) {
	return f.sessions, f.err
}

func (f fakeSessionResolver) GetPaneWorkingDir(session string) string {
	return f.paths[session]
}

func TestResolveSessionForCWD_ExactMatchPreferred(t *testing.T) {
	wd := t.TempDir()
	exact := filepath.Join(wd, "project", ".worktrees", "project-feat")
	parent := filepath.Dir(exact)
	cwd := exact

	resolver := fakeSessionResolver{
		sessions: []tmux.Session{{Name: "cb_parent"}, {Name: "cb_exact"}},
		paths: map[string]string{
			"cb_parent": parent,
			"cb_exact":  exact,
		},
	}

	session, worktreePath, err := resolveSessionForCWD(resolver, cwd)
	if err != nil {
		t.Fatalf("resolveSessionForCWD() error = %v", err)
	}
	if session != "cb_exact" {
		t.Fatalf("session = %q, want %q", session, "cb_exact")
	}
	if worktreePath != filepath.Clean(exact) {
		t.Fatalf("worktreePath = %q, want %q", worktreePath, filepath.Clean(exact))
	}
}

func TestResolveSessionForCWD_LongestPrefixWins(t *testing.T) {
	wd := t.TempDir()
	short := filepath.Join(wd, "project")
	long := filepath.Join(wd, "project", ".worktrees", "project-feat")
	cwd := filepath.Join(long, "nested", "dir")

	resolver := fakeSessionResolver{
		sessions: []tmux.Session{{Name: "cb_short"}, {Name: "cb_long"}},
		paths: map[string]string{
			"cb_short": short,
			"cb_long":  long,
		},
	}

	session, worktreePath, err := resolveSessionForCWD(resolver, cwd)
	if err != nil {
		t.Fatalf("resolveSessionForCWD() error = %v", err)
	}
	if session != "cb_long" {
		t.Fatalf("session = %q, want %q", session, "cb_long")
	}
	if worktreePath != filepath.Clean(long) {
		t.Fatalf("worktreePath = %q, want %q", worktreePath, filepath.Clean(long))
	}
}

func TestResolveSessionForCWD_NoMatch(t *testing.T) {
	wd := t.TempDir()
	cwd := filepath.Join(wd, "project", ".worktrees", "project-feat")
	other := filepath.Join(wd, "different", ".worktrees", "different-feat")

	resolver := fakeSessionResolver{
		sessions: []tmux.Session{{Name: "cb_other"}},
		paths: map[string]string{
			"cb_other": other,
		},
	}

	_, _, err := resolveSessionForCWD(resolver, cwd)
	if err == nil {
		t.Fatal("resolveSessionForCWD() expected error, got nil")
	}
}

func TestResolveSessionForCWD_SlashBranchPath(t *testing.T) {
	wd := t.TempDir()
	panePath := filepath.Join(wd, "repo", ".worktrees", "repo-feature", "add-login")
	cwd := filepath.Join(panePath, "internal", "cmd")

	resolver := fakeSessionResolver{
		sessions: []tmux.Session{{Name: "cb_feature/add-login"}},
		paths: map[string]string{
			"cb_feature/add-login": panePath,
		},
	}

	session, worktreePath, err := resolveSessionForCWD(resolver, cwd)
	if err != nil {
		t.Fatalf("resolveSessionForCWD() error = %v", err)
	}
	if session != "cb_feature/add-login" {
		t.Fatalf("session = %q, want %q", session, "cb_feature/add-login")
	}
	if worktreePath != filepath.Clean(panePath) {
		t.Fatalf("worktreePath = %q, want %q", worktreePath, filepath.Clean(panePath))
	}
}
