package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestGetLocalDiff_WithChanges(t *testing.T) {
	// Set up a repo on main with initial commit, then create a feature branch with changes
	repoDir := setupTestRepo(t, "main", "")

	// Create a feature branch
	cmd := exec.Command("git", "checkout", "-b", "feature-branch")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create feature branch: %v\n%s", err, out)
	}

	// Add a new file and commit on the feature branch
	newFile := filepath.Join(repoDir, "feature.txt")
	if err := os.WriteFile(newFile, []byte("new feature\n"), 0644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	cmd = exec.Command("git", "add", "feature.txt")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "add feature")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git commit: %v\n%s", err, out)
	}

	diff, err := GetLocalDiff(repoDir, "main")
	if err != nil {
		t.Fatalf("GetLocalDiff failed: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff, got empty string")
	}
	if !strings.Contains(diff, "feature.txt") {
		t.Errorf("expected diff to contain 'feature.txt', got:\n%s", diff)
	}
}

func TestGetLocalDiff_NoChanges(t *testing.T) {
	// On main with no diverged branch — diff against self should be empty
	repoDir := setupTestRepo(t, "main", "")

	diff, err := GetLocalDiff(repoDir, "main")
	if err != nil {
		t.Fatalf("GetLocalDiff should not error on empty diff: %v", err)
	}
	if strings.TrimSpace(diff) != "" {
		t.Errorf("expected empty diff, got:\n%s", diff)
	}
}

func TestGetLocalDiff_InvalidTargetBranch(t *testing.T) {
	repoDir := setupTestRepo(t, "main", "")

	_, err := GetLocalDiff(repoDir, "nonexistent-branch")
	if err == nil {
		t.Error("expected error for nonexistent target branch, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-branch") {
		t.Errorf("expected error to mention branch name, got: %v", err)
	}
}

func TestGetLocalDiff_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := GetLocalDiff(dir, "main")
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestGetLocalDiff_CustomTargetBranch(t *testing.T) {
	// Set up repo, create a "develop" branch, then a feature branch off it
	repoDir := setupTestRepo(t, "main", "")

	// Create develop branch with an extra commit
	cmd := exec.Command("git", "checkout", "-b", "develop")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create develop branch: %v\n%s", err, out)
	}
	devFile := filepath.Join(repoDir, "develop.txt")
	if err := os.WriteFile(devFile, []byte("develop\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", "develop.txt")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "develop commit")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git commit: %v\n%s", err, out)
	}

	// Create feature branch off develop
	cmd = exec.Command("git", "checkout", "-b", "feature-from-develop")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create feature branch: %v\n%s", err, out)
	}
	featFile := filepath.Join(repoDir, "feature2.txt")
	if err := os.WriteFile(featFile, []byte("feature from develop\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", "feature2.txt")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "feature from develop")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to git commit: %v\n%s", err, out)
	}

	// Diff against develop should only show feature2.txt, not develop.txt
	diff, err := GetLocalDiff(repoDir, "develop")
	if err != nil {
		t.Fatalf("GetLocalDiff failed: %v", err)
	}
	if !strings.Contains(diff, "feature2.txt") {
		t.Errorf("expected diff to contain 'feature2.txt', got:\n%s", diff)
	}
	if strings.Contains(diff, "develop.txt") {
		t.Errorf("diff should not contain 'develop.txt' when diffing against develop, got:\n%s", diff)
	}
}

func TestLoadCustomPrompt_DotfileExists(t *testing.T) {
	dir := t.TempDir()
	content := "Custom prompt content"
	dotfile := filepath.Join(dir, ".pullreview-custom.md")
	if err := os.WriteFile(dotfile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}
}

func TestLoadCustomPrompt_RegularFileExists(t *testing.T) {
	dir := t.TempDir()
	content := "Regular file content"
	regularFile := filepath.Join(dir, "pullreview-custom.md")
	if err := os.WriteFile(regularFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write regular file: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}
}

func TestLoadCustomPrompt_DotfileTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	dotContent := "Dotfile content"
	regularContent := "Regular content"

	dotfile := filepath.Join(dir, ".pullreview-custom.md")
	if err := os.WriteFile(dotfile, []byte(dotContent), 0644); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}
	regularFile := filepath.Join(dir, "pullreview-custom.md")
	if err := os.WriteFile(regularFile, []byte(regularContent), 0644); err != nil {
		t.Fatalf("failed to write regular file: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	if got != dotContent {
		t.Errorf("expected dotfile content %q to take precedence, got %q", dotContent, got)
	}
}

func TestLoadCustomPrompt_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	dotfile := filepath.Join(dir, ".pullreview-custom.md")
	if err := os.WriteFile(dotfile, []byte("   \n\t  \n"), 0644); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	if got != "" {
		t.Errorf("expected empty string for whitespace-only file, got %q", got)
	}
}

func TestLoadCustomPrompt_NoFilesExist(t *testing.T) {
	dir := t.TempDir()

	got := LoadCustomPrompt(dir, false)
	if got != "" {
		t.Errorf("expected empty string when no files exist, got %q", got)
	}
}

func TestLoadCustomPrompt_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	content := "\n\n  Custom content with spaces  \n\n"
	expected := "Custom content with spaces"
	dotfile := filepath.Join(dir, ".pullreview-custom.md")
	if err := os.WriteFile(dotfile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLoadCustomPrompt_UnreadableFile(t *testing.T) {
	// Skip on Windows since file permissions work differently
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping file permission test on Windows")
	}

	dir := t.TempDir()
	dotfile := filepath.Join(dir, ".pullreview-custom.md")
	regularFile := filepath.Join(dir, "pullreview-custom.md")
	regularContent := "Fallback content"

	// Create unreadable dotfile (mode 0000)
	if err := os.WriteFile(dotfile, []byte("unreadable"), 0000); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}
	// Create readable regular file as fallback
	if err := os.WriteFile(regularFile, []byte(regularContent), 0644); err != nil {
		t.Fatalf("failed to write regular file: %v", err)
	}

	got := LoadCustomPrompt(dir, false)
	// Should fall back to regular file when dotfile is unreadable
	if got != regularContent {
		t.Errorf("expected fallback to regular file %q, got %q", regularContent, got)
	}

	// Clean up by making file deletable again
	os.Chmod(dotfile, 0644)
}

// Clean up any temp dirs created by tests (optional, since t.TempDir handles it)
func TestMain(m *testing.M) {
	code := m.Run()
	// No global cleanup needed
	os.Exit(code)
}
