package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rsanzone/clawdbay/internal/config"
	"github.com/spf13/cobra"
)

var projectAddName string
var projectRemoveByName string

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage configured projects",
}

var projectAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a configured project",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectAdd,
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a configured project",
	Args: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(projectRemoveByName) != "" {
			if len(args) > 0 {
				return fmt.Errorf("path argument is not allowed with --name")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("expected exactly 1 path argument, or use --name")
		}
		return nil
	},
	RunE: runProjectRemove,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured projects",
	Args:  cobra.NoArgs,
	RunE:  runProjectList,
}

func init() {
	projectAddCmd.Flags().StringVar(&projectAddName, "name", "", "optional project display name")
	projectRemoveCmd.Flags().StringVar(&projectRemoveByName, "name", "", "remove by exact configured project name")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectListCmd)
	rootCmd.AddCommand(projectCmd)
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	canonicalPath, err := config.CanonicalPath(args[0])
	if err != nil {
		return fmt.Errorf("failed to canonicalize project path %q: %w", args[0], err)
	}

	name := strings.TrimSpace(projectAddName)
	if projectAddName != "" && name == "" {
		return fmt.Errorf("--name must be non-empty when provided")
	}

	cfg, err := config.LoadUserConfig()
	if err != nil {
		return err
	}

	for _, p := range cfg.Projects {
		if p.Path == canonicalPath {
			return fmt.Errorf("project already configured: %s", canonicalPath)
		}
	}

	cfg.Projects = append(cfg.Projects, config.ProjectConfig{Path: canonicalPath, Name: name})
	if err := config.SaveUserConfig(cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added project: %s\n", canonicalPath)
	return nil
}

func runProjectRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return err
	}

	if strings.TrimSpace(projectRemoveByName) != "" {
		return removeProjectByName(cmd, cfg, strings.TrimSpace(projectRemoveByName))
	}
	return removeProjectByPath(cmd, cfg, args[0])
}

func removeProjectByPath(cmd *cobra.Command, cfg config.UserConfig, inputPath string) error {
	canonicalInputPath, err := config.CanonicalPath(inputPath)
	if err != nil {
		return fmt.Errorf("failed to canonicalize removal path %q: %w", inputPath, err)
	}

	filtered := make([]config.ProjectConfig, 0, len(cfg.Projects))
	removedCount := 0
	for _, p := range cfg.Projects {
		canonicalConfiguredPath, canonicalErr := config.CanonicalPath(p.Path)
		if canonicalErr != nil {
			filtered = append(filtered, p)
			continue
		}
		if canonicalConfiguredPath == canonicalInputPath {
			removedCount++
			continue
		}
		filtered = append(filtered, p)
	}

	if removedCount == 0 {
		return fmt.Errorf("no configured project matched canonical path %s", canonicalInputPath)
	}

	cfg.Projects = filtered
	if err := config.SaveUserConfig(cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed project: %s\n", canonicalInputPath)
	return nil
}

func removeProjectByName(cmd *cobra.Command, cfg config.UserConfig, name string) error {
	matchIndexes := make([]int, 0, 1)
	for i, p := range cfg.Projects {
		if p.Name == name {
			matchIndexes = append(matchIndexes, i)
		}
	}

	if len(matchIndexes) == 0 {
		return fmt.Errorf("no configured project matched name %q", name)
	}
	if len(matchIndexes) > 1 {
		return fmt.Errorf("project name %q is ambiguous; use canonical path removal", name)
	}

	idx := matchIndexes[0]
	removedPath := cfg.Projects[idx].Path
	cfg.Projects = append(cfg.Projects[:idx], cfg.Projects[idx+1:]...)
	if err := config.SaveUserConfig(cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed project %q: %s\n", name, removedPath)
	return nil
}

func runProjectList(cmd *cobra.Command, _ []string) error {
	cfg, exists, err := config.LoadUserConfigWithMeta()
	if err != nil {
		return err
	}

	if !exists {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No project config found. Add one with: cb project add <path>")
		return nil
	}

	if len(cfg.Projects) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No configured projects. Add one with: cb project add <path>")
		return nil
	}

	for _, p := range cfg.Projects {
		displayName := p.Name
		if displayName == "" {
			displayName = filepath.Base(p.Path)
		}

		status := "OK"
		canonicalPath, canonicalErr := config.CanonicalPath(p.Path)
		if canonicalErr != nil {
			status = "INVALID: " + canonicalErr.Error()
		} else if canonicalPath != filepath.Clean(p.Path) {
			status = fmt.Sprintf("INVALID: configured path is not canonical (canonical=%s)", canonicalPath)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n  path: %s\n  status: %s\n", displayName, p.Path, status)
	}

	return nil
}
