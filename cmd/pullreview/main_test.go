package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCmd creates a minimal cobra command with the same flags and validation
// as the real root command, but uses a custom RunE to isolate flag validation.
func newTestCmd(runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "pullreview",
		Args: cobra.MaximumNArgs(1),
		RunE: runE,
	}
	cmd.Flags().BoolVar(&localReview, "local", false, "")
	cmd.Flags().BoolVar(&postToBB, "post", false, "")
	cmd.Flags().StringVar(&targetBranch, "target", "main", "")
	return cmd
}

// resetFlags resets package-level flag vars to defaults between tests.
func resetFlags() {
	localReview = false
	postToBB = false
	targetBranch = "main"
	showVersion = false
}

func TestFlagValidation_PositionalArgWithoutLocal(t *testing.T) {
	resetFlags()
	testCmd := newTestCmd(func(_ *cobra.Command, args []string) error {
		if !localReview && len(args) > 0 {
			return fmt.Errorf("positional arguments are only accepted with --local; did you mean: pullreview --local %s", args[0])
		}
		return nil
	})
	testCmd.SetArgs([]string{"/some/path"})
	gotErr := testCmd.Execute()
	if gotErr == nil {
		t.Fatal("expected error when positional arg provided without --local")
	}
	if !strings.Contains(gotErr.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("unexpected error message: %v", gotErr)
	}
}

func TestFlagValidation_PostWithLocal(t *testing.T) {
	resetFlags()
	testCmd := newTestCmd(func(_ *cobra.Command, _ []string) error {
		if localReview && postToBB {
			return fmt.Errorf("--post cannot be used with --local (no Bitbucket PR to post to)")
		}
		return nil
	})
	testCmd.SetArgs([]string{"--local", "--post"})
	gotErr := testCmd.Execute()
	if gotErr == nil {
		t.Fatal("expected error when --post used with --local")
	}
	if !strings.Contains(gotErr.Error(), "--post cannot be used with --local") {
		t.Errorf("unexpected error message: %v", gotErr)
	}
}

func TestFlagValidation_LocalWithPositionalArg(t *testing.T) {
	resetFlags()
	var gotArgs []string
	testCmd := newTestCmd(func(_ *cobra.Command, args []string) error {
		if !localReview && len(args) > 0 {
			return fmt.Errorf("positional arguments are only accepted with --local")
		}
		gotArgs = args
		return nil
	})
	testCmd.SetArgs([]string{"--local", "/some/path"})
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "/some/path" {
		t.Errorf("expected args [/some/path], got %v", gotArgs)
	}
	if !localReview {
		t.Error("expected localReview to be true")
	}
}

func TestFlagValidation_TooManyPositionalArgs(t *testing.T) {
	resetFlags()
	testCmd := newTestCmd(func(_ *cobra.Command, _ []string) error {
		return nil
	})
	testCmd.SetArgs([]string{"--local", "/path1", "/path2"})
	err := testCmd.Execute()
	if err == nil {
		t.Fatal("expected error with two positional args")
	}
}

func TestFlagValidation_LocalWithoutArgs(t *testing.T) {
	resetFlags()
	var gotArgs []string
	testCmd := newTestCmd(func(_ *cobra.Command, args []string) error {
		gotArgs = args
		return nil
	})
	testCmd.SetArgs([]string{"--local"})
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 0 {
		t.Errorf("expected no args, got %v", gotArgs)
	}
	if !localReview {
		t.Error("expected localReview to be true")
	}
}

func TestFlagValidation_TargetFlag(t *testing.T) {
	resetFlags()
	testCmd := newTestCmd(func(_ *cobra.Command, _ []string) error {
		return nil
	})
	testCmd.SetArgs([]string{"--local", "--target", "develop"})
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetBranch != "develop" {
		t.Errorf("expected targetBranch 'develop', got %q", targetBranch)
	}
}

func TestFlagValidation_TargetDefaultsToMain(t *testing.T) {
	resetFlags()
	testCmd := newTestCmd(func(_ *cobra.Command, _ []string) error {
		return nil
	})
	testCmd.SetArgs([]string{"--local"})
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetBranch != "main" {
		t.Errorf("expected targetBranch 'main', got %q", targetBranch)
	}
}
