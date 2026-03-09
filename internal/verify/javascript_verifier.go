package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// JavaScriptVerifier handles verification for JavaScript/TypeScript projects.
// It is framework-agnostic and works with React, Angular, Svelte, Vue, Next.js, etc.
type JavaScriptVerifier struct {
	repoPath       string
	packageManager string // "npm", "yarn", or "pnpm"
	verbose        bool
	config         *VerificationConfig
	packageJSON    map[string]interface{} // Parsed package.json content
}

// NewJavaScriptVerifier creates a new JavaScript/TypeScript verifier.
// It detects the package manager from lock files.
func NewJavaScriptVerifier(repoPath string, verbose bool, cfg *VerificationConfig) (*JavaScriptVerifier, error) {
	v := &JavaScriptVerifier{
		repoPath: repoPath,
		verbose:  verbose,
		config:   cfg,
	}

	// Verify package.json exists
	pkgPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(pkgPath); err != nil {
		return nil, fmt.Errorf("JavaScript project detected but package.json not found at %s", pkgPath)
	}

	// Parse package.json
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}
	if err := json.Unmarshal(data, &v.packageJSON); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Detect package manager from lock files (pnpm > yarn > npm)
	v.packageManager = v.detectPackageManager()

	if verbose {
		fmt.Printf("JavaScript/TypeScript project detected, using %s\n", v.packageManager)
	}

	return v, nil
}

// detectPackageManager checks for lock files to determine the package manager.
func (v *JavaScriptVerifier) detectPackageManager() string {
	if _, err := os.Stat(filepath.Join(v.repoPath, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(v.repoPath, "yarn.lock")); err == nil {
		return "yarn"
	}
	return "npm"
}

// Verify runs all configured JavaScript/TypeScript verification checks.
func (v *JavaScriptVerifier) Verify() (*VerificationResult, error) {
	result := &VerificationResult{
		VetPassed:   true, // Not applicable for JS/TS
		FmtPassed:   true,
		BuildPassed: true,
		TestsPassed: true,
	}

	// Always install dependencies first
	if err := v.installDependencies(); err != nil {
		return result, fmt.Errorf("dependency installation failed: %w", err)
	}

	var errors []string

	// Run lint (uses RunFmt config flag, analogous to gofmt for JS)
	if v.config.RunFmt {
		passed, output, err := v.runLint()
		result.FmtPassed = passed
		result.FmtOutput = output
		if err != nil {
			return result, fmt.Errorf("lint execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("lint failed:\n%s", output))
			if v.verbose {
				fmt.Printf("❌ lint failed:\n%s\n", output)
			}
		} else if v.verbose {
			fmt.Println("✓ lint passed")
		}
	}

	// Run build
	if v.config.RunBuild {
		passed, output, err := v.runBuild()
		result.BuildPassed = passed
		result.BuildOutput = output
		if err != nil {
			return result, fmt.Errorf("build execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("build failed:\n%s", output))
			if v.verbose {
				fmt.Printf("❌ build failed:\n%s\n", output)
			}
		} else if v.verbose {
			fmt.Println("✓ build passed")
		}
	}

	// Run tests (only if build passed)
	if v.config.RunTests {
		if !result.BuildPassed {
			result.TestsPassed = false
			result.TestsOutput = "skipped due to build failure"
		} else {
			passed, output, err := v.runTests()
			result.TestsPassed = passed
			result.TestsOutput = output
			if err != nil {
				return result, fmt.Errorf("test execution error: %w", err)
			}
			if !passed {
				errors = append(errors, fmt.Sprintf("tests failed:\n%s", output))
				if v.verbose {
					fmt.Printf("❌ tests failed:\n%s\n", output)
				}
			} else if v.verbose {
				fmt.Println("✓ tests passed")
			}
		}
	}

	if len(errors) > 0 {
		result.CombinedErrors = strings.Join(errors, "\n\n")
	}

	result.AllPassed = result.FmtPassed && result.BuildPassed && result.TestsPassed

	return result, nil
}

// installDependencies runs the package manager install command.
// Skips if node_modules already exists to avoid slow unnecessary installs.
func (v *JavaScriptVerifier) installDependencies() error {
	nodeModules := filepath.Join(v.repoPath, "node_modules")
	if _, err := os.Stat(nodeModules); err == nil {
		// node_modules exists, skip install
		return nil
	}

	if v.verbose {
		fmt.Printf("Running %s install...\n", v.packageManager)
	}

	var cmd *exec.Cmd
	switch v.packageManager {
	case "pnpm":
		cmd = exec.Command("pnpm", "install", "--frozen-lockfile")
	case "yarn":
		cmd = exec.Command("yarn", "install", "--frozen-lockfile")
	default:
		cmd = exec.Command("npm", "ci")
	}
	cmd.Dir = v.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("npm install failed - check network connection and package.json dependencies:\n%s",
				combineOutput(stdout.String(), stderr.String()))
		}
		return fmt.Errorf("failed to run %s install: %w", v.packageManager, err)
	}

	return nil
}

// runLint runs the lint script if it exists in package.json.
// Returns (true, "", nil) silently if no lint script is defined.
func (v *JavaScriptVerifier) runLint() (bool, string, error) {
	hasLint, err := v.hasScript("lint")
	if err != nil {
		return false, "", err
	}
	if !hasLint {
		if v.verbose {
			fmt.Println("No lint script in package.json, skipping lint")
		}
		return true, "", nil
	}

	cmd := v.makeRunCommand("lint")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, runErr
	}

	return true, output, nil
}

// runBuild runs the build script. Returns an error if no build script exists.
func (v *JavaScriptVerifier) runBuild() (bool, string, error) {
	hasBuild, err := v.hasScript("build")
	if err != nil {
		return false, "", err
	}
	if !hasBuild {
		return false, "Build script not found in package.json - add a 'build' script or disable verify_build", nil
	}

	cmd := v.makeRunCommand("build")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, runErr
	}

	return true, output, nil
}

// runTests runs the test script. Returns (true, "", nil) silently if no test
// script exists or if it's the default placeholder.
func (v *JavaScriptVerifier) runTests() (bool, string, error) {
	hasTest, err := v.hasScript("test")
	if err != nil {
		return false, "", err
	}
	if !hasTest {
		if v.verbose {
			fmt.Println("No test script in package.json, skipping tests")
		}
		return true, "", nil
	}

	// Check if test script is a placeholder (npm default)
	if v.isPlaceholderTestScript() {
		if v.verbose {
			fmt.Println("Test script is a placeholder, skipping tests")
		}
		return true, "", nil
	}

	cmd := v.makeTestCommand()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, runErr
	}

	return true, output, nil
}

// hasScript checks whether a script name exists in package.json scripts.
func (v *JavaScriptVerifier) hasScript(scriptName string) (bool, error) {
	scripts, ok := v.packageJSON["scripts"]
	if !ok {
		return false, nil
	}
	scriptsMap, ok := scripts.(map[string]interface{})
	if !ok {
		return false, nil
	}
	_, exists := scriptsMap[scriptName]
	return exists, nil
}

// isPlaceholderTestScript returns true if the test script is npm's default placeholder.
func (v *JavaScriptVerifier) isPlaceholderTestScript() bool {
	scripts, ok := v.packageJSON["scripts"]
	if !ok {
		return false
	}
	scriptsMap, ok := scripts.(map[string]interface{})
	if !ok {
		return false
	}
	testScript, ok := scriptsMap["test"].(string)
	if !ok {
		return false
	}
	return strings.Contains(testScript, "no test specified")
}

// makeRunCommand creates an exec.Command to run a named npm script.
func (v *JavaScriptVerifier) makeRunCommand(script string) *exec.Cmd {
	var cmd *exec.Cmd
	switch v.packageManager {
	case "pnpm":
		cmd = exec.Command("pnpm", "run", script)
	case "yarn":
		cmd = exec.Command("yarn", script)
	default:
		cmd = exec.Command("npm", "run", script)
	}
	cmd.Dir = v.repoPath
	return cmd
}

// makeTestCommand creates an exec.Command for running tests.
// Uses the idiomatic `npm test` / `yarn test` / `pnpm test` form.
func (v *JavaScriptVerifier) makeTestCommand() *exec.Cmd {
	var cmd *exec.Cmd
	switch v.packageManager {
	case "pnpm":
		cmd = exec.Command("pnpm", "test")
	case "yarn":
		cmd = exec.Command("yarn", "test")
	default:
		cmd = exec.Command("npm", "test")
	}
	cmd.Dir = v.repoPath
	return cmd
}
