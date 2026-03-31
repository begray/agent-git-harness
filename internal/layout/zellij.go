package layout

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/begray/agh/internal/config"
)

// zellijManager creates panes in the current zellij session.
type zellijManager struct {
	cfg config.Config
}

func newZellijManager(cfg config.Config) *zellijManager {
	return &zellijManager{cfg: cfg}
}

func (m *zellijManager) Name() string { return "zellij" }

func (m *zellijManager) Start(feature, worktreeDir, shellCmd string) (SessionHandle, error) {
	if !m.inZellij() {
		return SessionHandle{}, fmt.Errorf("not inside a zellij session (set layout = \"terminal\" or \"sway\" in config, or start zellij first)")
	}

	// Determine direction: right for first pane, down for subsequent.
	direction := "right"
	if m.hasExistingAghPane() {
		direction = "down"
	}

	// zellij action new-pane --direction <dir> --cwd <dir> -- bash -c 'shellCmd'
	args := []string{"action", "new-pane", "--direction", direction,
		"--cwd", worktreeDir, "--",
		"bash", "-c", shellCmd}

	cmd := exec.Command("zellij", args...)
	if err := cmd.Run(); err != nil {
		return SessionHandle{}, fmt.Errorf("zellij new-pane: %w", err)
	}

	// Rename the pane so we can find it later
	paneName := "agh-" + feature
	renameCmd := exec.Command("zellij", "action", "rename-pane", paneName)
	if err := renameCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to rename zellij pane: %v\n", err)
	}

	return SessionHandle{
		Type:   "zellij",
		PaneID: paneName,
	}, nil
}

func (m *zellijManager) IsAlive(handle SessionHandle) bool {
	if handle.PaneID == "" {
		return isProcessAlive(handle.PID)
	}
	// Check if pane exists by searching dump-layout output
	return m.paneExists(handle.PaneID)
}

func (m *zellijManager) Kill(handle SessionHandle) error {
	if handle.PaneID == "" {
		return killProcess(handle.PID)
	}

	// Focus the pane by name, then close it.
	// zellij doesn't have "kill pane by ID" — we need to focus it first.
	focusCmd := exec.Command("zellij", "action", "go-to-pane-by-name", handle.PaneID)
	if err := focusCmd.Run(); err != nil {
		return fmt.Errorf("zellij focus pane %q: %w", handle.PaneID, err)
	}

	closeCmd := exec.Command("zellij", "action", "close-pane")
	if err := closeCmd.Run(); err != nil {
		return fmt.Errorf("zellij close-pane: %w", err)
	}
	return nil
}

func (m *zellijManager) inZellij() bool {
	return os.Getenv("ZELLIJ") != ""
}

// hasExistingAghPane checks if there are already agh-named panes.
func (m *zellijManager) hasExistingAghPane() bool {
	cmd := exec.Command("zellij", "action", "dump-layout")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "agh-")
}

// paneExists checks if a pane with the given name exists in the layout.
func (m *zellijManager) paneExists(paneName string) bool {
	cmd := exec.Command("zellij", "action", "dump-layout")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), paneName)
}
