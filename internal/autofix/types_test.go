package autofix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplier_ApplyFixes_SingleLine(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := "test.go"
	originalContent := `package main

func hello() {
	fmt.Println("hello")
}
`
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create applier
	applier := NewApplier(tmpDir)

	// Apply fix
	fixes := []Fix{
		{
			File:         testFile,
			LineStart:    4,
			LineEnd:      4,
			OriginalCode: `fmt.Println("hello")`,
			FixedCode:    `fmt.Println("Hello, World!")`,
		},
	}

	modifiedFiles, err := applier.ApplyFixes(fixes)
	if err != nil {
		t.Fatalf("ApplyFixes failed: %v", err)
	}

	// Verify modified files
	if len(modifiedFiles) != 1 || modifiedFiles[0] != testFile {
		t.Errorf("expected [%s], got %v", testFile, modifiedFiles)
	}

	// Verify file contents
	content, err := os.ReadFile(filepath.Join(tmpDir, testFile))
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if !strings.Contains(string(content), `fmt.Println("Hello, World!")`) {
		t.Errorf("fix not applied correctly, got:\n%s", string(content))
	}
}

func TestApplier_ApplyFixes_MultiLine(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := "test.go"
	originalContent := `package main

func process(data string) {
	if data != "" {
		fmt.Println(data)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create applier
	applier := NewApplier(tmpDir)

	// Apply fix that replaces 3 lines
	fixes := []Fix{
		{
			File:      testFile,
			LineStart: 4,
			LineEnd:   6,
			OriginalCode: `if data != "" {
		fmt.Println(data)
	}`,
			FixedCode: `if data == "" {
		return
	}
	fmt.Println(data)`,
		},
	}

	modifiedFiles, err := applier.ApplyFixes(fixes)
	if err != nil {
		t.Fatalf("ApplyFixes failed: %v", err)
	}

	// Verify modified files
	if len(modifiedFiles) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(modifiedFiles))
	}

	// Verify file contents
	content, err := os.ReadFile(filepath.Join(tmpDir, testFile))
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if !strings.Contains(string(content), `if data == ""`) {
		t.Errorf("fix not applied correctly, got:\n%s", string(content))
	}
}

func TestApplier_ApplyFixes_MultipleFixes_SameFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := "test.go"
	originalContent := `package main

import "fmt"

func hello() {
	fmt.Println("hello")
}

func goodbye() {
	fmt.Println("goodbye")
}
`
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create applier
	applier := NewApplier(tmpDir)

	// Apply multiple fixes
	fixes := []Fix{
		{
			File:         testFile,
			LineStart:    6,
			LineEnd:      6,
			OriginalCode: `fmt.Println("hello")`,
			FixedCode:    `fmt.Println("Hello!")`,
		},
		{
			File:         testFile,
			LineStart:    10,
			LineEnd:      10,
			OriginalCode: `fmt.Println("goodbye")`,
			FixedCode:    `fmt.Println("Goodbye!")`,
		},
	}

	modifiedFiles, err := applier.ApplyFixes(fixes)
	if err != nil {
		t.Fatalf("ApplyFixes failed: %v", err)
	}

	// Verify modified files
	if len(modifiedFiles) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(modifiedFiles))
	}

	// Verify file contents
	content, err := os.ReadFile(filepath.Join(tmpDir, testFile))
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `fmt.Println("Hello!")`) {
		t.Errorf("first fix not applied correctly, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `fmt.Println("Goodbye!")`) {
		t.Errorf("second fix not applied correctly, got:\n%s", contentStr)
	}
}

func TestApplier_ApplyFixes_FileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	applier := NewApplier(tmpDir)

	fixes := []Fix{
		{
			File:         "nonexistent.go",
			OriginalCode: "something",
			FixedCode:    "test",
		},
	}

	_, err = applier.ApplyFixes(fixes)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestApplier_ApplyFixes_OriginalCodeNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := "test.go"
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	applier := NewApplier(tmpDir)

	fixes := []Fix{
		{
			File:         testFile,
			OriginalCode: "this code does not exist in the file",
			FixedCode:    "replacement",
		},
	}

	_, err = applier.ApplyFixes(fixes)
	if err == nil {
		t.Error("expected error when original code not found, got nil")
	}
}

func TestApplier_ApplyFixes_MissingOriginalCode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := "test.go"
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	applier := NewApplier(tmpDir)

	fixes := []Fix{
		{
			File:         testFile,
			OriginalCode: "", // Missing original code
			FixedCode:    "test",
		},
	}

	_, err = applier.ApplyFixes(fixes)
	if err == nil {
		t.Error("expected error for missing original code, got nil")
	}
}

func TestApplier_RestoreBackups(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autofix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := "test.go"
	originalContent := "original content"
	err = os.WriteFile(filepath.Join(tmpDir, testFile), []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create applier and apply fix
	applier := NewApplier(tmpDir)

	fixes := []Fix{
		{
			File:         testFile,
			LineStart:    1,
			LineEnd:      1,
			OriginalCode: "original content",
			FixedCode:    "modified content",
		},
	}

	_, err = applier.ApplyFixes(fixes)
	if err != nil {
		t.Fatalf("ApplyFixes failed: %v", err)
	}

	// Verify file was modified
	content, _ := os.ReadFile(filepath.Join(tmpDir, testFile))
	if string(content) != "modified content" {
		t.Errorf("expected modified content, got: %s", string(content))
	}

	// Restore backups
	err = applier.RestoreBackups()
	if err != nil {
		t.Fatalf("RestoreBackups failed: %v", err)
	}

	// Verify file was restored
	content, err = os.ReadFile(filepath.Join(tmpDir, testFile))
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != originalContent {
		t.Errorf("expected original content after restore, got: %s", string(content))
	}
}

func TestApplier_EmptyFixes(t *testing.T) {
	applier := NewApplier("/tmp")

	modifiedFiles, err := applier.ApplyFixes(nil)
	if err != nil {
		t.Errorf("unexpected error for empty fixes: %v", err)
	}
	if modifiedFiles != nil {
		t.Errorf("expected nil for empty fixes, got: %v", modifiedFiles)
	}

	modifiedFiles, err = applier.ApplyFixes([]Fix{})
	if err != nil {
		t.Errorf("unexpected error for empty fixes slice: %v", err)
	}
	if modifiedFiles != nil {
		t.Errorf("expected nil for empty fixes slice, got: %v", modifiedFiles)
	}
}

func TestGetLeadingWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", ""},
		{"  hello", "  "},
		{"\thello", "\t"},
		{"\t  hello", "\t  "},
		{"   ", "   "},
		{"", ""},
	}

	for _, tt := range tests {
		result := getLeadingWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("getLeadingWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
