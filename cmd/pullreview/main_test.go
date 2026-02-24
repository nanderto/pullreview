package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCmd creates a cobra command wired to the real runPullReview function
// with the same flags and arg validation as the production command.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "pullreview",
		Args: cobra.MaximumNArgs(1),
		RunE: runPullReview,
	}
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "")
	cmd.Flags().StringVar(&prID, "pr", "", "")
	cmd.Flags().StringVar(&bbEmail, "email", "", "")
	cmd.Flags().StringVar(&bbAPIToken, "token", "", "")
	cmd.Flags().StringVar(&repoSlug, "repo", "", "")
	cmd.Flags().BoolVar(&showVersion, "version", false, "")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "")
	cmd.Flags().BoolVar(&postToBB, "post", false, "")
	cmd.Flags().BoolVar(&skipInline, "skip-inline", false, "")
	cmd.Flags().BoolVar(&localReview, "local", false, "")
	cmd.Flags().StringVar(&targetBranch, "target", "main", "")
	return cmd
}

// resetAllFlags resets all package-level flag vars to defaults between tests.
func resetAllFlags() {
	cfgFile = ""
	prID = ""
	bbEmail = ""
	bbAPIToken = ""
	repoSlug = ""
	showVersion = false
	verbose = false
	postToBB = false
	skipInline = false
	localReview = false
	targetBranch = "main"
}

func TestFlagValidation_PositionalArgWithoutLocal(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"/some/path"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when positional arg provided without --local")
	}
	if !strings.Contains(err.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFlagValidation_PostWithLocal(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--local", "--post"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --post used with --local")
	}
	if !strings.Contains(err.Error(), "--post cannot be used with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFlagValidation_TooManyPositionalArgs(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--local", "/path1", "/path2"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error with two positional args")
	}
}

func TestFlagValidation_LocalWithPositionalArg(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	// Use a non-existent path — runPullReview will fail at git branch detection,
	// which proves the path was correctly passed through the validation stage.
	cmd.SetArgs([]string{"--local", "/nonexistent/repo/path"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
	// Should fail at git operations, not at flag validation
	if strings.Contains(err.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("should not have failed at flag validation: %v", err)
	}
}

func TestFlagValidation_LocalWithoutArgs(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	// --local without a path uses cwd; will fail at config loading or later,
	// but should pass flag validation.
	cmd.SetArgs([]string{"--local"})
	err := cmd.Execute()
	// May fail downstream (config, git, etc.) but not at flag validation
	if err != nil && strings.Contains(err.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("should not have failed at flag validation: %v", err)
	}
}

func TestFlagValidation_TargetFlag(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--local", "--target", "develop"})
	_ = cmd.Execute() // will fail downstream, we only care about flag parsing
	if targetBranch != "develop" {
		t.Errorf("expected targetBranch 'develop', got %q", targetBranch)
	}
}

func TestFlagValidation_TargetDefaultsToMain(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--local"})
	_ = cmd.Execute() // will fail downstream, we only care about flag parsing
	if targetBranch != "main" {
		t.Errorf("expected targetBranch 'main', got %q", targetBranch)
	}
}

func TestVersion(t *testing.T) {
	resetAllFlags()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--version should not error: %v", err)
	}
}
