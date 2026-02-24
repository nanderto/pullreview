package main

import (
	"strings"
	"testing"
)

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

func TestValidateFlags_PositionalArgWithoutLocal(t *testing.T) {
	resetAllFlags()
	err := validateFlags([]string{"/some/path"})
	if err == nil {
		t.Fatal("expected error when positional arg provided without --local")
	}
	if !strings.Contains(err.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFlags_PostWithLocal(t *testing.T) {
	resetAllFlags()
	localReview = true
	postToBB = true
	err := validateFlags(nil)
	if err == nil {
		t.Fatal("expected error when --post used with --local")
	}
	if !strings.Contains(err.Error(), "--post cannot be used with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFlags_LocalWithPositionalArg(t *testing.T) {
	resetAllFlags()
	localReview = true
	err := validateFlags([]string{"/some/path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_LocalWithoutArgs(t *testing.T) {
	resetAllFlags()
	localReview = true
	err := validateFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_NoFlagsNoArgs(t *testing.T) {
	resetAllFlags()
	err := validateFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_PostWithoutLocal(t *testing.T) {
	resetAllFlags()
	postToBB = true
	err := validateFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
