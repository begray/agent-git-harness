package layout

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/begray/agh/internal/config"
)

// tmuxManager creates panes in the current tmux session.
type tmuxManager struct {
	cfg config.Config
}

func newTmuxManager(cfg config.Config) *tmuxManager {
	return &tmuxManager{cfg: cfg}
}

func (m *tmuxManager) Name() string { return "tmux" }

func (m *tmuxManager) Start(feature, worktreeDir, shellCmd string) (SessionHandle, error) {
	if !m.inTmux() {
		return SessionHandle{}, fmt.Errorf("not inside a tmux session (set layout = \"terminal\" or \"sway\" in config, or start tmux first)")
	}

	// Check if there are already agh panes in the current window.
	// First feature: split horizontally (create right column).
	// Subsequent: split vertically within an existing right pane.
	var args []string
	existingPane := m.findExistingAghPane()
	if existingPane != "" {
		// Split the existing agh pane vertically (stack below it)
		args = []string{"split-window", "-v", "-d", "-t", existingPane,
			"-P", "-F", "#{pane_id}",
			"-c", worktreeDir, "bash", "-c", shellCmd}
	} else {
		// First feature: split horizontally (right column)
		args = []string{"split-window", "-h", "-d",
			"-P", "-F", "#{pane_id}",
			"-c", worktreeDir, "bash", "-c", shellCmd}
	}

	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return SessionHandle{}, fmt.Errorf("tmux split-window: %w", err)
	}

	paneID := strings.TrimSpace(string(out))

	return SessionHandle{
		Type:   "tmux",
		PaneID: paneID,
	}, nil
}

func (m *tmuxManager) IsAlive(handle SessionHandle) bool {
	if handle.PaneID == "" {
		// Fallback to PID check for backward compat
		return isProcessAlive(handle.PID)
	}
	cmd := exec.Command("tmux", "display-message", "-p", "-t", handle.PaneID, "#{pane_pid}")
	return cmd.Run() == nil
}

func (m *tmuxManager) Kill(handle SessionHandle) error {
	if handle.PaneID == "" {
		return killProcess(handle.PID)
	}
	cmd := exec.Command("tmux", "kill-pane", "-t", handle.PaneID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux kill-pane: %w", err)
	}
	return nil
}

func (m *tmuxManager) inTmux() bool {
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_id}")
	return cmd.Run() == nil
}

// findExistingAghPane looks for a pane in the current window that was
// created by agh (has "agh-" in its title or command). We use a simple
// heuristic: check pane commands for our AI tool pattern.
// Returns the pane ID of any existing agh pane, or empty string.
func (m *tmuxManager) findExistingAghPane() string {
	// List all panes in the current window with their IDs and commands
	cmd := exec.Command("tmux", "list-panes", "-F", "#{pane_id} #{pane_current_command}")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	// The AI tool name gives us a hint
	aiTool := m.cfg.AITool

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		paneID, paneCmd := parts[0], parts[1]
		// Match if the pane is running our AI tool or bash (which wraps it)
		if strings.Contains(paneCmd, aiTool) {
			return paneID
		}
	}

	// Fallback: if there are more than 1 pane, the rightmost non-active one
	// is likely ours. But safer to return empty and create a new column.
	return ""
}
