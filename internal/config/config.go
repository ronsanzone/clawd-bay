package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	// SupportedConfigVersion is the only config version supported by this binary.
	SupportedConfigVersion = 1
	configFileName         = "config.toml"
)

// Config holds ClawdBay configuration paths.
type Config struct {
	ConfigDir string
}

// UserConfig is the persisted configuration file schema.
type UserConfig struct {
	Version  int             `toml:"version"`
	Projects []ProjectConfig `toml:"projects"`
}

// ProjectConfig defines one configured project root.
type ProjectConfig struct {
	Path string `toml:"path"`
	Name string `toml:"name,omitempty"`
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

// ConfigFilePath returns ~/.config/cb/config.toml.
func (c *Config) ConfigFilePath() string {
	return filepath.Join(c.ConfigDir, configFileName)
}

// CanonicalPath resolves a path for all matching/comparison operations.
func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to make absolute path %q: %w", path, err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks for %q: %w", abs, err)
	}

	return filepath.Clean(resolved), nil
}

// LoadUserConfig loads config.toml. Missing file returns empty valid config.
func LoadUserConfig() (UserConfig, error) {
	cfg, _, err := LoadUserConfigWithMeta()
	return cfg, err
}

// LoadUserConfigWithMeta loads config.toml and indicates whether file existed.
func LoadUserConfigWithMeta() (cfg UserConfig, exists bool, err error) {
	c, err := New()
	if err != nil {
		return UserConfig{}, false, err
	}

	path := c.ConfigFilePath()
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return UserConfig{Version: SupportedConfigVersion, Projects: []ProjectConfig{}}, false, nil
		}
		return UserConfig{}, false, fmt.Errorf("failed to read config file %s: %w", path, readErr)
	}

	if len(bytes.TrimSpace(content)) == 0 {
		return UserConfig{Version: SupportedConfigVersion, Projects: []ProjectConfig{}}, true, nil
	}

	parsed, parseErr := parseUserConfigTOML(content)
	if parseErr != nil {
		return UserConfig{}, true, fmt.Errorf("failed to parse config file %s: %w", path, parseErr)
	}

	if validateErr := validateLoadedConfig(parsed); validateErr != nil {
		return UserConfig{}, true, fmt.Errorf("invalid config file %s: %w", path, validateErr)
	}

	return parsed, true, nil
}

// SaveUserConfig validates, canonicalizes, and atomically persists config.toml.
func SaveUserConfig(cfg UserConfig) error {
	normalized, err := normalizeForSave(cfg)
	if err != nil {
		return err
	}

	c, err := New()
	if err != nil {
		return err
	}

	if err := c.EnsureDirs(); err != nil {
		return err
	}

	content := renderUserConfigTOML(normalized)
	path := c.ConfigFilePath()
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, "config-*.toml")
	if err != nil {
		return fmt.Errorf("failed to create temp config file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set mode on temp config file: %w", err)
	}

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to atomically replace config file %s: %w", path, err)
	}

	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set mode on config file %s: %w", path, err)
	}

	return nil
}

func validateLoadedConfig(cfg UserConfig) error {
	if cfg.Version != SupportedConfigVersion {
		return fmt.Errorf("unsupported version %d (supported: %d)", cfg.Version, SupportedConfigVersion)
	}

	for i, p := range cfg.Projects {
		if strings.TrimSpace(p.Path) == "" {
			return fmt.Errorf("projects[%d].path is required", i)
		}
		if p.Name != "" && strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("projects[%d].name must be non-empty when provided", i)
		}
	}

	return nil
}

func normalizeForSave(cfg UserConfig) (UserConfig, error) {
	if cfg.Version == 0 {
		cfg.Version = SupportedConfigVersion
	}
	if cfg.Version != SupportedConfigVersion {
		return UserConfig{}, fmt.Errorf("unsupported version %d (supported: %d)", cfg.Version, SupportedConfigVersion)
	}

	normalized := UserConfig{
		Version:  SupportedConfigVersion,
		Projects: make([]ProjectConfig, 0, len(cfg.Projects)),
	}

	seen := map[string]struct{}{}
	for i, p := range cfg.Projects {
		if strings.TrimSpace(p.Path) == "" {
			return UserConfig{}, fmt.Errorf("projects[%d].path is required", i)
		}
		if p.Name != "" && strings.TrimSpace(p.Name) == "" {
			return UserConfig{}, fmt.Errorf("projects[%d].name must be non-empty when provided", i)
		}

		canonicalPath, err := CanonicalPath(p.Path)
		if err != nil {
			return UserConfig{}, fmt.Errorf("projects[%d].path %q is not canonicalizable: %w", i, p.Path, err)
		}
		if !filepath.IsAbs(canonicalPath) {
			return UserConfig{}, fmt.Errorf("projects[%d].path %q is not absolute after normalization", i, p.Path)
		}
		if _, ok := seen[canonicalPath]; ok {
			return UserConfig{}, fmt.Errorf("duplicate canonical project path: %s", canonicalPath)
		}
		seen[canonicalPath] = struct{}{}

		normalized.Projects = append(normalized.Projects, ProjectConfig{
			Path: canonicalPath,
			Name: strings.TrimSpace(p.Name),
		})
	}

	sort.SliceStable(normalized.Projects, func(i, j int) bool {
		iDisplay := normalized.Projects[i].Name
		if iDisplay == "" {
			iDisplay = filepath.Base(normalized.Projects[i].Path)
		}
		jDisplay := normalized.Projects[j].Name
		if jDisplay == "" {
			jDisplay = filepath.Base(normalized.Projects[j].Path)
		}
		if iDisplay != jDisplay {
			return iDisplay < jDisplay
		}
		return normalized.Projects[i].Path < normalized.Projects[j].Path
	})

	return normalized, nil
}

func parseUserConfigTOML(content []byte) (UserConfig, error) {
	cfg := UserConfig{Projects: []ProjectConfig{}}
	var inProject bool

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}

		if line == "[[projects]]" {
			cfg.Projects = append(cfg.Projects, ProjectConfig{})
			inProject = true
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return UserConfig{}, fmt.Errorf("line %d: expected key/value assignment", lineNo)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "version":
			if inProject {
				return UserConfig{}, fmt.Errorf("line %d: version must be top-level", lineNo)
			}
			v, err := strconv.Atoi(value)
			if err != nil {
				return UserConfig{}, fmt.Errorf("line %d: invalid version value %q", lineNo, value)
			}
			cfg.Version = v
		case "path":
			if !inProject || len(cfg.Projects) == 0 {
				return UserConfig{}, fmt.Errorf("line %d: path must be inside [[projects]]", lineNo)
			}
			s, err := parseTOMLString(value)
			if err != nil {
				return UserConfig{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			cfg.Projects[len(cfg.Projects)-1].Path = s
		case "name":
			if !inProject || len(cfg.Projects) == 0 {
				return UserConfig{}, fmt.Errorf("line %d: name must be inside [[projects]]", lineNo)
			}
			s, err := parseTOMLString(value)
			if err != nil {
				return UserConfig{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			cfg.Projects[len(cfg.Projects)-1].Name = s
		default:
			return UserConfig{}, fmt.Errorf("line %d: unknown key %q", lineNo, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return UserConfig{}, fmt.Errorf("failed reading config content: %w", err)
	}

	if cfg.Version == 0 {
		return UserConfig{}, fmt.Errorf("missing required version")
	}

	return cfg, nil
}

func parseTOMLString(v string) (string, error) {
	if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
		return "", fmt.Errorf("expected quoted string, got %q", v)
	}
	s, err := strconv.Unquote(v)
	if err != nil {
		return "", fmt.Errorf("invalid quoted string %q", v)
	}
	return s, nil
}

func stripInlineComment(line string) string {
	inQuote := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if r == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func renderUserConfigTOML(cfg UserConfig) []byte {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("version = %d\n", cfg.Version))
	if len(cfg.Projects) > 0 {
		b.WriteString("\n")
	}
	for i, p := range cfg.Projects {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[projects]]\n")
		b.WriteString(fmt.Sprintf("path = %s\n", strconv.Quote(p.Path)))
		if p.Name != "" {
			b.WriteString(fmt.Sprintf("name = %s\n", strconv.Quote(p.Name)))
		}
	}
	return []byte(b.String())
}
