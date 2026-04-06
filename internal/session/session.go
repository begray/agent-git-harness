package session

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// SpawnIDE launches IntelliJ IDEA for the given worktree directory.
// If IDEA is already running, it opens the project in the existing instance
// via the JetBrains Toolbox protocol handler, avoiding the DirectoryLock
// conflict introduced in recent IDEA versions.
// Returns the PID of the IDEA process (existing or newly spawned).
func SpawnIDE(worktreeDir string) (int, error) {
	absDir, err := filepath.Abs(worktreeDir)
	if err != nil {
		absDir = worktreeDir
	}

	// If IDEA is already running, open the project in the existing instance.
	// Recent IDEA versions use a DirectoryLock with a Unix domain socket for
	// IPC. When the socket goes stale (e.g. after an update), spawning a new
	// `idea` process fails. The Toolbox protocol handler bypasses this.
	if existingPID := findAnyIDEProcess(); existingPID > 0 {
		if err := openInRunningIDE(absDir); err != nil {
			return 0, fmt.Errorf("IDEA is already running (pid %d), but could not open project: %w\nTry opening %s manually in IDEA", existingPID, err, absDir)
		}
		return existingPID, nil
	}

	cmd := exec.Command("idea", absDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting IDEA: %w", err)
	}

	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// openInRunningIDE opens a project in an already-running IDEA instance
// using the JetBrains Toolbox protocol handler (jetbrains://idea/...).
// This avoids the DirectoryLock socket issue by delegating to the Toolbox
// daemon which has its own IPC mechanism with the running IDE.
func openInRunningIDE(projectDir string) error {
	uri := fmt.Sprintf("jetbrains://idea/navigate/reference?project=%s", url.QueryEscape(projectDir))
	cmd := exec.Command("xdg-open", uri)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xdg-open jetbrains:// URI: %w", err)
	}
	return nil
}

// findAnyIDEProcess searches /proc for any running IntelliJ IDEA JVM process.
// Returns the PID if found, 0 otherwise.
func findAnyIDEProcess() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
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

		args := string(cmdline)
		if strings.Contains(args, "com.intellij.idea.Main") {
			return pid
		}
	}

	return 0
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
