package layout

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/begray/agh/internal/config"
)

// swayManager spawns a new terminal window and arranges it via swaymsg.
type swayManager struct {
	cfg      config.Config
	terminal *terminalManager
}

func newSwayManager(cfg config.Config) *swayManager {
	return &swayManager{
		cfg:      cfg,
		terminal: newTerminalManager(cfg),
	}
}

func (m *swayManager) Name() string { return "sway" }

func (m *swayManager) Start(feature, worktreeDir, shellCmd string) (SessionHandle, error) {
	handle, err := m.terminal.Start(feature, worktreeDir, shellCmd)
	if err != nil {
		return handle, err
	}

	m.arrange(feature)
	return handle, nil
}

func (m *swayManager) IsAlive(handle SessionHandle) bool {
	return m.terminal.IsAlive(handle)
}

func (m *swayManager) Kill(handle SessionHandle) error {
	return m.terminal.Kill(handle)
}

func (m *swayManager) arrange(feature string) {
	windowID := "agh-" + feature

	if !waitForSwayWindow(windowID, 5*time.Second) {
		fmt.Fprintf(os.Stderr, "warning: sway window %q did not appear within timeout\n", windowID)
		return
	}

	selector := swaySelector(windowID)

	moveCmd := exec.Command("swaymsg", fmt.Sprintf(`%s move right`, selector))
	if err := moveCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sway move failed: %v\n", err)
		return
	}

	layoutCmd := exec.Command("swaymsg", fmt.Sprintf(`%s layout stacking`, selector))
	if err := layoutCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sway layout failed: %v\n", err)
	}
}

// swaySelector returns the appropriate swaymsg selector for a window.
func swaySelector(windowID string) string {
	node := findSwayWindow(windowID)
	if node != nil && node.AppID == "" {
		return fmt.Sprintf(`[class="%s"]`, windowID)
	}
	return fmt.Sprintf(`[app_id="%s"]`, windowID)
}

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
