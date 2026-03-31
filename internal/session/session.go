package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

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

// IsIDEAlive checks whether an IDEA instance is running for the given
// worktree directory. It searches /proc instead of relying on the stored
// PID, which goes stale because the "idea" launcher exits immediately.
func IsIDEAlive(worktreeDir string) bool {
	_, err := FindIDEProcess(worktreeDir)
	return err == nil
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

// FindIDEProcess finds the IDEA process that has the given worktree directory
// in its command line. The "idea" launcher script exits immediately after
// starting the JVM, so the stored PID becomes stale. This function searches
// /proc for the actual IDEA process.
func FindIDEProcess(worktreeDir string) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("reading /proc: %w", err)
	}

	for _, entry := range entries {
		pid := 0
		if _, err := fmt.Sscanf(entry.Name(), "%d", &pid); err != nil {
			continue
		}

		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}

		// cmdline is null-separated; check if it looks like an IDEA process
		// with our worktree path
		args := string(cmdline)
		if strings.Contains(args, "idea") && strings.Contains(args, worktreeDir) {
			return pid, nil
		}
	}

	return 0, fmt.Errorf("no IDEA process found for %s", worktreeDir)
}
