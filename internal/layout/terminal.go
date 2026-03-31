package layout

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/begray/agh/internal/config"
)

// terminalManager spawns a new terminal window. Used as fallback when no
// multiplexer is available, and internally by swayManager.
type terminalManager struct {
	cfg config.Config
}

func newTerminalManager(cfg config.Config) *terminalManager {
	return &terminalManager{cfg: cfg}
}

func (m *terminalManager) Name() string { return "terminal" }

func (m *terminalManager) Start(feature, worktreeDir, shellCmd string) (SessionHandle, error) {
	termCmd, termArgs, err := m.cfg.TerminalArgs(feature, worktreeDir)
	if err != nil {
		return SessionHandle{}, err
	}

	args := append(termArgs, "bash", "-c", shellCmd)

	cmd := exec.Command(termCmd, args...)
	cmd.Dir = worktreeDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return SessionHandle{}, fmt.Errorf("starting terminal: %w", err)
	}

	go cmd.Wait()

	return SessionHandle{
		Type: "pid",
		PID:  cmd.Process.Pid,
	}, nil
}

func (m *terminalManager) IsAlive(handle SessionHandle) bool {
	return isProcessAlive(handle.PID)
}

func (m *terminalManager) Kill(handle SessionHandle) error {
	return killProcess(handle.PID)
}

// shellQuote wraps a string in single quotes for safe shell usage.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildShellCmd constructs the shell command string for running an AI tool.
// On exit, falls back to an interactive shell.
func BuildShellCmd(cfg config.Config, resume bool) (string, error) {
	aiCmd, aiArgs, err := cfg.AIToolArgs(resume)
	if err != nil {
		return "", err
	}

	shellCmd := aiCmd
	for _, a := range aiArgs {
		shellCmd += " " + shellQuote(a)
	}
	if resume {
		baseCmd := aiCmd
		baseArgs, _ := cfg.AIToolBaseArgs()
		for _, a := range baseArgs {
			baseCmd += " " + shellQuote(a)
		}
		shellCmd += " || " + baseCmd
	}
	shellCmd += "; exec $SHELL"
	return shellCmd, nil
}

// isProcessAlive checks if a process with the given PID is running.
func isProcessAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// killProcess sends SIGTERM to a process by PID.
func killProcess(pid int) error {
	if pid == 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	return nil
}
