package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Terminal string     `toml:"terminal"`
	AITool   string     `toml:"ai_tool"`
	Sway     SwayConfig `toml:"sway"`

	Terminals map[string]TerminalConfig `toml:"terminals"`
	AITools   map[string]AIToolConfig   `toml:"ai_tools"`
}

type SwayConfig struct {
	Enabled bool   `toml:"enabled"`
	Layout  string `toml:"layout"`
}

type TerminalConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type AIToolConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
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
		Terminal: "auto",
		AITool:   "claude",
		Sway: SwayConfig{
			Enabled: true,
			Layout:  "right-stack",
		},
		Terminals: map[string]TerminalConfig{
			"wezterm": {
				Command: "wezterm",
				Args:    []string{"start", "--class", "agh-{{feature}}", "--"},
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
			},
			"aider": {
				Command: "aider",
				Args:    []string{},
			},
		},
	}
}

func Load(aghDir string) (Config, error) {
	cfg := DefaultConfig()
	configPath := filepath.Join(aghDir, "config.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
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

func (c Config) TerminalArgs(feature string) (string, []string, error) {
	terminal, err := c.ResolveTerminal()
	if err != nil {
		return "", nil, err
	}
	tc, ok := c.Terminals[terminal]
	if !ok {
		return "", nil, fmt.Errorf("unknown terminal %q (detected or configured); add a [terminals.%s] section to config", terminal, terminal)
	}

	args := make([]string, len(tc.Args))
	for i, a := range tc.Args {
		args[i] = strings.ReplaceAll(a, "{{feature}}", feature)
	}

	return tc.Command, args, nil
}

func (c Config) AIToolArgs() (string, []string, error) {
	at, ok := c.AITools[c.AITool]
	if !ok {
		return "", nil, fmt.Errorf("unknown ai tool %q", c.AITool)
	}
	return at.Command, at.Args, nil
}

// WriteDefault writes the default config as a commented TOML file.
func WriteDefault(path string) error {
	content := `# agh configuration
# See: agh --help

# Terminal emulator: "auto" detects from environment, or set explicitly
# Supported: wezterm, foot, alacritty, kitty
terminal = "auto"

# Default AI coding tool
ai_tool = "claude"

[sway]
# Enable sway window management (move feature terminals to the right, stack them)
enabled = true
layout = "right-stack"

[terminals.wezterm]
command = "wezterm"
args = ["start", "--class", "agh-{{feature}}", "--"]

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

[ai_tools.aider]
command = "aider"
args = []
`
	return os.WriteFile(path, []byte(content), 0o644)
}
