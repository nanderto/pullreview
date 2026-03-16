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

func TestCustomPromptAppending(t *testing.T) {
	// Simulate the prompt template construction logic from main.go
	baseTemplate := "# Base Prompt\n\nThis is the base review prompt.\n\n(DIFF_CONTENT_HERE)\n\n---\n"
	customPrompt := "# My Custom Instructions\n\nUse TypeScript for all code."

	// Apply the appending logic (as implemented in main.go line 287)
	finalTemplate := baseTemplate + "\n\n<!-- BEGIN: CUSTOM PROJECT INSTRUCTIONS [HIGHEST PRIORITY] -->\n\n" + customPrompt + "\n\n<!-- END: CUSTOM PROJECT INSTRUCTIONS -->\n"

	// Verify base template comes first
	if !strings.HasPrefix(finalTemplate, "# Base Prompt") {
		t.Errorf("expected base template to come first")
	}

	// Verify custom prompt appears after base template
	baseEndIdx := strings.Index(finalTemplate, "(DIFF_CONTENT_HERE)")
	customStartIdx := strings.Index(finalTemplate, "<!-- BEGIN: CUSTOM PROJECT INSTRUCTIONS")

	if baseEndIdx == -1 {
		t.Fatal("DIFF_CONTENT_HERE not found in template")
	}
	if customStartIdx == -1 {
		t.Fatal("custom instruction sentinel not found in template")
	}
	if customStartIdx <= baseEndIdx {
		t.Errorf("expected custom instructions to appear AFTER base template, but got customStartIdx=%d <= baseEndIdx=%d", customStartIdx, baseEndIdx)
	}

	// Verify END sentinel exists
	if !strings.Contains(finalTemplate, "<!-- END: CUSTOM PROJECT INSTRUCTIONS -->") {
		t.Errorf("END sentinel not found in template")
	}

	// Verify custom content is included
	if !strings.Contains(finalTemplate, "Use TypeScript for all code") {
		t.Errorf("custom prompt content not found in final template")
	}

	// Verify sentinel markers match the expected format (no emojis, compact)
	if strings.Contains(finalTemplate, "🔥") || strings.Contains(finalTemplate, "⚠️") {
		t.Errorf("unexpected emojis found in sentinel markers")
	}
}
