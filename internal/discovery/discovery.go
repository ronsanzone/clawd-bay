package discovery

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/rsanzone/clawdbay/internal/tmux"
)

const mainRepoLabel = "(main repo)"

// TmuxInspector is the tmux surface needed for scoped project discovery.
type TmuxInspector interface {
	ListSessions() ([]tmux.Session, error)
	ListWindows(session string) ([]tmux.Window, error)
	GetPaneWorkingDir(session string) string
	GetSessionOption(session, key string) (string, error)
	DetectAgentInfo(session, window string) tmux.AgentInfo
}

// ProjectNode is one configured project and its worktrees.
type ProjectNode struct {
	Name         string
	Path         string
	Worktrees    []WorktreeNode
	InvalidError string
}

// WorktreeNode represents a discovered worktree path (or main repo synthetic node).
type WorktreeNode struct {
	Name       string
	Path       string
	IsMainRepo bool
	Sessions   []SessionNode
}

// SessionNode is a tmux session attached to a discovered worktree.
type SessionNode struct {
	Name    string
	Status  tmux.Status
	Windows []tmux.Window
}

// Result is the shared discovery output for dash/list.
type Result struct {
	Projects       []ProjectNode
	WindowStatuses map[string]tmux.Status
	WindowAgents   map[string]tmux.AgentType
	ConfigMissing  bool
}

// Service discovers configured project/worktree/session hierarchy.
type Service struct {
	tmuxClient TmuxInspector
	execCmd    func(name string, args ...string) ([]byte, error)
}

// NewService creates a discovery service.
func NewService(tmuxClient TmuxInspector) *Service {
	return &Service{
		tmuxClient: tmuxClient,
		execCmd: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
	}
}

// Discover builds project/worktree hierarchy and overlays tmux runtime state.
func (s *Service) Discover() (Result, error) {
	result := Result{
		WindowStatuses: make(map[string]tmux.Status),
		WindowAgents:   make(map[string]tmux.AgentType),
	}

	cfg, exists, err := config.LoadUserConfigWithMeta()
	if err != nil {
		return Result{}, err
	}
	result.ConfigMissing = !exists

	runtimeProjects := make([]runtimeProject, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		displayName := p.Name
		if displayName == "" {
			displayName = filepath.Base(p.Path)
		}

		node := ProjectNode{
			Name:      displayName,
			Path:      p.Path,
			Worktrees: []WorktreeNode{},
		}

		canonicalProjectPath, canonicalErr := config.CanonicalPath(p.Path)
		if canonicalErr != nil {
			node.InvalidError = canonicalErr.Error()
			runtimeProjects = append(runtimeProjects, runtimeProject{node: node})
			continue
		}

		node.Path = canonicalProjectPath
		worktrees, worktreeErr := s.discoverWorktrees(canonicalProjectPath)
		if worktreeErr != nil {
			node.InvalidError = worktreeErr.Error()
		}
		node.Worktrees = worktrees
		runtimeProjects = append(runtimeProjects, runtimeProject{
			canonicalPath: canonicalProjectPath,
			node:          node,
		})
	}

	sort.SliceStable(runtimeProjects, func(i, j int) bool {
		if runtimeProjects[i].node.Name != runtimeProjects[j].node.Name {
			return runtimeProjects[i].node.Name < runtimeProjects[j].node.Name
		}
		return runtimeProjects[i].node.Path < runtimeProjects[j].node.Path
	})

	if s.tmuxClient != nil {
		if err := s.overlaySessions(runtimeProjects, &result); err != nil {
			return Result{}, err
		}
	}

	result.Projects = make([]ProjectNode, 0, len(runtimeProjects))
	for _, rp := range runtimeProjects {
		for wi := range rp.node.Worktrees {
			sort.SliceStable(rp.node.Worktrees[wi].Sessions, func(i, j int) bool {
				return rp.node.Worktrees[wi].Sessions[i].Name < rp.node.Worktrees[wi].Sessions[j].Name
			})
		}
		result.Projects = append(result.Projects, rp.node)
	}

	return result, nil
}

type runtimeProject struct {
	canonicalPath string
	node          ProjectNode
}

func (s *Service) discoverWorktrees(projectPath string) ([]WorktreeNode, error) {
	main := WorktreeNode{Name: mainRepoLabel, Path: projectPath, IsMainRepo: true}

	if s.execCmd == nil {
		return []WorktreeNode{main}, nil
	}

	output, err := s.execCmd("git", "-C", projectPath, "worktree", "list", "--porcelain")
	if err != nil {
		return []WorktreeNode{main}, fmt.Errorf("failed to list worktrees for %s: %w", projectPath, err)
	}

	seen := map[string]struct{}{projectPath: {}}
	worktreesRoot := filepath.Join(projectPath, ".worktrees")

	for _, rawPath := range ParseWorktreeListPorcelain(string(output)) {
		canonicalPath, canonicalErr := config.CanonicalPath(rawPath)
		if canonicalErr != nil {
			continue
		}
		if canonicalPath == projectPath || isPathWithin(canonicalPath, worktreesRoot) {
			seen[canonicalPath] = struct{}{}
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		if path == projectPath {
			continue
		}
		paths = append(paths, path)
	}

	sort.SliceStable(paths, func(i, j int) bool {
		iRel := relativeWorktreeName(projectPath, paths[i])
		jRel := relativeWorktreeName(projectPath, paths[j])
		if iRel != jRel {
			return iRel < jRel
		}
		return paths[i] < paths[j]
	})

	result := []WorktreeNode{main}
	for _, wtPath := range paths {
		result = append(result, WorktreeNode{
			Name:       relativeWorktreeName(projectPath, wtPath),
			Path:       wtPath,
			IsMainRepo: false,
		})
	}

	return result, nil
}

func (s *Service) overlaySessions(projects []runtimeProject, result *Result) error {
	sessions, err := s.tmuxClient.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	for _, session := range sessions {
		projectIndex, worktreeIndex := s.sessionPlacement(projects, session.Name)
		if projectIndex < 0 || worktreeIndex < 0 {
			continue
		}

		windows, windowsErr := s.tmuxClient.ListWindows(session.Name)
		if windowsErr != nil {
			windows = []tmux.Window{}
		}
		sort.SliceStable(windows, func(i, j int) bool {
			return windows[i].Index < windows[j].Index
		})

		windowStatuses := make([]tmux.Status, 0, len(windows))
		for _, w := range windows {
			key := session.Name + ":" + w.Name
			info := s.tmuxClient.DetectAgentInfo(session.Name, w.Name)
			if info.Detected {
				result.WindowStatuses[key] = info.Status
				result.WindowAgents[key] = info.Type
				windowStatuses = append(windowStatuses, info.Status)
			}
		}
		projects[projectIndex].node.Worktrees[worktreeIndex].Sessions = append(
			projects[projectIndex].node.Worktrees[worktreeIndex].Sessions,
			SessionNode{
				Name:    session.Name,
				Status:  rollupStatuses(windowStatuses),
				Windows: windows,
			},
		)
	}

	return nil
}

func (s *Service) sessionPlacement(projects []runtimeProject, sessionName string) (projectIndex, worktreeIndex int) {
	projectIndex, worktreeIndex = s.sessionPlacementFromPinnedHome(projects, sessionName)
	if projectIndex >= 0 && worktreeIndex >= 0 {
		return projectIndex, worktreeIndex
	}

	// Unpinned/invalid pinned sessions are owned by pane cwd, but always grouped
	// under the project's synthetic "(main repo)" node.
	panePath := s.tmuxClient.GetPaneWorkingDir(sessionName)
	if panePath == "" {
		return -1, -1
	}
	canonicalPanePath, err := config.CanonicalPath(panePath)
	if err != nil {
		return -1, -1
	}
	projectIndex = bestProjectMatch(projects, canonicalPanePath)
	if projectIndex < 0 {
		return -1, -1
	}

	return projectIndex, mainRepoWorktreeIndex(projects[projectIndex].node.Worktrees)
}

func (s *Service) sessionPlacementFromPinnedHome(projects []runtimeProject, sessionName string) (projectIndex, worktreeIndex int) {
	homePath, err := s.tmuxClient.GetSessionOption(sessionName, tmux.SessionOptionHomePath)
	if err != nil || strings.TrimSpace(homePath) == "" {
		return -1, -1
	}

	canonicalHomePath, err := config.CanonicalPath(homePath)
	if err != nil {
		return -1, -1
	}

	projectIndex = bestProjectMatch(projects, canonicalHomePath)
	if projectIndex < 0 {
		return -1, -1
	}

	worktreeIndex = bestWorktreeMatch(projects[projectIndex].node.Worktrees, canonicalHomePath)
	if worktreeIndex < 0 {
		return -1, -1
	}
	return projectIndex, worktreeIndex
}

func bestProjectMatch(projects []runtimeProject, path string) int {
	best := -1
	bestLen := -1
	for i, p := range projects {
		if p.canonicalPath == "" {
			continue
		}
		if !isPathWithinOrEqual(path, p.canonicalPath) {
			continue
		}
		if len(p.canonicalPath) > bestLen {
			best = i
			bestLen = len(p.canonicalPath)
		}
	}
	return best
}

func bestWorktreeMatch(worktrees []WorktreeNode, path string) int {
	best := -1
	bestLen := -1
	for i, wt := range worktrees {
		if !isPathWithinOrEqual(path, wt.Path) {
			continue
		}
		if len(wt.Path) > bestLen {
			best = i
			bestLen = len(wt.Path)
		}
	}
	return best
}

func mainRepoWorktreeIndex(worktrees []WorktreeNode) int {
	for i := range worktrees {
		if worktrees[i].IsMainRepo {
			return i
		}
	}
	return -1
}

func relativeWorktreeName(projectPath, worktreePath string) string {
	rel, err := filepath.Rel(projectPath, worktreePath)
	if err != nil {
		return filepath.Base(worktreePath)
	}
	return filepath.ToSlash(rel)
}

func isPathWithin(path, root string) bool {
	if path == root {
		return false
	}
	return isPathWithinOrEqual(path, root)
}

func isPathWithinOrEqual(path, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if cleanPath == cleanRoot {
		return true
	}
	prefix := cleanRoot + string(filepath.Separator)
	return strings.HasPrefix(cleanPath, prefix)
}

func rollupStatuses(statuses []tmux.Status) tmux.Status {
	hasWaiting := false
	hasIdle := false
	for _, s := range statuses {
		switch s {
		case tmux.StatusWorking:
			return tmux.StatusWorking
		case tmux.StatusWaiting:
			hasWaiting = true
		case tmux.StatusIdle:
			hasIdle = true
		}
	}
	if hasWaiting {
		return tmux.StatusWaiting
	}
	if hasIdle {
		return tmux.StatusIdle
	}
	return tmux.StatusDone
}

// ParseWorktreeListPorcelain parses `git worktree list --porcelain` output.
func ParseWorktreeListPorcelain(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		path := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		if path == "" {
			continue
		}
		result = append(result, path)
	}
	return result
}
