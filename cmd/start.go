package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
	"github.com/begray/agh/internal/session"
	"github.com/begray/agh/internal/worktree"
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

	// If feature is already tracked, resume dead sessions instead of failing
	if existing, err := proj.LoadFeature(featureName); err == nil {
		return resumeFeature(proj, existing)
	}

	wtPath := proj.WorktreePath(featureName)
	branchExists := worktree.BranchExists(proj.RootDir, featureName)

	// If branch exists, check if it's already checked out in a worktree
	// (possibly at a non-standard path)
	if branchExists {
		if existingWt := worktree.FindWorktreeForBranch(proj.RootDir, featureName); existingWt != "" {
			wtPath = existingWt
		}
	}

	_, wtErr := os.Stat(wtPath)
	worktreeExists := wtErr == nil

	// Determine base branch and parent feature context
	cwd, _ := os.Getwd()
	var parentFeature string
	var baseBranch string

	baseBranch, err = worktree.CurrentBranch(cwd)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if cwd != proj.RootDir {
		parentFeature = findFeatureByWorktree(proj, cwd)
	}

	switch {
	case worktreeExists && branchExists:
		// Attach to existing worktree and branch
		fmt.Printf("Attaching to existing worktree %s (branch: %s)\n", wtPath, featureName)
		actualBranch, err := worktree.CurrentBranch(wtPath)
		if err != nil {
			return fmt.Errorf("worktree at %s exists but is not a valid git directory: %w", wtPath, err)
		}
		baseBranch = actualBranch

	case branchExists && !worktreeExists:
		// Create worktree for existing branch
		fmt.Printf("Creating worktree %s for existing branch %s\n", wtPath, featureName)
		if err := worktree.CheckoutExisting(proj.RootDir, wtPath, featureName); err != nil {
			return err
		}
		baseBranch = featureName

	default:
		// Create new branch and worktree
		fmt.Printf("Creating worktree %s (branch: %s, base: %s)\n", wtPath, featureName, baseBranch)
		if parentFeature != "" {
			if err := worktree.CreateFromRef(proj.RootDir, wtPath, featureName, baseBranch); err != nil {
				return err
			}
		} else {
			if err := worktree.Create(proj.RootDir, wtPath, featureName); err != nil {
				return err
			}
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

	// Launch all sessions
	launchSessions(proj, feature)

	if err := proj.SaveFeature(feature); err != nil {
		return fmt.Errorf("saving feature state: %w", err)
	}

	fmt.Printf("Feature %q started successfully\n", featureName)
	if parentFeature != "" {
		fmt.Printf("  Based on feature: %s\n", parentFeature)
	}
	return nil
}

// resumeFeature checks which sessions are dead and respawns them.
func resumeFeature(proj *project.Project, feature *project.Feature) error {
	fmt.Printf("Resuming feature %q\n", feature.Name)

	termAlive := session.IsProcessAlive(feature.TerminalPID)
	ideAlive := feature.IDE == "" || session.IsProcessAlive(feature.IDEPID)

	if termAlive && ideAlive {
		fmt.Println("All sessions already running")
		return nil
	}

	if !termAlive {
		launchTerminal(proj, feature)
	} else {
		fmt.Printf("Terminal already running (pid %d)\n", feature.TerminalPID)
	}

	if !ideAlive {
		launchIDE(proj, feature)
	} else if feature.IDE != "" {
		fmt.Printf("IDE already running (pid %d)\n", feature.IDEPID)
	}

	return proj.SaveFeature(feature)
}

// launchSessions spawns terminal, arranges sway, and launches IDE.
func launchSessions(proj *project.Project, feature *project.Feature) {
	launchTerminal(proj, feature)
	if feature.IDE != "" {
		launchIDE(proj, feature)
	}
}

func launchTerminal(proj *project.Project, feature *project.Feature) {
	terminal, err := proj.Config.ResolveTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		return
	}
	fmt.Printf("Launching %s in %s terminal...\n", proj.Config.AITool, terminal)
	termPID, err := session.SpawnTerminal(proj.Config, feature.Name, feature.Worktree)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to launch terminal: %v\n", err)
		return
	}
	feature.TerminalPID = termPID

	if proj.Config.Sway.Enabled {
		session.ArrangeSway(proj.Config, feature.Name)
	}
}

func launchIDE(proj *project.Project, feature *project.Feature) {
	ide := proj.DetectIDE()
	if ide == "" {
		return
	}
	feature.IDE = ide
	fmt.Printf("Launching %s...\n", ide)
	idePID, err := session.SpawnIDE(feature.Worktree)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to launch IDE: %v\n", err)
		return
	}
	feature.IDEPID = idePID
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
