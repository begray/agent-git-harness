# AGENT.md — Agent Git Harness (agh)

## Project Overview

`agh` (Agent Git Harness) is a Go CLI that manages AI-assisted feature development using git worktrees. It creates isolated worktrees per feature, spawns AI coding tools (Claude Code, aider) in terminal windows, and optionally opens IDEs — keeping the main checkout clean.

## Build

```bash
go build -o agh .
go install .
```

No tests yet.

## Architecture

### Entrypoint
- `main.go` → `cmd.Execute()` (cobra root command)

### Commands (`cmd/`)
Each file registers a cobra subcommand via `init()`:
- **start** — creates branch + worktree, spawns terminal with AI tool, optional IDE + sway layout
- **stop** — removes worktree (optionally deletes branch), cleans up feature state
- **list** — lists active features from `.agh/features/`
- **status** — checks process liveness and worktree health
- **diff** — runs `git diff` in feature worktree
- **exec** — runs arbitrary commands in feature worktree
- **init** — generates `.agh/config.toml`
- **completion** — shell completion for bash/zsh/fish

### Internal packages (`internal/`)
- **config** — loads `.agh/config.toml`, provides defaults for terminals (wezterm, foot, alacritty, kitty) and AI tools (claude, aider, pi), auto-detects terminal from environment
- **project** — finds project root (walks up to git repo), manages feature state as JSON files in `.agh/features/`
- **session** — spawns terminal + AI tool processes, handles sway window management via `swaymsg`
- **worktree** — git worktree creation/removal, branch management via `git` CLI

### Data flow
```
user runs `agh start foo` →
  project.FindRoot() finds repo root →
  config.Load() reads .agh/config.toml →
  worktree.Create() makes branch + worktree →
  session.Start() spawns terminal with AI tool →
  project.SaveFeature() persists state to .agh/features/foo.json
```

## Conventions

- Feature state: `.agh/features/<name>.json` (JSON with branch, worktree path, PIDs)
- Config: `.agh/config.toml` with `[terminals.*]` and `[ai_tools.*]` sections
- Worktree placement: sibling directories (`~/src/project/` → `~/src/project-feature/`)
- Template vars: `{{feature}}` in terminal/AI tool args replaced at runtime
- No network calls — local git and process management only
- Standard Go error handling, cobra command pattern
- Go 1.22+, deps: cobra, BurntSushi/toml
