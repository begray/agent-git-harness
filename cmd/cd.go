package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/begray/agh/internal/project"
	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:   "cd <feature-name>",
	Short: "Spawn a subshell in a feature's worktree directory",
	Long: `Opens a new shell session in the feature's worktree directory.
Use "exit" to return to your previous directory.

Use "agh cd -" to return to the project root directory.`,
	Args: cobra.ExactArgs(1),
	RunE: runCd,
}

func init() {
	rootCmd.AddCommand(cdCmd)
	cdCmd.ValidArgsFunction = completeFeatureNames
}

func runCd(cmd *cobra.Command, args []string) error {
	if os.Getenv("AGH_FEATURE") != "" {
		return fmt.Errorf("already in agh subshell for feature %q — use \"exit\" first", os.Getenv("AGH_FEATURE"))
	}

	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	target := args[0]

	var dir string
	if target == "-" {
		dir = proj.RootDir
	} else {
		feature, err := proj.LoadFeature(target)
		if err != nil {
			return fmt.Errorf("feature %q not found: %w", target, err)
		}
		dir = feature.Worktree
	}

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	featureName := target
	if target == "-" {
		featureName = "(root)"
	}

	fmt.Printf("Entering %s — use \"exit\" to return\n", dir)

	c := exec.Command(shell)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "AGH_FEATURE="+featureName)

	return c.Run()
}
