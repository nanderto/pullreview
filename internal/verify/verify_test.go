package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifier_RunAll_AllPass(t *testing.T) {
	// Create temp directory with valid Go code
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create valid Go file
	goFile := `package testpkg

func Hello() string {
	return "hello"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create verifier
	cfg := &VerificationConfig{
		RunVet:   true,
		RunFmt:   true,
		RunBuild: true,
		RunTests: false, // No tests to run
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if !result.VetPassed {
		t.Errorf("expected VetPassed=true, got false. Output: %s", result.VetOutput)
	}
	if !result.FmtPassed {
		t.Errorf("expected FmtPassed=true, got false. Output: %s", result.FmtOutput)
	}
	if !result.BuildPassed {
		t.Errorf("expected BuildPassed=true, got false. Output: %s", result.BuildOutput)
	}
	if !result.AllPassed {
		t.Errorf("expected AllPassed=true, got false. Errors: %s", result.CombinedErrors)
	}
}

func TestVerifier_RunAll_VetFails(t *testing.T) {
	// Create temp directory with code that fails vet
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file with vet issue (Printf format issue)
	goFile := `package testpkg

import "fmt"

func Hello() {
	fmt.Printf("%s", 123)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create verifier
	cfg := &VerificationConfig{
		RunVet:   true,
		RunFmt:   false,
		RunBuild: false,
		RunTests: false,
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed with execution error: %v", err)
	}

	if result.VetPassed {
		t.Errorf("expected VetPassed=false for vet issues")
	}
	if result.AllPassed {
		t.Errorf("expected AllPassed=false when vet fails")
	}
}

func TestVerifier_RunAll_FmtFails(t *testing.T) {
	// Create temp directory with unformatted code
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create unformatted Go file (bad indentation)
	goFile := `package testpkg

func Hello() string {
return "hello"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create verifier
	cfg := &VerificationConfig{
		RunVet:   false,
		RunFmt:   true,
		RunBuild: false,
		RunTests: false,
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if result.FmtPassed {
		t.Errorf("expected FmtPassed=false for unformatted code")
	}
	if result.AllPassed {
		t.Errorf("expected AllPassed=false when fmt fails")
	}
}

func TestVerifier_RunAll_BuildFails(t *testing.T) {
	// Create temp directory with code that fails to build
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file with syntax error
	goFile := `package testpkg

func Hello() string {
	return undefined
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create verifier
	cfg := &VerificationConfig{
		RunVet:   false,
		RunFmt:   false,
		RunBuild: true,
		RunTests: false,
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if result.BuildPassed {
		t.Errorf("expected BuildPassed=false for build error")
	}
	if result.AllPassed {
		t.Errorf("expected AllPassed=false when build fails")
	}
}

func TestVerifier_RunAll_TestsFail(t *testing.T) {
	// Create temp directory with failing tests
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create source file
	goFile := `package testpkg

func Hello() string {
	return "hello"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create failing test
	testFile := `package testpkg

import "testing"

func TestHello(t *testing.T) {
	if Hello() != "goodbye" {
		t.Error("expected goodbye")
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create verifier
	cfg := &VerificationConfig{
		RunVet:   false,
		RunFmt:   false,
		RunBuild: true,
		RunTests: true,
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if !result.BuildPassed {
		t.Errorf("expected BuildPassed=true")
	}
	if result.TestsPassed {
		t.Errorf("expected TestsPassed=false for failing test")
	}
	if result.AllPassed {
		t.Errorf("expected AllPassed=false when tests fail")
	}
}

func TestVerifier_RunAll_SkipTestsOnBuildFailure(t *testing.T) {
	// Create temp directory with code that fails to build
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file with syntax error
	goFile := `package testpkg

func Hello() string {
	return undefined
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create verifier with both build and tests enabled
	cfg := &VerificationConfig{
		RunVet:   false,
		RunFmt:   false,
		RunBuild: true,
		RunTests: true,
		RepoPath: tmpDir,
		Verbose:  false,
	}
	verifier := NewVerifier(cfg)

	// Run verification
	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	if result.BuildPassed {
		t.Errorf("expected BuildPassed=false")
	}
	if result.TestsPassed {
		t.Errorf("expected TestsPassed=false when build fails")
	}
	if result.TestsOutput != "skipped due to build failure" {
		t.Errorf("expected tests to be skipped, got: %s", result.TestsOutput)
	}
}

func TestVerifier_SetVerbose(t *testing.T) {
	cfg := &VerificationConfig{
		Verbose: false,
	}
	verifier := NewVerifier(cfg)

	verifier.SetVerbose(true)

	if !verifier.config.Verbose {
		t.Errorf("expected Verbose=true after SetVerbose(true)")
	}
}

func TestVerifier_Verify_AliasRunAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create minimal valid go project
	goMod := `module testpkg

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goFile := `package testpkg
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	cfg := &VerificationConfig{
		RunVet:   false,
		RunFmt:   false,
		RunBuild: false,
		RunTests: false,
		RepoPath: tmpDir,
	}
	verifier := NewVerifier(cfg)

	// Both Verify and RunAll should work
	result1, err1 := verifier.Verify()
	result2, err2 := verifier.RunAll()

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	if result1.AllPassed != result2.AllPassed {
		t.Error("Verify() and RunAll() should produce same results")
	}
}

func TestCombineOutput(t *testing.T) {
	tests := []struct {
		stdout   string
		stderr   string
		expected string
	}{
		{"", "", ""},
		{"out", "", "out"},
		{"", "err", "err"},
		{"out", "err", "out\nerr"},
		{"  out  ", "  err  ", "out\nerr"},
	}

	for _, tt := range tests {
		result := combineOutput(tt.stdout, tt.stderr)
		if result != tt.expected {
			t.Errorf("combineOutput(%q, %q) = %q, want %q",
				tt.stdout, tt.stderr, result, tt.expected)
		}
	}
}
