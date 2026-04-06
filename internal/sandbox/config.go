package sandbox

// SandboxConfig defines sandbox settings from the [sandbox] config section.
type SandboxConfig struct {
	Enabled bool `toml:"enabled"`

	// ExtraROPaths are added to allowRead on top of the agent profile defaults.
	// Useful for paths outside the project (e.g. shared corp libraries, tool configs).
	ExtraROPaths []string `toml:"extra_ro_paths"`

	// ExtraRWPaths are added to allowWrite on top of the agent profile defaults.
	// Useful for additional caches or sockets the agent needs to write to.
	ExtraRWPaths []string `toml:"extra_rw_paths"`

	// ExtraDenyRead are added to denyRead on top of the agent profile defaults.
	// Useful for org-specific credential files not covered by the built-in deny list.
	ExtraDenyRead []string `toml:"extra_deny_read"`

	// ContextFiles lists filenames to scan for in ancestor directories between
	// the worktree and $HOME. Any found are added to allowRead so agents can
	// read project context placed at multiple levels of the directory hierarchy.
	ContextFiles []string `toml:"context_files"`
}

// DefaultSandboxConfig returns sandbox config with sensible defaults.
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		Enabled:       false,
		ExtraROPaths:  []string{},
		ExtraRWPaths:  []string{},
		ExtraDenyRead: []string{},
		ContextFiles:  []string{"AGENTS.md", "CLAUDE.md", ".claude/settings.json"},
	}
}
