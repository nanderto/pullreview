package utils

import (
	"bytes"
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
