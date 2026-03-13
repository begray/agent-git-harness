package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
)

var execCmd = &cobra.Command{
	Use:   "exec <feature-name> -- <command> [args...]",
	Short: "Run a command in a feature's worktree directory",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	command := args[1:]

	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	feature, err := proj.LoadFeature(featureName)
	if err != nil {
		return fmt.Errorf("feature %q not found: %w", featureName, err)
	}

	c := exec.Command(command[0], command[1:]...)
	c.Dir = feature.Worktree
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	return c.Run()
}
