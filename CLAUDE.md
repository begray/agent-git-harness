# CLAUDE.md — Agent Git Harness (agh)

## Project Overview

`agh` is a Go CLI tool for managing AI-assisted feature work with git worktrees. It spawns isolated worktrees with AI coding sessions (Claude Code, aider, etc.) and optional IDE support.

## Build & Run

```bash
go build -o agh .        # build binary
go install .             # install to $GOPATH/bin
./agh                    # run locally
```

No test suite exists yet.

## Project Structure

```
main.go                         # entrypoint, calls cmd.Execute()
cmd/                            # cobra CLI commands
  root.go                       # root command, sets up cobra
  start.go                      # create branch + worktree + spawn AI session
  stop.go                       # remove worktree, optionally delete branch
  list.go                       # list active features
  status.go                     # check health of features
  diff.go                       # show changes in a feature worktree
  exec.go                       # run commands in a feature worktree
  init.go                       # initialize .agh/config.toml
  completion.go                 # shell completions (bash/zsh/fish)
internal/
  config/config.go              # TOML config loading, defaults, terminal/AI tool definitions
  project/project.go            # project root detection, feature state persistence (.agh/features/)
  session/session.go            # terminal + AI tool process spawning, sway integration
  worktree/worktree.go          # git worktree + branch operations
```

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/BurntSushi/toml` — config parsing
- Go 1.22+

## Conventions

- All commands resolve the project root via `project.FindRoot()` which walks up to find a git repo with `.agh/` or a main worktree.
- Feature state is stored as JSON in `.agh/features/<name>.json`.
- Config is `.agh/config.toml` (auto-created on first use).
- Terminal and AI tool commands use `{{feature}}` template placeholders.
- Worktrees are created as sibling directories: `project-dir` → `project-dir-<feature>`.
- No external services or network calls — purely local git + process management.

## Code Style

- Standard Go style, `gofmt`/`goimports`.
- Error handling: return errors up, commands print and `os.Exit(1)`.
- cobra command pattern: each file in `cmd/` registers one command via `init()`.
