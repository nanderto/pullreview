package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper to create a temporary git repo with a branch and remote
func setupTestRepo(t *testing.T, branchName, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo and make initial commit
	cmds := [][]string{
		{"git", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run %v: %v\n%s", args, err, out)
		}
	}

	// Create a file and commit it so the branch exists
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cmd := exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git commit: %v\n%s", err, out)
	}

	// Only create and checkout a new branch if it's not 'main'
	if branchName != "main" {
		cmd = exec.Command("git", "checkout", "-b", branchName)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to git checkout -b: %v\n%s", err, out)
		}
	}

	// Add remote if provided
	if remoteURL != "" {
		cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to add remote: %v\n%s", err, out)
		}
	}

	return dir
}

func TestGetCurrentGitBranch(t *testing.T) {
	branch := "test-branch"
	repoDir := setupTestRepo(t, branch, "")

	got, err := GetCurrentGitBranch(repoDir)
	if err != nil {
		t.Fatalf("GetCurrentGitBranch failed: %v", err)
	}
	if got != branch {
		t.Errorf("expected branch %q, got %q", branch, got)
	}
}

func TestGetRepoSlugFromGitRemote_HTTPS(t *testing.T) {
	repoSlug := "my-repo"
	remoteURL := "https://bitbucket.org/myteam/" + repoSlug + ".git"
	repoDir := setupTestRepo(t, "main", remoteURL)

	got, err := GetRepoSlugFromGitRemote(repoDir)
	if err != nil {
		t.Fatalf("GetRepoSlugFromGitRemote failed: %v", err)
	}
	if got != repoSlug {
		t.Errorf("expected repo slug %q, got %q", repoSlug, got)
	}
}

func TestGetRepoSlugFromGitRemote_SSH(t *testing.T) {
	repoSlug := "another-repo"
	remoteURL := "git@bitbucket.org:myteam/" + repoSlug + ".git"
	repoDir := setupTestRepo(t, "main", remoteURL)

	got, err := GetRepoSlugFromGitRemote(repoDir)
	if err != nil {
		t.Fatalf("GetRepoSlugFromGitRemote failed: %v", err)
	}
	if got != repoSlug {
		t.Errorf("expected repo slug %q, got %q", repoSlug, got)
	}
}

func TestGetRepoSlugFromGitRemote_NoGit(t *testing.T) {
	// Directory that is not a git repo
	dir := t.TempDir()
	_, err := GetRepoSlugFromGitRemote(dir)
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestGetRepoSlugFromGitRemote_WeirdURL(t *testing.T) {
	repoSlug := "strange-repo"
	remoteURL := "ssh://git@bitbucket.org/myteam/" + repoSlug + ".git"
	repoDir := setupTestRepo(t, "main", remoteURL)

	got, err := GetRepoSlugFromGitRemote(repoDir)
	if err != nil {
		t.Fatalf("GetRepoSlugFromGitRemote failed: %v", err)
	}
	if got != repoSlug {
		t.Errorf("expected repo slug %q, got %q", repoSlug, got)
	}
}

// Clean up any temp dirs created by tests (optional, since t.TempDir handles it)
func TestMain(m *testing.M) {
	code := m.Run()
	// No global cleanup needed
	os.Exit(code)
}
