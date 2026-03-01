package main

import (
	"strings"
	"testing"
)

func TestValidateFlags_PositionalArgWithoutLocal(t *testing.T) {
	err := validateFlags(false, false, "", []string{"/some/path"})
	if err == nil {
		t.Fatal("expected error when positional arg provided without --local")
	}
	if !strings.Contains(err.Error(), "positional arguments are only accepted with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFlags_PostWithLocal(t *testing.T) {
	err := validateFlags(true, true, "", nil)
	if err == nil {
		t.Fatal("expected error when --post used with --local")
	}
	if !strings.Contains(err.Error(), "--post cannot be used with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFlags_PRWithLocal(t *testing.T) {
	err := validateFlags(true, false, "123", nil)
	if err == nil {
		t.Fatal("expected error when --pr used with --local")
	}
	if !strings.Contains(err.Error(), "--pr cannot be used with --local") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateFlags_LocalWithPositionalArg(t *testing.T) {
	err := validateFlags(true, false, "", []string{"/some/path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_LocalWithoutArgs(t *testing.T) {
	err := validateFlags(true, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_NoFlagsNoArgs(t *testing.T) {
	err := validateFlags(false, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_PostWithoutLocal(t *testing.T) {
	err := validateFlags(false, true, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_PRWithoutLocal(t *testing.T) {
	err := validateFlags(false, false, "123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlags_PositionalArgHintIncludesPath(t *testing.T) {
	err := validateFlags(false, false, "", []string{"/my/repo"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "/my/repo") {
		t.Errorf("error should include the provided path, got: %v", err)
	}
}
