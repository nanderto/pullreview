package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguages_GoOnly(t *testing.T) {
	// Create temporary directory with Go files
	tempDir := t.TempDir()

	// Create go.mod (config file)
	createFile(t, tempDir, "go.mod", "module test\n\ngo 1.21\n")

	// Create Go source files
	createFile(t, tempDir, "main.go", "package main\n\nfunc main() {}\n")
	createFile(t, tempDir, "util.go", "package main\n\nfunc helper() {}\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) != 1 {
		t.Errorf("Expected 1 language, got %d: %v", len(languages), languages)
	}

	if languages[0] != "go" {
		t.Errorf("Expected 'go', got '%s'", languages[0])
	}
}

func TestDetectLanguages_Python(t *testing.T) {
	tempDir := t.TempDir()

	// Create Python config file
	createFile(t, tempDir, "requirements.txt", "requests==2.28.0\n")

	// Create Python source files
	createFile(t, tempDir, "main.py", "def main():\n    pass\n")
	createFile(t, tempDir, "utils.py", "def helper():\n    pass\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) != 1 {
		t.Errorf("Expected 1 language, got %d: %v", len(languages), languages)
	}

	if languages[0] != "python" {
		t.Errorf("Expected 'python', got '%s'", languages[0])
	}
}

func TestDetectLanguages_JavaScript(t *testing.T) {
	tempDir := t.TempDir()

	// Create package.json (config file)
	createFile(t, tempDir, "package.json", `{
		"name": "test",
		"version": "1.0.0",
		"dependencies": {
			"express": "^4.18.0"
		}
	}`)

	// Create JavaScript source files
	createFile(t, tempDir, "index.js", "console.log('hello');\n")
	createFile(t, tempDir, "app.js", "const express = require('express');\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) == 0 {
		t.Fatal("Expected at least 1 language, got 0")
	}

	if languages[0] != "javascript" {
		t.Errorf("Expected 'javascript', got '%s'", languages[0])
	}
}

func TestDetectLanguages_TypeScript(t *testing.T) {
	tempDir := t.TempDir()

	// Create tsconfig.json (config file)
	createFile(t, tempDir, "tsconfig.json", `{
		"compilerOptions": {
			"target": "ES2020",
			"module": "commonjs"
		}
	}`)

	// Create package.json with TypeScript dependency
	createFile(t, tempDir, "package.json", `{
		"name": "test",
		"devDependencies": {
			"typescript": "^5.0.0"
		}
	}`)

	// Create TypeScript source files
	createFile(t, tempDir, "index.ts", "const x: number = 42;\n")
	createFile(t, tempDir, "app.ts", "interface User { name: string; }\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) == 0 {
		t.Fatal("Expected at least 1 language, got 0")
	}

	// TypeScript should be detected
	hasTypeScript := false
	for _, lang := range languages {
		if lang == "typescript" {
			hasTypeScript = true
			break
		}
	}

	if !hasTypeScript {
		t.Errorf("Expected 'typescript' in languages, got: %v", languages)
	}
}

func TestDetectLanguages_MultiLanguage(t *testing.T) {
	tempDir := t.TempDir()

	// Create Go files
	createFile(t, tempDir, "go.mod", "module test\n\ngo 1.21\n")
	createFile(t, tempDir, "main.go", "package main\n")
	createFile(t, tempDir, "util.go", "package main\n")
	createFile(t, tempDir, "helper.go", "package main\n")

	// Create Python files
	createFile(t, tempDir, "requirements.txt", "requests==2.28.0\n")
	createFile(t, tempDir, "script.py", "import sys\n")
	createFile(t, tempDir, "utils.py", "def helper():\n    pass\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) != 2 {
		t.Errorf("Expected 2 languages, got %d: %v", len(languages), languages)
	}

	// Check both languages are present
	hasGo := false
	hasPython := false
	for _, lang := range languages {
		if lang == "go" {
			hasGo = true
		}
		if lang == "python" {
			hasPython = true
		}
	}

	if !hasGo {
		t.Error("Expected 'go' in languages")
	}
	if !hasPython {
		t.Error("Expected 'python' in languages")
	}
}

func TestDetectLanguages_NoLanguages(t *testing.T) {
	tempDir := t.TempDir()

	// Create only non-code files
	createFile(t, tempDir, "README.md", "# Test Project\n")
	createFile(t, tempDir, "config.txt", "some config\n")

	_, err := DetectLanguages(tempDir)
	if err == nil {
		t.Error("Expected error for no languages detected, got nil")
	}

	if err != nil && err.Error() != "no recognized programming languages detected in repository" {
		t.Errorf("Expected 'no recognized programming languages' error, got: %v", err)
	}
}

func TestDetectLanguages_IgnoresVendor(t *testing.T) {
	tempDir := t.TempDir()

	// Create Go files in main directory
	createFile(t, tempDir, "go.mod", "module test\n\ngo 1.21\n")
	createFile(t, tempDir, "main.go", "package main\n")

	// Create vendor directory with Python files (should be ignored)
	vendorDir := filepath.Join(tempDir, "vendor")
	os.Mkdir(vendorDir, 0755)
	createFile(t, vendorDir, "vendored.py", "import sys\n")
	createFile(t, vendorDir, "lib.py", "def helper(): pass\n")
	createFile(t, vendorDir, "util.py", "x = 1\n")
	createFile(t, vendorDir, "module.py", "y = 2\n")
	createFile(t, vendorDir, "script.py", "z = 3\n")
	createFile(t, vendorDir, "extra.py", "a = 4\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	if len(languages) != 1 {
		t.Errorf("Expected 1 language (vendor should be ignored), got %d: %v", len(languages), languages)
	}

	if languages[0] != "go" {
		t.Errorf("Expected 'go', got '%s'", languages[0])
	}

	// Ensure Python was not detected (vendor was ignored)
	for _, lang := range languages {
		if lang == "python" {
			t.Error("Python should not be detected (vendor directory should be ignored)")
		}
	}
}

func TestDetectLanguages_IgnoresNodeModules(t *testing.T) {
	tempDir := t.TempDir()

	// Create JavaScript files
	createFile(t, tempDir, "package.json", `{"name": "test"}`)
	createFile(t, tempDir, "index.js", "console.log('main');\n")

	// Create node_modules with TypeScript files (should be ignored)
	nodeModulesDir := filepath.Join(tempDir, "node_modules")
	os.Mkdir(nodeModulesDir, 0755)
	createFile(t, nodeModulesDir, "lib.ts", "const x: number = 1;\n")
	createFile(t, nodeModulesDir, "util.ts", "interface User {}\n")
	createFile(t, nodeModulesDir, "helper.ts", "type Id = string;\n")
	createFile(t, nodeModulesDir, "module.ts", "enum Status {}\n")
	createFile(t, nodeModulesDir, "extra.ts", "class Test {}\n")
	createFile(t, nodeModulesDir, "more.ts", "function test() {}\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Should only detect JavaScript
	if len(languages) != 1 {
		t.Errorf("Expected 1 language, got %d: %v", len(languages), languages)
	}

	if languages[0] != "javascript" {
		t.Errorf("Expected 'javascript', got '%s'", languages[0])
	}
}

func TestDetectLanguages_ConfigFileOverride(t *testing.T) {
	tempDir := t.TempDir()

	// Create go.mod (config file) but only 2 Go files (below threshold of 5)
	createFile(t, tempDir, "go.mod", "module test\n\ngo 1.21\n")
	createFile(t, tempDir, "main.go", "package main\n")
	createFile(t, tempDir, "util.go", "package main\n")

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Should still detect Go because of config file (overrides threshold)
	if len(languages) != 1 {
		t.Errorf("Expected 1 language, got %d: %v", len(languages), languages)
	}

	if languages[0] != "go" {
		t.Errorf("Expected 'go' (config file should override threshold), got '%s'", languages[0])
	}
}

func TestDetectLanguages_MinimumThreshold(t *testing.T) {
	tempDir := t.TempDir()

	// Create only 3 Python files (below threshold of 5, no config file)
	createFile(t, tempDir, "script1.py", "import sys\n")
	createFile(t, tempDir, "script2.py", "def test(): pass\n")
	createFile(t, tempDir, "script3.py", "x = 1\n")

	_, err := DetectLanguages(tempDir)

	// Should fail - not enough files and no config file
	if err == nil {
		t.Error("Expected error for below threshold files without config, got nil")
	}
}

func TestDetectLanguages_PriorityOrder(t *testing.T) {
	tempDir := t.TempDir()

	// Create src and scripts directories
	os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "scripts"), 0755)

	// Create Go files (10 files)
	createFile(t, tempDir, "go.mod", "module test\n")
	for i := 1; i <= 10; i++ {
		filename := filepath.Join("src", "file"+fmt.Sprintf("%d", i)+".go")
		createFile(t, tempDir, filename, "package main\n")
	}

	// Create Python files (20 files - more than Go)
	createFile(t, tempDir, "requirements.txt", "requests==2.28.0\n")
	for i := 1; i <= 20; i++ {
		filename := filepath.Join("scripts", "script"+fmt.Sprintf("%d", i)+".py")
		createFile(t, tempDir, filename, "import sys\n")
	}

	languages, err := DetectLanguages(tempDir)
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Both should be detected
	if len(languages) != 2 {
		t.Errorf("Expected 2 languages, got %d: %v", len(languages), languages)
	}

	// Should be sorted by file count (Python has more files)
	if languages[0] != "python" {
		t.Errorf("Expected 'python' first (most files), got '%s'", languages[0])
	}
	if languages[1] != "go" {
		t.Errorf("Expected 'go' second, got '%s'", languages[1])
	}
}

func TestDetectLanguages_InvalidPath(t *testing.T) {
	_, err := DetectLanguages("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestDetectLanguages_EmptyPath(t *testing.T) {
	_, err := DetectLanguages("")
	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

func TestDetectLanguages_FileNotDirectory(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	createFile(t, tempDir, "test.txt", "content")

	_, err := DetectLanguages(filePath)
	if err == nil {
		t.Error("Expected error when path is a file not directory, got nil")
	}
}

// Helper function to create files in tests
func createFile(t *testing.T, baseDir, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(baseDir, relPath)
	dir := filepath.Dir(fullPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", fullPath, err)
	}
}
