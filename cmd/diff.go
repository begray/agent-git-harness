package cmd

import (
	"fmt"

	"github.com/begray/agh/internal/project"
	"github.com/begray/agh/internal/worktree"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <feature-name> [-- git-diff-args...]",
	Short: "Show git diff for a feature worktree",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	extraArgs := args[1:]

	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	feature, err := proj.LoadFeature(featureName)
	if err != nil {
		return fmt.Errorf("feature %q not found: %w", featureName, err)
	}

	return worktree.Diff(feature.Worktree, extraArgs...)
}
