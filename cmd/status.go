package cmd

import (
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/config"
	"github.com/begray/agh/internal/layout"
	"github.com/begray/agh/internal/project"
	"github.com/begray/agh/internal/session"
	"github.com/begray/agh/internal/worktree"
)

var statusCmd = &cobra.Command{
	Use:   "status [feature-name]",
	Short: "Show status of a feature or all features",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	var features []*project.Feature
	if len(args) == 1 {
		f, err := proj.LoadFeature(args[0])
		if err != nil {
			return fmt.Errorf("feature %q not found: %w", args[0], err)
		}
		features = []*project.Feature{f}
	} else {
		features, err = proj.ListFeatures()
		if err != nil {
			return err
		}
	}

	if len(features) == 0 {
		fmt.Println("No active features")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FEATURE\tLAYOUT\tAI SESSION\tIDE\tWORKTREE\tBRANCH")
	for _, f := range features {
		termStatus := sessionStatus(f, proj.Config)
		ideStatus := "-"
		if f.IDE != "" {
			if idePID, err := session.FindIDEProcess(f.Worktree); err == nil {
				ideStatus = fmt.Sprintf("running (pid %d)", idePID)
			} else {
				ideStatus = fmt.Sprintf("dead (pid %d)", f.IDEPID)
			}
		}
		wtStatus := worktreeStatus(f.Worktree)
		branchStatus := branchStatus(proj.RootDir, f)

		layoutType := f.Session.Type
		if layoutType == "" {
			layoutType = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			f.Name, layoutType, termStatus, ideStatus, wtStatus, branchStatus,
		)
	}
	w.Flush()
	return nil
}

func processStatus(pid int) string {
	if pid == 0 {
		return "not tracked"
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Sprintf("dead (pid %d)", pid)
	}
	// Signal 0 checks if process exists without actually sending a signal
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return fmt.Sprintf("dead (pid %d)", pid)
	}
	return fmt.Sprintf("running (pid %d)", pid)
}

func worktreeStatus(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	if !info.IsDir() {
		return "invalid"
	}
	return "ok"
}

func branchStatus(projectRoot string, f *project.Feature) string {
	if f.Worktree == "" {
		return "unknown"
	}
	branch, err := worktree.CurrentBranch(f.Worktree)
	if err != nil {
		return "detached/error"
	}
	if branch != f.Branch {
		return fmt.Sprintf("diverged (%s)", branch)
	}
	return branch
}

func sessionStatus(f *project.Feature, cfg config.Config) string {
	if f.Session.Type != "" {
		mgr, err := layout.NewForHandle(f.Session, cfg)
		if err != nil {
			return "error"
		}
		if mgr.IsAlive(f.Session) {
			if f.Session.PaneID != "" {
				return fmt.Sprintf("running (%s)", f.Session.PaneID)
			}
			return fmt.Sprintf("running (pid %d)", f.Session.PID)
		}
		return "dead"
	}
	// Legacy fallback
	return processStatus(f.TerminalPID)
}
