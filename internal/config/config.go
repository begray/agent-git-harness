package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/begray/agh/internal/sandbox"
)

type Config struct {
	Terminal string     `toml:"terminal"`
	AITool   string     `toml:"ai_tool"`
	Layout   string     `toml:"layout"` // "auto", "tmux", "zellij", "sway", "none"

	// Deprecated: use Layout instead. Kept for backward compat.
	Sway SwayConfig `toml:"sway"`

	Terminals     map[string]TerminalConfig `toml:"terminals"`
	AITools       map[string]AIToolConfig   `toml:"ai_tools"`
	LayoutOptions LayoutOptionsConfig       `toml:"layout_options"`
	Sandbox       sandbox.SandboxConfig     `toml:"sandbox"`
}

type SwayConfig struct {
	Enabled bool   `toml:"enabled"`
	Layout  string `toml:"layout"`
}

type LayoutOptionsConfig struct {
	Sway   SwayLayoutConfig   `toml:"sway"`
	Tmux   TmuxLayoutConfig   `toml:"tmux"`
	Zellij ZellijLayoutConfig `toml:"zellij"`
}

type SwayLayoutConfig struct {
	Arrange string `toml:"arrange"`
}

type TmuxLayoutConfig struct {
	SessionName string `toml:"session_name"`
}

type ZellijLayoutConfig struct {
	// Reserved for future options
}

type TerminalConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type AIToolConfig struct {
	Command    string   `toml:"command"`
	Args       []string `toml:"args"`
	ResumeArgs []string `toml:"resume_args"`
}

// DetectTerminal identifies the current terminal from environment variables.
// Returns empty string if no known terminal is detected.
func DetectTerminal() string {
	// TERM_PROGRAM is the most portable signal
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "wezterm":
		return "wezterm"
	case "alacritty":
		return "alacritty"
	case "foot":
		return "foot"
	case "kitty":
		return "kitty"
	}

	// Fall back to terminal-specific env vars
	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		return "wezterm"
	}
	if os.Getenv("FOOT_SOCK") != "" {
		return "foot"
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if os.Getenv("ALACRITTY_SOCKET") != "" {
		return "alacritty"
	}

	return ""
}

func DefaultConfig() Config {
	return Config{
		Sandbox: sandbox.DefaultSandboxConfig(),
		Terminal: "auto",
		AITool:   "claude",
		Layout:   "auto",
		Sway: SwayConfig{
			Enabled: false,
			Layout:  "right-stack",
		},
		LayoutOptions: LayoutOptionsConfig{
			Sway: SwayLayoutConfig{
				Arrange: "right-stack",
			},
			Tmux: TmuxLayoutConfig{
				SessionName: "agh",
			},
		},
		Terminals: map[string]TerminalConfig{
			"wezterm": {
				Command: "wezterm",
				Args:    []string{"start", "--class", "agh-{{feature}}", "--cwd", "{{workdir}}", "--"},
			},
			"foot": {
				Command: "foot",
				Args:    []string{"-a", "agh-{{feature}}"},
			},
			"alacritty": {
				Command: "alacritty",
				Args:    []string{"--class", "agh-{{feature}}", "-e"},
			},
			"kitty": {
				Command: "kitty",
				Args:    []string{"--class", "agh-{{feature}}"},
			},
		},
		AITools: map[string]AIToolConfig{
			"claude": {
				Command: "claude",
				Args:    []string{},
				ResumeArgs: []string{"--continue"},
			},
			"pi": {
				Command: "pi",
				Args:    []string{},
				ResumeArgs: []string{"--continue"},
			},
		},
	}
}

// GlobalConfigDir returns the directory for the global agh config.
// Uses $XDG_CONFIG_HOME/agh or ~/.config/agh as fallback.
func GlobalConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "agh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agh")
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return filepath.Join(GlobalConfigDir(), "config.toml")
}

func Load(aghDir string) (Config, error) {
	cfg := DefaultConfig()

	// Layer 1: global config (~/.config/agh/config.toml)
	if err := loadLayer(&cfg, GlobalConfigPath()); err != nil {
		return cfg, fmt.Errorf("global config: %w", err)
	}

	// Layer 2: project config (.agh/config.toml)
	if err := loadLayer(&cfg, filepath.Join(aghDir, "config.toml")); err != nil {
		return cfg, fmt.Errorf("project config: %w", err)
	}

	return cfg, nil
}

// loadLayer reads a TOML file and merges it into the config.
// Missing files are silently skipped.
func loadLayer(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	return nil
}

// ResolveTerminal returns the effective terminal name, resolving "auto" if needed.
func (c Config) ResolveTerminal() (string, error) {
	if c.Terminal != "auto" {
		return c.Terminal, nil
	}
	detected := DetectTerminal()
	if detected == "" {
		return "", fmt.Errorf("could not auto-detect terminal (set 'terminal' in config)")
	}
	return detected, nil
}


// ResolveLayout returns the effective layout manager name.
// Handles "auto" detection and backward compat with [sway] config.
func (c Config) ResolveLayout() string {
	if c.Layout != "" && c.Layout != "auto" {
		return c.Layout
	}
	// Backward compat: if layout not set but sway.enabled is explicit
	if c.Layout == "" && c.Sway.Enabled {
		return "sway"
	}
	if c.Layout == "auto" || c.Layout == "" {
		if os.Getenv("TMUX") != "" {
			return "tmux"
		}
		if os.Getenv("ZELLIJ") != "" {
			return "zellij"
		}
		if os.Getenv("SWAYSOCK") != "" {
			return "sway"
		}
		return "terminal"
	}
	return "terminal"
}

func (c Config) TerminalArgs(feature, workdir string) (string, []string, error) {
	terminal, err := c.ResolveTerminal()
	if err != nil {
		return "", nil, err
	}
	tc, ok := c.Terminals[terminal]
	if !ok {
		return "", nil, fmt.Errorf("unknown terminal %q (detected or configured); add a [terminals.%s] section to config", terminal, terminal)
	}

	replacer := strings.NewReplacer("{{feature}}", feature, "{{workdir}}", workdir)
	args := make([]string, len(tc.Args))
	for i, a := range tc.Args {
		args[i] = replacer.Replace(a)
	}

	return tc.Command, args, nil
}

func (c Config) AIToolArgs(resume bool) (string, []string, error) {
	at, ok := c.AITools[c.AITool]
	if !ok {
		return "", nil, fmt.Errorf("unknown ai tool %q", c.AITool)
	}
	if resume && len(at.ResumeArgs) > 0 {
		args := append(at.Args[:len(at.Args):len(at.Args)], at.ResumeArgs...)
		return at.Command, args, nil
	}
	return at.Command, at.Args, nil
}

// AIToolBaseArgs returns the AI tool command and base args (without resume args).
func (c Config) AIToolBaseArgs() ([]string, error) {
	at, ok := c.AITools[c.AITool]
	if !ok {
		return nil, fmt.Errorf("unknown ai tool %q", c.AITool)
	}
	return at.Args, nil
}

// WriteDefault writes the default config as a commented TOML file.
func WriteDefault(path string) error {
	content := `# agh project configuration
# Overrides global config (~/.config/agh/config.toml) and built-in defaults.
# See: agh --help

# Terminal emulator: "auto" detects from environment, or set explicitly
# Supported: wezterm, foot, alacritty, kitty
# Only used with layout = "sway" or "terminal" (tmux/zellij manage their own panes)
# terminal = "auto"

# Default AI coding tool (uncomment to override global config)
# ai_tool = "claude"

# Layout manager: "auto" detects from environment, or set explicitly
# Supported: auto, tmux, zellij, sway, none
# auto detection: $TMUX → tmux, $ZELLIJ → zellij, $SWAYSOCK → sway, else → terminal
# layout = "auto"

[layout_options.tmux]
session_name = "agh"

[layout_options.sway]
arrange = "right-stack"

# Deprecated: use layout = "sway" instead
# [sway]
# enabled = false
# layout = "right-stack"

[terminals.wezterm]
command = "wezterm"
args = ["start", "--class", "agh-{{feature}}", "--cwd", "{{workdir}}", "--"]

[terminals.foot]
command = "foot"
args = ["-a", "agh-{{feature}}"]

[terminals.alacritty]
command = "alacritty"
args = ["--class", "agh-{{feature}}", "-e"]

[terminals.kitty]
command = "kitty"
args = ["--class", "agh-{{feature}}"]

[ai_tools.claude]
command = "claude"
args = []
resume_args = ["--continue"]

[ai_tools.pi]
command = "pi"
args = []
resume_args = ["--continue"]

# Sandbox: greywall-based isolation for AI agent sessions.
# Greywall provides deny-by-default filesystem access, network filtering via
# greyproxy (with credential substitution), Landlock, seccomp, and eBPF monitoring.
# Built-in profiles for claude, pi, opencode handle agent-specific paths.
# Requires greywall: https://github.com/GreyhavenHQ/greywall
# Requires greyproxy for network access: run "greywall setup"
[sandbox]
enabled = false

# Additional paths the agent needs read access to (beyond the built-in profile).
# extra_ro_paths = ["/data/shared-libs"]

# Additional paths the agent needs write access to.
# extra_rw_paths = ["/var/run/docker.sock"]

# Additional deny-read rules on top of the built-in profile defaults.
# extra_deny_read = ["~/.vault-token", "~/.config/CorpTool"]

# Filenames to scan for in ancestor directories (worktree up to $HOME).
# Found files are added to allowRead so agents can read layered project context.
context_files = ["AGENTS.md", "CLAUDE.md", ".claude/settings.json"]
`
	return os.WriteFile(path, []byte(content), 0o644)
}

// WriteGlobalDefault writes a minimal global config template.
// Global config sets personal defaults that apply to all projects.
// Project-level .agh/config.toml overrides these values.
func WriteGlobalDefault(path string) error {
	content := `# agh global configuration (~/.config/agh/config.toml)
# Personal defaults applied to all projects.
# Project-level .agh/config.toml overrides these values.

# Default AI coding tool: "claude", "pi", etc.
# ai_tool = "claude"

# Terminal emulator: "auto", "wezterm", "foot", "alacritty", "kitty"
# terminal = "auto"

# Layout manager: "auto", "tmux", "zellij", "sway", "none"
# layout = "auto"

# You can also define custom AI tools or override built-in ones:
# [ai_tools.my-tool]
# command = "my-tool"
# args = []
# resume_args = ["--continue"]
`
	return os.WriteFile(path, []byte(content), 0o644)
}
