package layout

import (
	"fmt"
	"os"

	"github.com/begray/agh/internal/config"
)

// SessionHandle identifies a running layout session.
type SessionHandle struct {
	Type      string `json:"type"`                // "pid", "tmux", "zellij"
	PID       int    `json:"pid,omitempty"`        // terminal process PID (sway/terminal)
	SessionID string `json:"session_id,omitempty"` // multiplexer session name
	PaneID    string `json:"pane_id,omitempty"`    // tmux pane id / zellij pane id
}

// Manager creates and manages layout sessions for AI tools.
type Manager interface {
	// Name returns the layout manager name.
	Name() string

	// Start creates a new pane/window running shellCmd in worktreeDir.
	Start(feature, worktreeDir, shellCmd string) (SessionHandle, error)

	// IsAlive checks if the session is still running.
	IsAlive(handle SessionHandle) bool

	// Kill terminates the session.
	Kill(handle SessionHandle) error
}

// Detect returns the best layout manager name from the environment.
func Detect() string {
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

// New creates a layout manager based on config.
// For "auto", it detects from the environment.
func New(cfg config.Config) (Manager, error) {
	name := cfg.ResolveLayout()
	return newByName(name, cfg)
}

// NewForHandle creates a layout manager matching a stored session handle.
// Used by stop/status to match the manager to what was used at start time.
func NewForHandle(handle SessionHandle, cfg config.Config) (Manager, error) {
	switch handle.Type {
	case "tmux":
		return newByName("tmux", cfg)
	case "zellij":
		return newByName("zellij", cfg)
	case "pid":
		// Could be sway or terminal — doesn't matter, both use PID-based kill/alive.
		return newByName("terminal", cfg)
	default:
		return newByName("terminal", cfg)
	}
}

func newByName(name string, cfg config.Config) (Manager, error) {
	switch name {
	case "tmux":
		return newTmuxManager(cfg), nil
	case "zellij":
		return newZellijManager(cfg), nil
	case "sway":
		return newSwayManager(cfg), nil
	case "terminal", "none":
		return newTerminalManager(cfg), nil
	default:
		return nil, fmt.Errorf("unknown layout manager %q", name)
	}
}
