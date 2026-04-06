package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/layout"
	"github.com/begray/agh/internal/sandbox"
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

	// Launch all sessions (fresh start, no --continue)
	launchSessions(proj, feature, false)

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

	mgr, err := layout.New(proj.Config)
	if err != nil {
		return fmt.Errorf("creating layout manager: %w", err)
	}

	termAlive := mgr.IsAlive(feature.Session)
	ideAlive := feature.IDE == "" || session.IsIDEAlive(feature.Worktree)

	if termAlive && ideAlive {
		fmt.Println("All sessions already running")
		return nil
	}

	if !termAlive {
		launchAISession(proj, feature, mgr, true)
	} else {
		fmt.Println("AI session already running")
	}

	if !ideAlive {
		launchIDE(proj, feature)
	} else if feature.IDE != "" {
		fmt.Printf("IDE already running (pid %d)\n", feature.IDEPID)
	}

	return proj.SaveFeature(feature)
}

// launchSessions spawns the AI tool session and optionally an IDE.
func launchSessions(proj *project.Project, feature *project.Feature, resume bool) {
	mgr, err := layout.New(proj.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		return
	}
	launchAISession(proj, feature, mgr, resume)
	if feature.IDE != "" {
		launchIDE(proj, feature)
	}
}

func launchAISession(proj *project.Project, feature *project.Feature, mgr layout.Manager, resume bool) {
	shellCmd, err := layout.BuildShellCmd(proj.Config, resume)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		return
	}

	if proj.Config.Sandbox.Enabled {
		shellCmd, err = sandbox.WrapShellCmd(shellCmd, feature.Worktree, proj.RootDir, proj.Config.AITool, proj.Config.Sandbox)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: sandbox disabled: %v\n", err)
		} else {
			fmt.Println("Sandbox: enabled (greywall)")
		}
	}

	fmt.Printf("Launching %s via %s...\n", proj.Config.AITool, mgr.Name())
	handle, err := mgr.Start(feature.Name, feature.Worktree, shellCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to launch AI session: %v\n", err)
		return
	}
	feature.Session = handle
	feature.TerminalPID = handle.PID // backward compat
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
