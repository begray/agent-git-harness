package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
	"github.com/begray/agh/internal/session"
	"github.com/begray/agh/internal/worktree"
)

var (
	deleteBranch bool
)

var stopCmd = &cobra.Command{
	Use:   "stop <feature-name>",
	Short: "Stop a feature: close sessions, remove worktree",
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().BoolVar(&deleteBranch, "delete-branch", false, "Also delete the local branch")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	featureName := args[0]

	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	feature, err := proj.LoadFeature(featureName)
	if err != nil {
		return fmt.Errorf("feature %q not found: %w", featureName, err)
	}

	// Kill terminal process
	if feature.TerminalPID != 0 {
		fmt.Printf("Stopping terminal (PID %d)...\n", feature.TerminalPID)
		session.KillProcess(feature.TerminalPID)
	}

	// Kill IDE process: find by worktree path since the "idea" launcher
	// exits immediately and the stored PID goes stale.
	if feature.IDE != "" {
		if idePID, err := session.FindIDEProcess(feature.Worktree); err == nil {
			fmt.Printf("Stopping IDE (PID %d)...\n", idePID)
			session.KillProcess(idePID)
		} else if feature.IDEPID != 0 {
			// Fall back to stored PID
			fmt.Printf("Stopping IDE (PID %d)...\n", feature.IDEPID)
			session.KillProcess(feature.IDEPID)
		}
	}

	// Remove worktree
	fmt.Printf("Removing worktree %s...\n", feature.Worktree)
	if err := worktree.Remove(proj.RootDir, feature.Worktree); err != nil {
		fmt.Printf("warning: %v\n", err)
	}

	// Optionally delete branch
	if deleteBranch {
		fmt.Printf("Deleting branch %s...\n", feature.Branch)
		if err := worktree.DeleteBranch(proj.RootDir, feature.Branch); err != nil {
			fmt.Printf("warning: %v\n", err)
		}
	}

	// Remove feature state
	if err := proj.RemoveFeature(featureName); err != nil {
		return fmt.Errorf("removing feature state: %w", err)
	}

	fmt.Printf("Feature %q stopped\n", featureName)
	return nil
}
