// Package sandbox wraps AI agent sessions in a greywall sandbox.
//
// Greywall (https://github.com/GreyhavenHQ/greywall) provides container-free,
// deny-by-default sandboxing via bubblewrap, Landlock, seccomp, and eBPF on
// Linux. Network traffic is routed through greyproxy, which handles credential
// substitution — agents receive placeholder tokens instead of raw secrets, and
// greyproxy re-injects the real values into outbound traffic transparently.
//
// This package generates a per-session greywall settings file that extends the
// built-in agent profile with project-specific paths (project root, ancestor
// context files, user-configured extras), then builds the greywall CLI invocation.
package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// greywallProfile names that correspond to agh's ai_tool values.
// Greywall ships built-in profiles for all of these.
var knownProfiles = map[string]bool{
	"claude":   true,
	"pi":       true,
	"opencode": true,
	"aider":    true,
}

// greywallSettings mirrors the greywall JSON config format (subset we use).
type greywallSettings struct {
	Filesystem greywallFilesystem `json:"filesystem,omitempty"`
}

type greywallFilesystem struct {
	AllowRead []string `json:"allowRead,omitempty"`
	AllowWrite []string `json:"allowWrite,omitempty"`
	DenyRead  []string `json:"denyRead,omitempty"`
}

// IsAvailable checks whether the greywall binary is installed.
func IsAvailable() bool {
	_, err := exec.LookPath("greywall")
	return err == nil
}

// WrapShellCmd wraps a shell command with greywall sandboxing.
// It generates a per-session settings file extending the agent's built-in
// greywall profile with project-specific paths, then returns the greywall
// CLI invocation string suitable for passing to `bash -c`.
func WrapShellCmd(shellCmd, worktreeDir, projectRootDir, aiTool string, cfg SandboxConfig) (string, error) {
	if !IsAvailable() {
		return "", fmt.Errorf("greywall is not installed; install it from https://github.com/GreyhavenHQ/greywall or set [sandbox] enabled = false")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	settings := buildSettings(worktreeDir, projectRootDir, homeDir, cfg)

	settingsPath, err := writeSettingsFile(settings)
	if err != nil {
		return "", fmt.Errorf("writing greywall settings: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("exec greywall")

	// Use built-in agent profile when available — covers agent config dirs,
	// auth tokens, and tool-specific caches out of the box.
	if knownProfiles[aiTool] {
		sb.WriteString(" --profile ")
		sb.WriteString(shellQuote(aiTool))
	}

	sb.WriteString(" --settings ")
	sb.WriteString(shellQuote(settingsPath))
	sb.WriteString(" -- bash -c ")
	sb.WriteString(shellQuote(shellCmd))

	return sb.String(), nil
}

// buildSettings constructs the greywall settings for this session.
// These are merged on top of the agent's built-in profile by greywall.
func buildSettings(worktreeDir, projectRootDir, homeDir string, cfg SandboxConfig) greywallSettings {
	var allowRead []string

	// Project root: a sibling directory to the worktree, not the CWD, so
	// greywall won't include it automatically. Needed for .agh config, shared
	// project files, and in-repo AGENTS.md / CLAUDE.md.
	if projectRootDir != worktreeDir {
		allowRead = append(allowRead, projectRootDir)
	}

	// Ancestor context files: AGENTS.md, CLAUDE.md, etc. placed at each level
	// of the directory hierarchy between the worktree and $HOME.
	allowRead = append(allowRead, scanContextFiles(worktreeDir, homeDir, cfg.ContextFiles)...)

	// User-configured extra readable paths.
	allowRead = append(allowRead, cfg.ExtraROPaths...)

	return greywallSettings{
		Filesystem: greywallFilesystem{
			AllowRead:  allowRead,
			AllowWrite: cfg.ExtraRWPaths,
			DenyRead:   cfg.ExtraDenyRead,
		},
	}
}

// scanContextFiles walks ancestor directories from worktreeDir up to (but not
// including) homeDir and returns paths of any matching context files found.
func scanContextFiles(worktreeDir, homeDir string, names []string) []string {
	if len(names) == 0 {
		return nil
	}
	var found []string
	dir := filepath.Dir(worktreeDir)
	for dir != homeDir && dir != "/" && dir != "." {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				found = append(found, path)
			}
		}
		dir = filepath.Dir(dir)
	}
	return found
}

// writeSettingsFile serialises settings to a temp file and returns its path.
// Greywall reads the file once at startup; it can be left in /tmp afterward.
func writeSettingsFile(settings greywallSettings) (string, error) {
	f, err := os.CreateTemp("", "agh-greywall-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// shellQuote wraps s in single quotes for safe shell embedding.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
