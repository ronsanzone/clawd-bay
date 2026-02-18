package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ronsanzone/clawd-bay/internal/tmux"
)

type sessionResolver interface {
	ListSessions() ([]tmux.Session, error)
	GetPaneWorkingDir(session string) string
}

func resolveSessionForCWD(tmuxClient sessionResolver, cwd string) (sessionName string, worktreePath string, err error) {
	normalizedCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", fmt.Errorf("failed to normalize current directory: %w", err)
	}
	normalizedCWD = filepath.Clean(normalizedCWD)

	sessions, err := tmuxClient.ListSessions()
	if err != nil {
		return "", "", fmt.Errorf("failed to list sessions: %w", err)
	}

	type candidate struct {
		session string
		path    string
		exact   bool
	}

	var candidates []candidate
	for _, s := range sessions {
		panePath := tmuxClient.GetPaneWorkingDir(s.Name)
		if panePath == "" {
			continue
		}

		normalizedPanePath, absErr := filepath.Abs(panePath)
		if absErr != nil {
			continue
		}
		normalizedPanePath = filepath.Clean(normalizedPanePath)

		if normalizedCWD == normalizedPanePath {
			candidates = append(candidates, candidate{
				session: s.Name,
				path:    normalizedPanePath,
				exact:   true,
			})
			continue
		}

		prefix := normalizedPanePath + string(filepath.Separator)
		if strings.HasPrefix(normalizedCWD, prefix) {
			candidates = append(candidates, candidate{
				session: s.Name,
				path:    normalizedPanePath,
			})
		}
	}

	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no cb_ session found for directory %s", normalizedCWD)
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.exact && !best.exact {
			best = c
			continue
		}
		if c.exact == best.exact && len(c.path) > len(best.path) {
			best = c
		}
	}

	return best.session, best.path, nil
}
