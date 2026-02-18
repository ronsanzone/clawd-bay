package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ronsanzone/clawd-bay/internal/config"
	"github.com/ronsanzone/clawd-bay/internal/tmux"
)

type fakeTmux struct {
	sessions   []tmux.Session
	paths      map[string]string
	options    map[string]string
	optionErrs map[string]error
	windows    map[string][]tmux.Window
	infos      map[string]tmux.AgentInfo
	err        error
}

func (f fakeTmux) ListSessions() ([]tmux.Session, error) {
	return f.sessions, f.err
}

func (f fakeTmux) ListWindows(session string) ([]tmux.Window, error) {
	if wins, ok := f.windows[session]; ok {
		return wins, nil
	}
	return []tmux.Window{}, nil
}

func (f fakeTmux) GetPaneWorkingDir(session string) string {
	return f.paths[session]
}

func (f fakeTmux) GetSessionOption(session, key string) (string, error) {
	optionKey := session + "|" + key
	if err, ok := f.optionErrs[optionKey]; ok {
		return "", err
	}
	if value, ok := f.options[optionKey]; ok {
		return value, nil
	}
	return "", errors.New("missing option")
}

func (f fakeTmux) DetectAgentInfo(session, window string) tmux.AgentInfo {
	if info, ok := f.infos[session+":"+window]; ok {
		return info
	}
	return tmux.AgentInfo{Type: tmux.AgentNone, Detected: false, Status: tmux.StatusDone}
}

func TestParseWorktreeListPorcelain(t *testing.T) {
	out := `worktree /tmp/repo
HEAD abc
branch refs/heads/main

worktree /tmp/repo/.worktrees/repo-feature/add-login
HEAD def
branch refs/heads/feature/add-login`

	got := ParseWorktreeListPorcelain(out)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[1] != "/tmp/repo/.worktrees/repo-feature/add-login" {
		t.Fatalf("got[1] = %q", got[1])
	}
}

func TestDiscover_MainRepoAndLongestWorktreeMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	wtBase := filepath.Join(repo, ".worktrees", "repo-feature")
	wtNested := filepath.Join(wtBase, "add-login")
	repoScripts := filepath.Join(repo, "scripts")
	nestedPkg := filepath.Join(wtNested, "pkg")
	for _, p := range []string{repo, wtBase, wtNested, repoScripts, nestedPkg} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version: config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{
			{Path: repo, Name: "repo"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	f := fakeTmux{
		sessions: []tmux.Session{
			{Name: "cb_main"},
			{Name: "cb_nested"},
			{Name: "cb_outside"},
		},
		paths: map[string]string{
			"cb_main":    repoScripts,
			"cb_nested":  nestedPkg,
			"cb_outside": filepath.Join(home, "elsewhere"),
		},
		options: map[string]string{
			"cb_main|" + tmux.SessionOptionHomePath:   repo,
			"cb_nested|" + tmux.SessionOptionHomePath: wtNested,
		},
		windows: map[string][]tmux.Window{
			"cb_main":   {{Index: 0, Name: "claude"}},
			"cb_nested": {{Index: 0, Name: "claude"}},
		},
		infos: map[string]tmux.AgentInfo{
			"cb_main:claude":   {Type: tmux.AgentClaude, Detected: true, Status: tmux.StatusIdle},
			"cb_nested:claude": {Type: tmux.AgentCodex, Detected: true, Status: tmux.StatusWorking},
		},
	}

	svc := &Service{
		tmuxClient: f,
		execCmd: func(name string, args ...string) ([]byte, error) {
			if name != "git" {
				return nil, fmt.Errorf("unexpected command %s", name)
			}
			return []byte(strings.Join([]string{
				"worktree " + repo,
				"worktree " + wtBase,
				"worktree " + wtNested,
			}, "\n")), nil
		},
	}

	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(result.Projects))
	}

	project := result.Projects[0]
	if len(project.Worktrees) != 3 {
		t.Fatalf("len(worktrees) = %d, want 3", len(project.Worktrees))
	}
	if project.Worktrees[0].Name != mainRepoLabel {
		t.Fatalf("worktrees[0].Name = %q, want %q", project.Worktrees[0].Name, mainRepoLabel)
	}

	if len(project.Worktrees[0].Sessions) != 1 || project.Worktrees[0].Sessions[0].Name != "cb_main" {
		t.Fatalf("main repo session mapping incorrect: %+v", project.Worktrees[0].Sessions)
	}

	var nestedSessions []SessionNode
	canonicalNestedPath, err := config.CanonicalPath(wtNested)
	if err != nil {
		t.Fatalf("CanonicalPath(wtNested) error = %v", err)
	}
	for _, wt := range project.Worktrees {
		if wt.Path == canonicalNestedPath {
			nestedSessions = wt.Sessions
		}
	}
	if len(nestedSessions) != 1 || nestedSessions[0].Name != "cb_nested" {
		t.Fatalf("nested worktree match failed: %+v", nestedSessions)
	}
}

func TestDiscover_PinnedHomePlacementIgnoresPaneDrift(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	wt := filepath.Join(repo, ".worktrees", "repo-feature")
	repoScripts := filepath.Join(repo, "scripts")
	driftDir := filepath.Join(repo, "tmp", "drift")
	for _, p := range []string{repo, wt, repoScripts, driftDir} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version: config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{
			{Path: repo, Name: "repo"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	f := fakeTmux{
		sessions: []tmux.Session{{Name: "cb_stable"}},
		paths: map[string]string{
			"cb_stable": driftDir,
		},
		options: map[string]string{
			"cb_stable|" + tmux.SessionOptionHomePath: wt,
		},
		windows: map[string][]tmux.Window{
			"cb_stable": {{Index: 0, Name: "claude"}},
		},
		infos: map[string]tmux.AgentInfo{
			"cb_stable:claude": {Type: tmux.AgentClaude, Detected: true, Status: tmux.StatusIdle},
		},
	}

	svc := &Service{
		tmuxClient: f,
		execCmd: func(name string, args ...string) ([]byte, error) {
			return []byte(strings.Join([]string{
				"worktree " + repo,
				"worktree " + wt,
			}, "\n")), nil
		},
	}

	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.Projects) != 1 || len(result.Projects[0].Worktrees) != 2 {
		t.Fatalf("unexpected discovery tree: %+v", result.Projects)
	}

	mainSessions := result.Projects[0].Worktrees[0].Sessions
	if len(mainSessions) != 0 {
		t.Fatalf("main repo sessions = %+v, want empty", mainSessions)
	}
	worktreeSessions := result.Projects[0].Worktrees[1].Sessions
	if len(worktreeSessions) != 1 || worktreeSessions[0].Name != "cb_stable" {
		t.Fatalf("pinned session placement mismatch: %+v", worktreeSessions)
	}
}

func TestDiscover_UnpinnedSessionFallsBackToMainRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	wt := filepath.Join(repo, ".worktrees", "repo-feature")
	repoScripts := filepath.Join(repo, "scripts")
	wtPkg := filepath.Join(wt, "pkg")
	for _, p := range []string{repo, wt, repoScripts, wtPkg} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	if err := config.SaveUserConfig(config.UserConfig{
		Version: config.SupportedConfigVersion,
		Projects: []config.ProjectConfig{
			{Path: repo, Name: "repo"},
		},
	}); err != nil {
		t.Fatalf("SaveUserConfig() error = %v", err)
	}

	f := fakeTmux{
		sessions: []tmux.Session{{Name: "cb_untagged"}},
		paths: map[string]string{
			"cb_untagged": wtPkg,
		},
		windows: map[string][]tmux.Window{
			"cb_untagged": {{Index: 0, Name: "claude"}},
		},
		infos: map[string]tmux.AgentInfo{
			"cb_untagged:claude": {Type: tmux.AgentCodex, Detected: true, Status: tmux.StatusWorking},
		},
		optionErrs: map[string]error{
			"cb_untagged|" + tmux.SessionOptionHomePath: errors.New("missing option"),
		},
	}

	svc := &Service{
		tmuxClient: f,
		execCmd: func(name string, args ...string) ([]byte, error) {
			return []byte(strings.Join([]string{
				"worktree " + repo,
				"worktree " + wt,
			}, "\n")), nil
		},
	}

	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(result.Projects))
	}
	project := result.Projects[0]
	if len(project.Worktrees[0].Sessions) != 1 || project.Worktrees[0].Sessions[0].Name != "cb_untagged" {
		t.Fatalf("main repo placement mismatch: %+v", project.Worktrees[0].Sessions)
	}
	if len(project.Worktrees[1].Sessions) != 0 {
		t.Fatalf("worktree sessions = %+v, want empty", project.Worktrees[1].Sessions)
	}
}

func TestDiscover_InvalidConfiguredProjectIsWarningOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "cb")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg dir: %v", err)
	}
	missingPath := filepath.Join(home, "does-not-exist")
	content := "version = 1\n\n[[projects]]\npath = " + fmt.Sprintf("%q", missingPath) + "\nname = \"ghost\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	svc := &Service{tmuxClient: fakeTmux{}, execCmd: nil}
	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(result.Projects))
	}
	if result.Projects[0].InvalidError == "" {
		t.Fatal("expected invalid project warning, got none")
	}
}

func TestDiscover_DeterministicOrdering(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoB := filepath.Join(home, "repo-b")
	repoA := filepath.Join(home, "repo-a")
	for _, p := range []string{repoA, repoB} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	cfgDir := filepath.Join(home, ".config", "cb")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg dir: %v", err)
	}
	manual := strings.Join([]string{
		"version = 1",
		"",
		"[[projects]]",
		"path = " + fmt.Sprintf("%q", repoB),
		"name = \"zeta\"",
		"",
		"[[projects]]",
		"path = " + fmt.Sprintf("%q", repoA),
		"name = \"alpha\"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(manual), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	svc := &Service{
		tmuxClient: fakeTmux{},
		execCmd: func(name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		},
	}

	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.Projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(result.Projects))
	}
	if result.Projects[0].Name != "alpha" || result.Projects[1].Name != "zeta" {
		t.Fatalf("unexpected project order: %q, %q", result.Projects[0].Name, result.Projects[1].Name)
	}
}

func TestDiscover_ConfigMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	svc := &Service{tmuxClient: fakeTmux{}}
	result, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if !result.ConfigMissing {
		t.Fatal("ConfigMissing = false, want true")
	}
}
