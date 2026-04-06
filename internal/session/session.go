package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// SpawnIDE launches IntelliJ IDEA for the given worktree directory.
// The Toolbox `idea` launcher handles both cases: if IDEA is already running,
// it opens the project in the existing instance; otherwise it starts a new one.
// Returns the PID of the launcher process.
func SpawnIDE(worktreeDir string) (int, error) {
	absDir, err := filepath.Abs(worktreeDir)
	if err != nil {
		absDir = worktreeDir
	}

	cmd := exec.Command("idea", absDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	// Strip snap environment variables that remap XDG paths. When agh runs
	// inside a snap terminal (e.g. alacritty), XDG_CACHE_HOME points to the
	// snap sandbox directory. IDEA uses XDG_CACHE_HOME to locate its IPC
	// socket, so it fails to find the already-running instance and tries to
	// start a second one, hitting the DirectoryLock error.
	cmd.Env = cleanSnapEnv()

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting IDEA: %w", err)
	}

	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// cleanSnapEnv returns a copy of the current environment with snap-specific
// variables removed so that child processes see standard XDG paths.
func cleanSnapEnv() []string {
	snapVars := map[string]bool{
		"SNAP": true, "SNAP_REVISION": true, "SNAP_ARCH": true,
		"SNAP_INSTANCE_NAME": true, "SNAP_INSTANCE_KEY": true,
		"SNAP_USER_DATA": true, "SNAP_USER_COMMON": true,
		"SNAP_COMMON": true, "SNAP_CONTEXT": true,
		"SNAP_REAL_HOME": true, "SNAP_REEXEC": true,
		"SNAP_EUID": true, "SNAP_UID": true,
		"SNAP_LAUNCHER_ARCH_TRIPLET": true,
		"XDG_CACHE_HOME": true, "XDG_DATA_HOME": true,
		"XDG_CONFIG_HOME": true, "XDG_STATE_HOME": true,
	}
	var env []string
	for _, e := range os.Environ() {
		key := e[:strings.IndexByte(e, '=')]
		if !snapVars[key] {
			env = append(env, e)
		}
	}
	return env
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
		if strings.Contains(args, "com.intellij.idea.Main") || isIDEABinary(args) {
			return pid
		}
	}

	return 0
}

// isIDEABinary checks if the cmdline looks like the IntelliJ IDEA native
// launcher installed by JetBrains Toolbox (e.g. .../intellij-idea-ultimate/bin/idea).
func isIDEABinary(cmdline string) bool {
	return strings.Contains(cmdline, "intellij-idea") && strings.HasSuffix(strings.SplitN(cmdline, "\x00", 2)[0], "/bin/idea")
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
