package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Create creates a new git worktree with a new branch.
// projectRoot is the main checkout, path is the target worktree directory,
// branch is the new branch name.
func Create(projectRoot, path, branch string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branch, path)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

// CreateFromRef creates a new git worktree based on a specific ref (branch/commit).
func CreateFromRef(projectRoot, path, branch, ref string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branch, path, ref)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add from ref: %w", err)
	}
	return nil
}

// Remove removes a git worktree.
func Remove(projectRoot, path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

// Diff runs git diff in the given worktree directory and returns the output.
func Diff(worktreePath string, args ...string) (string, error) {
	gitArgs := append([]string{"diff"}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = worktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %w\n%s", err, out)
	}
	return string(out), nil
}

// CurrentBranch returns the current branch name in the given directory.
func CurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// BranchExists checks if a local branch exists.
func BranchExists(projectRoot, branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/"+branch)
	cmd.Dir = projectRoot
	return cmd.Run() == nil
}

// CheckoutExisting creates a worktree for an existing branch (no -b).
func CheckoutExisting(projectRoot, path, branch string) error {
	cmd := exec.Command("git", "worktree", "add", path, branch)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add (existing branch): %w", err)
	}
	return nil
}

// DeleteBranch deletes a local branch.
func DeleteBranch(projectRoot, branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git branch -D: %w", err)
	}
	return nil
}
