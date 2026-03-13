# agh — Agent Git Harness

CLI tool for managing AI-assisted feature work with git worktrees. Spawns isolated worktrees with AI coding sessions (Claude Code, aider, etc.) and optional IDE support, keeping your main checkout clean.

## Install

```bash
go install github.com/begray/agh@latest
```

Or build from source:

```bash
git clone https://github.com/begray/agh.git
cd agh
go install .
```

## Usage

Run from your project's main checkout (or any of its worktrees):

```bash
# Start a new feature — creates branch, worktree, opens terminal with AI tool
cd ~/src/my-project
agh start WAC-123-new-feature

# Start from an existing feature (feature chaining)
cd ~/src/my-project-WAC-123-new-feature
agh start WAC-456-depends-on-123

# Resume work on an existing branch (auto-detects)
agh start WAC-123-new-feature  # works whether branch/worktree already exist

# List active features
agh list

# Check process and worktree health
agh status
agh status WAC-123-new-feature

# View changes in a feature
agh diff WAC-123-new-feature

# Run arbitrary commands in a feature's worktree
agh exec WAC-123-new-feature -- git log --oneline -5

# Stop a feature (remove worktree, keep branch)
agh stop WAC-123-new-feature

# Stop and delete branch
agh stop WAC-123-new-feature --delete-branch
```

## What `agh start` does

1. Creates a git branch and worktree as a sibling directory:
   `~/src/my-project/` → `~/src/my-project-WAC-123-new-feature/`
2. Spawns a terminal window running your AI tool (e.g. Claude Code)
3. Auto-detects IntelliJ IDEA projects and opens the worktree in IDEA
4. Optionally arranges windows via sway (feature terminals stacked on the right)
5. Tracks state in `.agh/features/` inside your project root

If the branch or worktree already exist, `agh start` attaches to them instead of failing.

## Configuration

Initialize config (also auto-generated on first use):

```bash
agh init
```

Config lives at `.agh/config.toml` in your project root (gitignored by default):

```toml
# "auto" detects from environment, or set explicitly
# Supported: wezterm, foot, alacritty, kitty
terminal = "auto"

# AI coding tool
ai_tool = "claude"

[sway]
enabled = true
layout = "right-stack"

[terminals.wezterm]
command = "wezterm"
args = ["start", "--class", "agh-{{feature}}", "--"]

[ai_tools.claude]
command = "claude"
args = []
```

`{{feature}}` in terminal args is replaced with the feature name at runtime.

## Shell completion

```bash
# Bash — add to ~/.bashrc
eval "$(agh completion bash)"

# Zsh — add to ~/.zshrc
eval "$(agh completion zsh)"

# Fish
agh completion fish | source
```

Tab-completes feature names for `stop`, `status`, `diff`, and `exec`.

## Project layout

```
.agh/                          # per-project, gitignored
├── config.toml                # terminal, AI tool, sway settings
├── features/
│   └── WAC-123-new-feature.json
└── .gitignore                 # contains "*"
```

## License

MIT
