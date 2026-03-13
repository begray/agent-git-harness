package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vvinogradov/agh/internal/project"
	"github.com/vvinogradov/agh/internal/session"
	"github.com/vvinogradov/agh/internal/worktree"
)

var startCmd = &cobra.Command{
	Use:   "start <feature-name>",
	Short: "Start a new feature: create branch, worktree, and launch AI session",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	featureName := args[0]

	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	if err := proj.InitAghDir(); err != nil {
		return err
	}

	// Check if feature already exists
	if existing, err := proj.LoadFeature(featureName); err == nil {
		return fmt.Errorf("feature %q already exists (worktree: %s)", existing.Name, existing.Worktree)
	}

	wtPath := proj.WorktreePath(featureName)

	// Determine base: if we're in a worktree, that's a parent feature
	cwd, _ := os.Getwd()
	var parentFeature string
	var baseBranch string

	baseBranch, err = worktree.CurrentBranch(cwd)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	// If cwd is different from project root, we're in a worktree (parent feature)
	if cwd != proj.RootDir {
		parentFeature = findFeatureByWorktree(proj, cwd)
	}

	// Create worktree with new branch from current HEAD
	fmt.Printf("Creating worktree %s (branch: %s, base: %s)\n", wtPath, featureName, baseBranch)
	if parentFeature != "" {
		// Branch from the parent feature's current HEAD
		if err := worktree.CreateFromRef(proj.RootDir, wtPath, featureName, baseBranch); err != nil {
			return err
		}
	} else {
		if err := worktree.Create(proj.RootDir, wtPath, featureName); err != nil {
			return err
		}
	}

	// Auto-detect IDE
	ide := proj.DetectIDE()

	feature := &project.Feature{
		Name:          featureName,
		Branch:        featureName,
		Worktree:      wtPath,
		BaseBranch:    baseBranch,
		ParentFeature: parentFeature,
		CreatedAt:     time.Now(),
		IDE:           ide,
		AITool:        proj.Config.AITool,
	}

	// Spawn terminal with AI tool
	fmt.Printf("Launching %s in %s terminal...\n", proj.Config.AITool, proj.Config.Terminal)
	termPID, err := session.SpawnTerminal(proj.Config, featureName, wtPath)
	if err != nil {
		return fmt.Errorf("spawning terminal: %w", err)
	}
	feature.TerminalPID = termPID

	// Arrange in sway if enabled
	if proj.Config.Sway.Enabled {
		// Give the terminal a moment to create its window
		time.Sleep(500 * time.Millisecond)
		session.ArrangeSway(proj.Config, featureName)
	}

	// Launch IDE if detected
	if ide != "" {
		fmt.Printf("Launching %s...\n", ide)
		idePID, err := session.SpawnIDE(wtPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to launch IDE: %v\n", err)
		} else {
			feature.IDEPID = idePID
		}
	}

	if err := proj.SaveFeature(feature); err != nil {
		return fmt.Errorf("saving feature state: %w", err)
	}

	fmt.Printf("Feature %q started successfully\n", featureName)
	if parentFeature != "" {
		fmt.Printf("  Based on feature: %s\n", parentFeature)
	}
	return nil
}

func findFeatureByWorktree(proj *project.Project, dir string) string {
	features, err := proj.ListFeatures()
	if err != nil {
		return ""
	}
	for _, f := range features {
		if f.Worktree == dir {
			return f.Name
		}
	}
	return ""
}
