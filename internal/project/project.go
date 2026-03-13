package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vvinogradov/agh/internal/config"
)

// Project represents a git project with .agh/ state directory.
type Project struct {
	// RootDir is the main checkout directory (not a worktree).
	RootDir string
	// Name is the base name of the project directory.
	Name   string
	AghDir string
	Config config.Config
}

// Feature represents an active feature work session.
type Feature struct {
	Name          string    `json:"name"`
	Branch        string    `json:"branch"`
	Worktree      string    `json:"worktree"`
	BaseBranch    string    `json:"base_branch"`
	ParentFeature string    `json:"parent_feature,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	IDE           string    `json:"ide,omitempty"`
	AITool        string    `json:"ai_tool"`
	TerminalPID   int       `json:"terminal_pid,omitempty"`
	IDEPID        int       `json:"ide_pid,omitempty"`
}

// Detect finds the project root from the current directory.
// Works from both the main checkout and from worktrees.
func Detect() (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting cwd: %w", err)
	}

	rootDir, err := findProjectRoot(cwd)
	if err != nil {
		return nil, err
	}

	aghDir := filepath.Join(rootDir, ".agh")
	cfg, err := config.Load(aghDir)
	if err != nil {
		return nil, err
	}

	return &Project{
		RootDir: rootDir,
		Name:    filepath.Base(rootDir),
		AghDir:  aghDir,
		Config:  cfg,
	}, nil
}

// findProjectRoot resolves the main checkout directory.
// If cwd is a worktree, follows .git file back to the main repo.
func findProjectRoot(dir string) (string, error) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return "", fmt.Errorf("not a git repository: %s", dir)
	}

	// Regular directory = main checkout
	if info.IsDir() {
		return dir, nil
	}

	// File = worktree, read gitdir pointer
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("reading .git file: %w", err)
	}

	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("unexpected .git file content: %s", line)
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(dir, gitdir)
	}

	// gitdir looks like: /path/to/main/.git/worktrees/<name>
	// Walk up to find the .git dir, then its parent is the main checkout
	dotGit := gitdir
	for filepath.Base(dotGit) != ".git" {
		parent := filepath.Dir(dotGit)
		if parent == dotGit {
			return "", fmt.Errorf("cannot resolve main checkout from gitdir: %s", gitdir)
		}
		dotGit = parent
	}

	return filepath.Dir(dotGit), nil
}

// InitAghDir creates the .agh/ directory structure if it doesn't exist.
// On first creation, also generates a default config file.
func (p *Project) InitAghDir() error {
	featuresDir := filepath.Join(p.AghDir, "features")

	// Check if this is a fresh init (directory doesn't exist yet)
	freshInit := false
	if _, err := os.Stat(p.AghDir); os.IsNotExist(err) {
		freshInit = true
	}

	if err := os.MkdirAll(featuresDir, 0o755); err != nil {
		return fmt.Errorf("creating .agh/features: %w", err)
	}

	gitignorePath := filepath.Join(p.AghDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("creating .gitignore: %w", err)
		}
	}

	// Auto-generate default config on first init
	if freshInit {
		configPath := filepath.Join(p.AghDir, "config.toml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := config.WriteDefault(configPath); err != nil {
				return fmt.Errorf("writing default config: %w", err)
			}
		}
	}

	return nil
}

// DetectIDE checks if the project uses IntelliJ IDEA.
func (p *Project) DetectIDE() string {
	ideaDir := filepath.Join(p.RootDir, ".idea")
	if info, err := os.Stat(ideaDir); err == nil && info.IsDir() {
		return "idea"
	}
	return ""
}

// SaveFeature writes feature state to .agh/features/<name>.json.
func (p *Project) SaveFeature(f *Feature) error {
	featuresDir := filepath.Join(p.AghDir, "features")
	if err := os.MkdirAll(featuresDir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling feature: %w", err)
	}

	path := filepath.Join(featuresDir, f.Name+".json")
	return os.WriteFile(path, data, 0o644)
}

// LoadFeature reads feature state from .agh/features/<name>.json.
func (p *Project) LoadFeature(name string) (*Feature, error) {
	path := filepath.Join(p.AghDir, "features", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading feature %q: %w", name, err)
	}

	var f Feature
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing feature %q: %w", name, err)
	}
	return &f, nil
}

// ListFeatures returns all active features.
func (p *Project) ListFeatures() ([]*Feature, error) {
	featuresDir := filepath.Join(p.AghDir, "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var features []*Feature
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		f, err := p.LoadFeature(name)
		if err != nil {
			continue
		}
		features = append(features, f)
	}
	return features, nil
}

// RemoveFeature deletes the feature state file.
func (p *Project) RemoveFeature(name string) error {
	path := filepath.Join(p.AghDir, "features", name+".json")
	return os.Remove(path)
}

// WorktreePath returns the expected worktree directory for a feature.
func (p *Project) WorktreePath(featureName string) string {
	parentDir := filepath.Dir(p.RootDir)
	return filepath.Join(parentDir, p.Name+"-"+featureName)
}
