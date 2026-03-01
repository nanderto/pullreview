package verify

import (
	"os"
	"path/filepath"
	"testing"
)

// writePackageJSON is a test helper that writes a package.json file.
func writePackageJSON(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
}

func TestNewJavaScriptVerifier_MissingPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	_, err = NewJavaScriptVerifier(tmpDir, false, cfg)
	if err == nil {
		t.Error("expected error when package.json is missing")
	}
}

func TestNewJavaScriptVerifier_InvalidPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	cfg := &VerificationConfig{RepoPath: tmpDir}
	_, err = NewJavaScriptVerifier(tmpDir, false, cfg)
	if err == nil {
		t.Error("expected error for invalid package.json")
	}
}

func TestDetectPackageManager_DefaultsToNpm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{"name": "test"}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.packageManager != "npm" {
		t.Errorf("expected packageManager=npm, got %s", v.packageManager)
	}
}

func TestDetectPackageManager_DetectsYarn(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{"name": "test"}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "yarn.lock"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create yarn.lock: %v", err)
	}

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.packageManager != "yarn" {
		t.Errorf("expected packageManager=yarn, got %s", v.packageManager)
	}
}

func TestDetectPackageManager_DetectsPnpm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{"name": "test"}`)
	// pnpm takes priority over yarn
	if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-lock.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create pnpm-lock.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "yarn.lock"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create yarn.lock: %v", err)
	}

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.packageManager != "pnpm" {
		t.Errorf("expected packageManager=pnpm (higher priority than yarn), got %s", v.packageManager)
	}
}

func TestHasScript_ReturnsTrueWhenScriptExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"build": "react-scripts build",
			"test": "react-scripts test",
			"lint": "eslint ."
		}
	}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, script := range []string{"build", "test", "lint"} {
		has, err := v.hasScript(script)
		if err != nil {
			t.Fatalf("hasScript(%q) returned error: %v", script, err)
		}
		if !has {
			t.Errorf("expected hasScript(%q)=true", script)
		}
	}
}

func TestHasScript_ReturnsFalseWhenScriptMissing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{"name": "test", "scripts": {"build": "vite build"}}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, script := range []string{"test", "lint"} {
		has, err := v.hasScript(script)
		if err != nil {
			t.Fatalf("hasScript(%q) returned error: %v", script, err)
		}
		if has {
			t.Errorf("expected hasScript(%q)=false", script)
		}
	}
}

func TestHasScript_NoScriptsSection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{"name": "test"}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	has, err := v.hasScript("build")
	if err != nil {
		t.Fatalf("hasScript returned error: %v", err)
	}
	if has {
		t.Error("expected hasScript=false when no scripts section")
	}
}

func TestIsPlaceholderTestScript_DetectsPlaceholder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"test": "echo \"Error: no test specified\" && exit 1"
		}
	}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !v.isPlaceholderTestScript() {
		t.Error("expected isPlaceholderTestScript=true for npm default placeholder")
	}
}

func TestIsPlaceholderTestScript_RealTestScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"test": "jest"
		}
	}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.isPlaceholderTestScript() {
		t.Error("expected isPlaceholderTestScript=false for real test script")
	}
}

func TestVerify_SkipsAllWhenConfigDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create node_modules to skip install
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"build": "exit 1",
			"test": "exit 1",
			"lint": "exit 1"
		}
	}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   false,
		RunBuild: false,
		RunTests: false,
	}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := v.Verify()
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if !result.AllPassed {
		t.Errorf("expected AllPassed=true when all checks disabled, errors: %s", result.CombinedErrors)
	}
}

func TestVerify_SkipsLintWhenNoLintScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create node_modules to skip install
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	// package.json with no lint script
	writePackageJSON(t, tmpDir, `{"name": "test", "scripts": {}}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   true, // lint enabled, but no lint script
		RunBuild: false,
		RunTests: false,
	}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := v.Verify()
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if !result.FmtPassed {
		t.Error("expected FmtPassed=true when no lint script (should skip silently)")
	}
}

func TestVerify_SkipsTestsWhenPlaceholder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"test": "echo \"Error: no test specified\" && exit 1"
		}
	}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   false,
		RunBuild: false,
		RunTests: true,
	}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := v.Verify()
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if !result.TestsPassed {
		t.Error("expected TestsPassed=true when test script is placeholder (should skip)")
	}
}

func TestVerify_BuildFailsWhenNoBuildScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	// No build script
	writePackageJSON(t, tmpDir, `{"name": "test", "scripts": {}}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   false,
		RunBuild: true,
		RunTests: false,
	}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := v.Verify()
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if result.BuildPassed {
		t.Error("expected BuildPassed=false when no build script")
	}
	if result.AllPassed {
		t.Error("expected AllPassed=false when build fails")
	}
}

func TestVerify_SkipsTestsOnBuildFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	// Build script missing (will fail), test script present
	writePackageJSON(t, tmpDir, `{
		"name": "test",
		"scripts": {
			"test": "jest"
		}
	}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   false,
		RunBuild: true,
		RunTests: true,
	}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := v.Verify()
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if result.BuildPassed {
		t.Error("expected BuildPassed=false")
	}
	if result.TestsPassed {
		t.Error("expected TestsPassed=false when build fails")
	}
	if result.TestsOutput != "skipped due to build failure" {
		t.Errorf("expected tests skipped message, got: %s", result.TestsOutput)
	}
}

func TestJavaScriptVerifier_InstallSkippedWhenNodeModulesExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// node_modules exists - install should be skipped
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	writePackageJSON(t, tmpDir, `{"name": "test"}`)

	cfg := &VerificationConfig{RepoPath: tmpDir}
	v, err := NewJavaScriptVerifier(tmpDir, false, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// installDependencies should return nil (skip) even without npm available
	if err := v.installDependencies(); err != nil {
		t.Errorf("expected no error when node_modules exists, got: %v", err)
	}
}

// TestVerifyDispatcher_JavaScriptRouting tests that the main Verifier
// routes JavaScript projects to the JavaScript verifier.
func TestVerifyDispatcher_JavaScriptRouting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "js-verify-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create node_modules so install is skipped
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	// Minimal JS project - no build/test scripts so nothing actually runs
	writePackageJSON(t, tmpDir, `{"name": "test", "scripts": {}}`)

	cfg := &VerificationConfig{
		RepoPath: tmpDir,
		RunFmt:   false,
		RunBuild: false,
		RunTests: false,
	}
	verifier := NewVerifier(cfg)

	// Verify primary language was detected as javascript
	if len(verifier.languages) == 0 {
		t.Fatal("expected at least one language detected")
	}
	if verifier.languages[0] != "javascript" {
		t.Errorf("expected primary language=javascript, got %s", verifier.languages[0])
	}

	result, err := verifier.RunAll()
	if err != nil {
		t.Fatalf("RunAll() returned error: %v", err)
	}
	if !result.AllPassed {
		t.Errorf("expected AllPassed=true when all checks disabled, errors: %s", result.CombinedErrors)
	}
}
