package verify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CSharpVerifier handles verification for C# projects using dotnet CLI.
type CSharpVerifier struct {
	repoPath     string
	verbose      bool
	config       *VerificationConfig
	solutionPath string // Path to .sln file (if found)
}

// NewCSharpVerifier creates a new C# verifier.
func NewCSharpVerifier(repoPath string, verbose bool, cfg *VerificationConfig) *CSharpVerifier {
	v := &CSharpVerifier{
		repoPath: repoPath,
		verbose:  verbose,
		config:   cfg,
	}
	
	// Find solution file
	v.solutionPath = v.findSolutionFile()
	
	return v
}

// findSolutionFile searches for a .sln file in the repository.
func (v *CSharpVerifier) findSolutionFile() string {
	var solutionPath string
	
	filepath.Walk(v.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip common ignore directories
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "bin" || name == "obj" {
				return filepath.SkipDir
			}
		}
		
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".sln") {
			solutionPath = path
			return filepath.SkipAll // Stop walking once we find one
		}
		
		return nil
	})
	
	return solutionPath
}

// Verify runs all C# verification checks.
func (v *CSharpVerifier) Verify() (*VerificationResult, error) {
	result := &VerificationResult{
		VetPassed:   true, // Not applicable for C#
		FmtPassed:   true, // Could add dotnet format later
		BuildPassed: true,
		TestsPassed: true,
	}
	
	// Check if we found a solution file
	if v.solutionPath == "" {
		if v.verbose {
			fmt.Println("Warning: No .sln file found, skipping C# verification")
		}
		return result, nil
	}
	
	if v.verbose {
		fmt.Printf("Using solution file: %s\n", v.solutionPath)
	}

	var errors []string

	// Run dotnet build (if enabled)
	if v.config.RunBuild {
		passed, output, err := v.runBuild()
		result.BuildPassed = passed
		result.BuildOutput = output
		if err != nil {
			return result, fmt.Errorf("dotnet build execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("dotnet build failed:\n%s", output))
			if v.verbose {
				fmt.Printf("❌ dotnet build failed:\n%s\n", output)
			}
		} else if v.verbose {
			fmt.Println("✓ dotnet build passed")
		}
	}

	// Run dotnet test (only if enabled and build passed)
	if v.config.RunTests {
		if result.BuildPassed {
			passed, output, err := v.runTest()
			result.TestsPassed = passed
			result.TestsOutput = output
			if err != nil {
				return result, fmt.Errorf("dotnet test execution error: %w", err)
			}
			if !passed {
				errors = append(errors, fmt.Sprintf("dotnet test failed:\n%s", output))
				if v.verbose {
					fmt.Printf("❌ dotnet test failed:\n%s\n", output)
				}
			} else if v.verbose {
				fmt.Println("✓ dotnet test passed")
			}
		} else {
			result.TestsPassed = false
			result.TestsOutput = "skipped due to build failure"
		}
	}

	// Combine errors
	if len(errors) > 0 {
		result.CombinedErrors = strings.Join(errors, "\n\n")
	}

	// Set overall result
	result.AllPassed = result.BuildPassed && result.TestsPassed

	return result, nil
}

// runBuild runs `dotnet build` on the solution file.
func (v *CSharpVerifier) runBuild() (bool, string, error) {
	cmd := exec.Command("dotnet", "build", v.solutionPath, "--no-incremental")
	cmd.Dir = v.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, err
	}

	return true, output, nil
}

// runTest runs `dotnet test` on the solution file.
func (v *CSharpVerifier) runTest() (bool, string, error) {
	cmd := exec.Command("dotnet", "test", v.solutionPath, "--no-build", "--verbosity", "normal")
	cmd.Dir = v.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, err
	}

	return true, output, nil
}
