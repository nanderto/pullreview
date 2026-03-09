package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Operations handles git operations for the auto-fix workflow.
type Operations struct {
	repoPath string
}

// NewOperations creates a new git Operations instance.
func NewOperations(repoPath string) *Operations {
	return &Operations{
		repoPath: repoPath,
	}
}

// GetCurrentBranch returns the current git branch name.
func (g *Operations) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GenerateBranchName creates a timestamped branch name for fixes.
func (g *Operations) GenerateBranchName(sourceBranch, prefix string) string {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf("%s-%s-%s", prefix, sourceBranch, timestamp)
}

// CreateBranch creates a new git branch.
func (g *Operations) CreateBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = g.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}
	return nil
}

// Checkout checks out a git branch.
func (g *Operations) Checkout(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = g.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

// StageFiles stages specific files for commit.
func (g *Operations) StageFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}
	return nil
}

// Commit commits staged changes with a message.
func (g *Operations) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	return nil
}

// Push pushes a branch to remote.
func (g *Operations) Push(branchName string) error {
	cmd := exec.Command("git", "push", "origin", branchName)
	cmd.Dir = g.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}
	return nil
}
