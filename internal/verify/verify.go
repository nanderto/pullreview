package verify

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// VerificationConfig holds configuration for build/test/lint verification.
type VerificationConfig struct {
	RunVet   bool
	RunFmt   bool
	RunBuild bool
	RunTests bool
	RepoPath string
	Verbose  bool
}

// VerificationResult holds the results of verification checks.
type VerificationResult struct {
	AllPassed      bool
	BuildPassed    bool
	TestsPassed    bool
	VetPassed      bool
	FmtPassed      bool
	BuildOutput    string
	TestsOutput    string
	VetOutput      string
	FmtOutput      string
	CombinedErrors string
}

// Verifier runs build/test/lint verification.
type Verifier struct {
	config    *VerificationConfig
	languages []string // Detected languages
}

// NewVerifier creates a new Verifier instance.
func NewVerifier(cfg *VerificationConfig) *Verifier {
	// Detect languages in the repository
	languages, err := DetectLanguages(cfg.RepoPath)
	if err != nil {
		// Fallback to Go if detection fails (for backwards compatibility)
		if cfg.Verbose {
			fmt.Printf("Warning: language detection failed: %v, assuming Go\n", err)
		}
		languages = []string{"go"}
	}
	
	if cfg.Verbose {
		fmt.Printf("Detected languages: %v\n", languages)
	}
	
	return &Verifier{
		config:    cfg,
		languages: languages,
	}
}

// SetVerbose enables debug output.
func (v *Verifier) SetVerbose(verbose bool) {
	v.config.Verbose = verbose
}

// Verify runs all configured verification checks.
func (v *Verifier) Verify() (*VerificationResult, error) {
	return v.RunAll()
}

// RunAll runs all configured verification checks.
// Checks are run in order: vet, fmt, build, test.
// Fails fast on first error to speed up iteration.
func (v *Verifier) RunAll() (*VerificationResult, error) {
	// Determine primary language
	primaryLang := "go" // default
	if len(v.languages) > 0 {
		primaryLang = v.languages[0]
	}
	
	if v.config.Verbose {
		fmt.Printf("Running verification for %s project\n", primaryLang)
	}
	
	// Route to appropriate verifier based on language
	switch primaryLang {
	case "csharp":
		return v.runCSharpVerification()
	case "go":
		return v.runGoVerification()
	default:
		// For unsupported languages, skip verification
		if v.config.Verbose {
			fmt.Printf("Warning: verification not yet supported for %s, skipping\n", primaryLang)
		}
		return &VerificationResult{
			AllPassed:   true,
			VetPassed:   true,
			FmtPassed:   true,
			BuildPassed: true,
			TestsPassed: true,
		}, nil
	}
}

// runCSharpVerification runs C# specific verification.
func (v *Verifier) runCSharpVerification() (*VerificationResult, error) {
	verifier := NewCSharpVerifier(v.config.RepoPath, v.config.Verbose, v.config)
	return verifier.Verify()
}

// runGoVerification runs Go specific verification (original implementation).
func (v *Verifier) runGoVerification() (*VerificationResult, error) {
	result := &VerificationResult{
		VetPassed:   true,
		FmtPassed:   true,
		BuildPassed: true,
		TestsPassed: true,
	}

	var errors []string

	// Run go vet (quickest static analysis)
	if v.config.RunVet {
		passed, output, err := v.runVet()
		result.VetPassed = passed
		result.VetOutput = output
		if err != nil {
			return result, fmt.Errorf("go vet execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("go vet failed:\n%s", output))
			if v.config.Verbose {
				fmt.Printf("❌ go vet failed:\n%s\n", output)
			}
		} else if v.config.Verbose {
			fmt.Println("✓ go vet passed")
		}
	}

	// Run gofmt check
	if v.config.RunFmt {
		passed, output, err := v.runFmt()
		result.FmtPassed = passed
		result.FmtOutput = output
		if err != nil {
			return result, fmt.Errorf("gofmt execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("gofmt check failed (unformatted files):\n%s", output))
			if v.config.Verbose {
				fmt.Printf("❌ gofmt check failed:\n%s\n", output)
			}
		} else if v.config.Verbose {
			fmt.Println("✓ gofmt passed")
		}
	}

	// Run go build
	if v.config.RunBuild {
		passed, output, err := v.runBuild()
		result.BuildPassed = passed
		result.BuildOutput = output
		if err != nil {
			return result, fmt.Errorf("go build execution error: %w", err)
		}
		if !passed {
			errors = append(errors, fmt.Sprintf("go build failed:\n%s", output))
			if v.config.Verbose {
				fmt.Printf("❌ go build failed:\n%s\n", output)
			}
		} else if v.config.Verbose {
			fmt.Println("✓ go build passed")
		}
	}

	// Run go test (slowest, only if others pass)
	if v.config.RunTests {
		// Skip tests if build failed
		if !result.BuildPassed {
			result.TestsPassed = false
			result.TestsOutput = "skipped due to build failure"
		} else {
			passed, output, err := v.runTests()
			result.TestsPassed = passed
			result.TestsOutput = output
			if err != nil {
				return result, fmt.Errorf("go test execution error: %w", err)
			}
			if !passed {
				errors = append(errors, fmt.Sprintf("go test failed:\n%s", output))
				if v.config.Verbose {
					fmt.Printf("❌ go test failed:\n%s\n", output)
				}
			} else if v.config.Verbose {
				fmt.Println("✓ go test passed")
			}
		}
	}

	// Combine errors
	if len(errors) > 0 {
		result.CombinedErrors = strings.Join(errors, "\n\n")
	}

	// Set overall result
	result.AllPassed = result.VetPassed && result.FmtPassed && result.BuildPassed && result.TestsPassed

	return result, nil
}

// runVet runs `go vet ./...`.
func (v *Verifier) runVet() (bool, string, error) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = v.config.RepoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := combineOutput(stdout.String(), stderr.String())

	if err != nil {
		// Exit error means vet found issues - not an execution error
		if _, ok := err.(*exec.ExitError); ok {
			return false, output, nil
		}
		return false, output, err
	}

	return true, output, nil
}

// runFmt runs `gofmt -s -l .` to check for unformatted files.
func (v *Verifier) runFmt() (bool, string, error) {
	cmd := exec.Command("gofmt", "-s", "-l", ".")
	cmd.Dir = v.config.RepoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// gofmt should not fail unless something is seriously wrong
		if _, ok := err.(*exec.ExitError); ok {
			return false, combineOutput(stdout.String(), stderr.String()), nil
		}
		return false, combineOutput(stdout.String(), stderr.String()), err
	}

	// If stdout has content, there are unformatted files
	output := strings.TrimSpace(stdout.String())
	if output != "" {
		return false, output, nil
	}

	return true, "", nil
}

// runBuild runs `go build ./...`.
func (v *Verifier) runBuild() (bool, string, error) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = v.config.RepoPath

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

// runTests runs `go test ./...`.
func (v *Verifier) runTests() (bool, string, error) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = v.config.RepoPath

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

// combineOutput combines stdout and stderr into a single string.
func combineOutput(stdout, stderr string) string {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if stdout == "" && stderr == "" {
		return ""
	}
	if stdout == "" {
		return stderr
	}
	if stderr == "" {
		return stdout
	}
	return stdout + "\n" + stderr
}
