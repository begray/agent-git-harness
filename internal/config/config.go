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

func DefaultConfig() Config {
	return Config{
		Terminal: "wezterm",
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

func (c Config) TerminalArgs(feature string) (string, []string, error) {
	tc, ok := c.Terminals[c.Terminal]
	if !ok {
		return "", nil, fmt.Errorf("unknown terminal %q", c.Terminal)
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

# Default terminal emulator
terminal = "wezterm"

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

[ai_tools.claude]
command = "claude"
args = []

[ai_tools.aider]
command = "aider"
args = []
`
	return os.WriteFile(path, []byte(content), 0o644)
}
