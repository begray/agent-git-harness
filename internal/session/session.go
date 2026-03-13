package session

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

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
func ArrangeSway(cfg config.Config, feature string) error {
	if !cfg.Sway.Enabled {
		return nil
	}

	appID := "agh-" + feature

	// Move the new window to the right
	moveCmd := exec.Command("swaymsg", fmt.Sprintf(`[app_id="%s"] move right`, appID))
	if err := moveCmd.Run(); err != nil {
		// Non-fatal: window might not be ready yet or sway not available
		fmt.Fprintf(os.Stderr, "warning: sway move failed (window may not be ready): %v\n", err)
		return nil
	}

	// Set stacking layout on the right container
	layoutCmd := exec.Command("swaymsg", fmt.Sprintf(`[app_id="%s"] layout stacking`, appID))
	if err := layoutCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sway layout failed: %v\n", err)
	}

	return nil
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
