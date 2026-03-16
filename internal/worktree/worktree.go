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

// Diff runs git diff in the given worktree directory, inheriting stdout/stderr
// so that color and pager work as if the user ran git diff directly.
func Diff(worktreePath string, args ...string) error {
	gitArgs := append([]string{"diff"}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// FindWorktreeForBranch returns the worktree path where a branch is checked out,
// or empty string if the branch is not checked out in any worktree.
func FindWorktreeForBranch(projectRoot, branch string) string {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	var currentPath string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		}
		if strings.TrimSpace(line) == "branch refs/heads/"+branch {
			return currentPath
		}
	}
	return ""
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
