package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/begray/agh/internal/config"
)

// SpawnTerminal launches a terminal window running the AI tool in the given directory.
// Returns the PID of the terminal process.
func SpawnTerminal(cfg config.Config, feature, worktreeDir string) (int, error) {
	termCmd, termArgs, err := cfg.TerminalArgs(feature, worktreeDir)
	if err != nil {
		return 0, err
	}

	aiCmd, aiArgs, err := cfg.AIToolArgs()
	if err != nil {
		return 0, err
	}

	// Terminal args + AI tool command + AI tool args
	args := append(termArgs, aiCmd)
	args = append(args, aiArgs...)

	cmd := exec.Command(termCmd, args...)
	cmd.Dir = worktreeDir
	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting terminal: %w", err)
	}

	// Release the process so it's not waited on
	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// SpawnIDE launches IntelliJ IDEA for the given worktree directory.
// Returns the PID of the IDEA process.
func SpawnIDE(worktreeDir string) (int, error) {
	cmd := exec.Command("idea", worktreeDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting IDEA: %w", err)
	}

	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// ArrangeSway moves the feature terminal to the right and stacks it.
// Polls for the window to appear before arranging.
func ArrangeSway(cfg config.Config, feature string) error {
	if !cfg.Sway.Enabled {
		return nil
	}

	windowID := "agh-" + feature

	// Wait for the window to appear in sway's tree
	if !waitForSwayWindow(windowID, 5*time.Second) {
		fmt.Fprintf(os.Stderr, "warning: sway window %q did not appear within timeout\n", windowID)
		return nil
	}

	// Use class= for XWayland windows (e.g. wezterm), app_id= for native Wayland
	selector := swaySelector(windowID)

	// Move the new window to the right
	moveCmd := exec.Command("swaymsg", fmt.Sprintf(`%s move right`, selector))
	if err := moveCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sway move failed: %v\n", err)
		return nil
	}

	// Set stacking layout on the right container
	layoutCmd := exec.Command("swaymsg", fmt.Sprintf(`%s layout stacking`, selector))
	if err := layoutCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sway layout failed: %v\n", err)
	}

	return nil
}

// swaySelector returns the appropriate swaymsg selector for a window.
// XWayland apps (like wezterm) use window_properties.class, while
// native Wayland apps use app_id.
func swaySelector(windowID string) string {
	cmd := exec.Command("swaymsg", "-t", "get_tree")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf(`[app_id="%s"]`, windowID)
	}
	tree := string(out)
	if strings.Contains(tree, fmt.Sprintf(`"class":"%s"`, windowID)) {
		return fmt.Sprintf(`[class="%s"]`, windowID)
	}
	return fmt.Sprintf(`[app_id="%s"]`, windowID)
}

// waitForSwayWindow polls sway's tree until a window with the given id appears.
func waitForSwayWindow(windowID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if swayWindowExists(windowID) {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// swayWindowExists checks if a window with the given id exists in sway's tree
// as either app_id (Wayland native) or class (XWayland).
func swayWindowExists(windowID string) bool {
	cmd := exec.Command("swaymsg", "-t", "get_tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	tree := string(out)
	return strings.Contains(tree, fmt.Sprintf(`"app_id":"%s"`, windowID)) ||
		strings.Contains(tree, fmt.Sprintf(`"class":"%s"`, windowID))
}

// IsProcessAlive checks if a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// KillProcess sends SIGTERM to a process by PID.
func KillProcess(pid int) error {
	if pid == 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // Process doesn't exist
	}
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		return nil // Already dead
	}
	return nil
}
