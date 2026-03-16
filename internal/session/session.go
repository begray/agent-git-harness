package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/begray/agh/internal/config"
)

// shellQuote wraps a string in single quotes for safe shell usage.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// SpawnTerminal launches a terminal window running the AI tool in the given directory.
// When resume is true, the AI tool is launched with resume args (e.g. --continue for claude).
// Returns the PID of the terminal process.
func SpawnTerminal(cfg config.Config, feature, worktreeDir string, resume bool) (int, error) {
	termCmd, termArgs, err := cfg.TerminalArgs(feature, worktreeDir)
	if err != nil {
		return 0, err
	}

	aiCmd, aiArgs, err := cfg.AIToolArgs(resume)
	if err != nil {
		return 0, err
	}

	// Build shell command that runs the AI tool and falls back to a shell on exit.
	// When resuming, try with resume args first, fall back to plain invocation.
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

	// Terminal args + shell invocation
	args := append(termArgs, "bash", "-c", shellCmd)

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
	node := findSwayWindow(windowID)
	if node != nil && node.AppID == "" {
		return fmt.Sprintf(`[class="%s"]`, windowID)
	}
	return fmt.Sprintf(`[app_id="%s"]`, windowID)
}

// waitForSwayWindow polls sway's tree until a window with the given id appears.
func waitForSwayWindow(windowID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if findSwayWindow(windowID) != nil {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

type swayNode struct {
	AppID            string           `json:"app_id"`
	WindowProperties *swayWindowProps `json:"window_properties"`
	Nodes            []swayNode       `json:"nodes"`
	FloatingNodes    []swayNode       `json:"floating_nodes"`
}

type swayWindowProps struct {
	Class string `json:"class"`
}

// findSwayWindow searches the sway tree for a window matching the given id
// by either app_id (Wayland native) or window_properties.class (XWayland).
func findSwayWindow(windowID string) *swayNode {
	cmd := exec.Command("swaymsg", "-t", "get_tree")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var root swayNode
	if err := json.Unmarshal(out, &root); err != nil {
		return nil
	}
	return walkSwayTree(&root, windowID)
}

func walkSwayTree(node *swayNode, windowID string) *swayNode {
	if node.AppID == windowID {
		return node
	}
	if node.WindowProperties != nil && node.WindowProperties.Class == windowID {
		return node
	}
	for i := range node.Nodes {
		if found := walkSwayTree(&node.Nodes[i], windowID); found != nil {
			return found
		}
	}
	for i := range node.FloatingNodes {
		if found := walkSwayTree(&node.FloatingNodes[i], windowID); found != nil {
			return found
		}
	}
	return nil
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
