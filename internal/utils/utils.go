package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// GetCurrentGitBranch returns the name of the current git branch in the given directory.
// Returns an empty string and error if not in a git repo or on failure.
func GetCurrentGitBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out.String())
	return branch, nil
}

// GetRepoSlugFromGitRemote returns the Bitbucket repo slug by parsing the 'origin' remote URL.
// It supports both HTTPS and SSH remote formats.
// Returns the repo slug (e.g., "bdirect-notifications") or an error if it cannot be determined.
func GetRepoSlugFromGitRemote(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	url := strings.TrimSpace(out.String())

	// Patterns:
	// HTTPS: https://bitbucket.org/workspace/repo_slug.git
	// SSH:   git@bitbucket.org:workspace/repo_slug.git
	// We want to extract the last path component, minus ".git"
	re := regexp.MustCompile(`[:/](?P<workspace>[^/]+)/(?P<repo>[^/]+?)(\.git)?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 3 {
		repoSlug := matches[2]
		repoSlug = strings.TrimSuffix(repoSlug, ".git")
		return repoSlug, nil
	}

	// Fallback: try to use path.Base
	base := path.Base(url)
	repoSlug := strings.TrimSuffix(base, ".git")
	if repoSlug != "" && repoSlug != "." && repoSlug != "/" {
		return repoSlug, nil
	}

	return "", err
}

// PromptYesNo prompts the user with a yes/no question and returns true if yes, false otherwise.
// The defaultAnswer parameter determines what happens on empty input ("y" or "n").
func PromptYesNo(question string, defaultAnswer string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	defaultAnswer = strings.ToLower(defaultAnswer)

	// Display prompt with default indicator
	prompt := question
	if defaultAnswer == "y" {
		prompt += " [Y/n]: "
	} else {
		prompt += " [y/N]: "
	}
	fmt.Print(prompt)

	// Read user input
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	// Trim whitespace and normalize
	input = strings.TrimSpace(strings.ToLower(input))

	// Handle empty input (use default)
	if input == "" {
		return defaultAnswer == "y", nil
	}

	// Check for yes/no
	if input == "y" || input == "yes" {
		return true, nil
	}
	if input == "n" || input == "no" {
		return false, nil
	}

	// Invalid input: treat as "no" (safer default)
	return false, nil
}
