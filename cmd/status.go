package cmd

import (
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vvinogradov/agh/internal/project"
	"github.com/vvinogradov/agh/internal/worktree"
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
	fmt.Fprintln(w, "FEATURE\tTERMINAL\tIDE\tWORKTREE\tBRANCH")
	for _, f := range features {
		termStatus := processStatus(f.TerminalPID)
		ideStatus := "-"
		if f.IDE != "" {
			ideStatus = processStatus(f.IDEPID)
		}
		wtStatus := worktreeStatus(f.Worktree)
		branchStatus := branchStatus(proj.RootDir, f)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			f.Name, termStatus, ideStatus, wtStatus, branchStatus,
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
