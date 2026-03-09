package autofix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"pullreview/internal/bitbucket"
	"pullreview/internal/llm"
	"pullreview/internal/review"
	"pullreview/internal/verify"
)

// AutofixResponse holds both issues and fixes from the combined autofix prompt.
type AutofixResponse struct {
	Issues  []AutofixIssue `json:"issues"`
	Fixes   []Fix          `json:"fixes"`
	Summary string         `json:"summary"`
}

// AutofixIssue represents a single issue found during autofix.
type AutofixIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Comment  string `json:"comment"`
	Severity string `json:"severity"`
}

// AutoFixer orchestrates the auto-fix workflow.
type AutoFixer struct {
	config    *AutoFixConfig
	llmClient *llm.Client
	applier   *Applier
	verifier  *verify.Verifier
	bbClient  *bitbucket.Client
	repoPath  string
	verbose   bool
}

// NewAutoFixer creates a new AutoFixer instance.
func NewAutoFixer(cfg *AutoFixConfig, llmClient *llm.Client, repoPath string) *AutoFixer {
	cfg.SetDefaults()

	applier := NewApplier(repoPath)
	verifierCfg := &verify.VerificationConfig{
		RunVet:   cfg.VerifyLint,
		RunFmt:   cfg.VerifyLint,
		RunBuild: cfg.VerifyBuild,
		RunTests: cfg.VerifyTests,
		RepoPath: repoPath,
		Verbose:  false,
	}
	verifier := verify.NewVerifier(verifierCfg)

	return &AutoFixer{
		config:    cfg,
		llmClient: llmClient,
		applier:   applier,
		verifier:  verifier,
		repoPath:  repoPath,
		verbose:   false,
	}
}

// SetVerbose enables debug output.
func (af *AutoFixer) SetVerbose(v bool) {
	af.verbose = v
	af.applier.SetVerbose(v)
	af.verifier.SetVerbose(v)
}

// SetBitbucketClient sets the Bitbucket client for PR operations.
func (af *AutoFixer) SetBitbucketClient(client *bitbucket.Client) {
	af.bbClient = client
}

// ApplyFixesDirectly applies a list of fixes without iterative generation.
// Used when fixes are already provided (e.g., from combined autofix prompt).
func (af *AutoFixer) ApplyFixesDirectly(
	ctx context.Context,
	fixes []Fix,
) (*FixResult, error) {
	result := &FixResult{
		Success:       false,
		FilesChanged:  []string{},
		FixesApplied:  0,
		FixesFailed:   0,
		Iterations:    1,
		ErrorMessages: []string{},
	}

	if len(fixes) == 0 {
		result.Success = true
		return result, nil
	}

	// Apply fixes and verify
	verificationResult, err := af.applyAndVerify(fixes)
	if err != nil {
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Apply/verify error: %v", err))
		// Restore backups on error
		af.applier.RestoreBackups()
		return result, fmt.Errorf("apply/verify failed: %w", err)
	}

	// Update result with verification status
	if verificationResult.BuildPassed {
		result.BuildStatus = "passed"
	} else {
		result.BuildStatus = "failed"
	}

	if verificationResult.TestsPassed {
		result.TestStatus = "passed"
	} else {
		result.TestStatus = "failed"
	}

	if verificationResult.VetPassed && verificationResult.FmtPassed {
		result.LintStatus = "passed"
	} else {
		result.LintStatus = "failed"
	}

	// Check if all verifications passed
	if verificationResult.AllPassed {
		if af.verbose {
			fmt.Println("✓ All verifications passed!")
		}
		result.Success = true
		result.FixesApplied = len(fixes)

		// Track modified files
		for _, fix := range fixes {
			if !containsString(result.FilesChanged, fix.File) {
				result.FilesChanged = append(result.FilesChanged, fix.File)
			}
		}

		// Clear backups since we succeeded
		af.applier.ClearBackups()

		return result, nil
	}

	// Verification failed
	result.Success = false
	result.FixesFailed = len(fixes)
	result.ErrorMessages = append(result.ErrorMessages, verificationResult.CombinedErrors)
	af.applier.RestoreBackups()

	return result, fmt.Errorf("verification failed: %s", verificationResult.CombinedErrors)
}

// GenerateAndApplyFixes runs the full fix generation and application workflow.
// This includes the iterative verification loop.
func (af *AutoFixer) GenerateAndApplyFixes(
	ctx context.Context,
	reviewComments []review.Comment,
	diff string,
	fileContents map[string]string,
) (*FixResult, error) {
	result := &FixResult{
		Success:       false,
		FilesChanged:  []string{},
		FixesApplied:  0,
		FixesFailed:   0,
		Iterations:    0,
		ErrorMessages: []string{},
	}

	if len(reviewComments) == 0 {
		if af.verbose {
			fmt.Println("No review comments to fix")
		}
		result.Success = true
		return result, nil
	}

	// Iterative fix loop
	var currentFix *FixResponse
	var verificationResult *verify.VerificationResult

	for iteration := 1; iteration <= af.config.MaxIterations; iteration++ {
		result.Iterations = iteration

		if af.verbose {
			fmt.Printf("\n=== Iteration %d/%d ===\n", iteration, af.config.MaxIterations)
		}

		// Generate fixes (either initial or correction)
		var err error
		if iteration == 1 {
			// Initial fix generation
			currentFix, err = af.generateFixes(ctx, reviewComments, diff, fileContents)
		} else {
			// Fix correction based on verification errors
			if verificationResult == nil || verificationResult.AllPassed {
				break
			}

			// Read current file contents after previous fix attempt
			updatedContents, readErr := af.readModifiedFiles(currentFix.Fixes)
			if readErr != nil {
				result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Failed to read modified files: %v", readErr))
				return result, fmt.Errorf("failed to read modified files: %w", readErr)
			}

			currentFix, err = af.requestFixCorrection(ctx, currentFix, verificationResult.CombinedErrors, updatedContents)
		}

		if err != nil {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Fix generation error: %v", err))
			return result, fmt.Errorf("fix generation failed at iteration %d: %w", iteration, err)
		}

		if len(currentFix.Fixes) == 0 {
			if af.verbose {
				fmt.Println("No fixes returned by LLM")
			}
			result.Success = true
			return result, nil
		}

		// Apply fixes and verify
		verificationResult, err = af.applyAndVerify(currentFix.Fixes)
		if err != nil {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Apply/verify error: %v", err))
			// Restore backups on error
			af.applier.RestoreBackups()
			return result, fmt.Errorf("apply/verify failed at iteration %d: %w", iteration, err)
		}

		// Update result with verification status
		if verificationResult.BuildPassed {
			result.BuildStatus = "passed"
		} else {
			result.BuildStatus = "failed"
		}

		if verificationResult.TestsPassed {
			result.TestStatus = "passed"
		} else {
			result.TestStatus = "failed"
		}

		if verificationResult.VetPassed && verificationResult.FmtPassed {
			result.LintStatus = "passed"
		} else {
			result.LintStatus = "failed"
		}

		// Check if all verifications passed
		if verificationResult.AllPassed {
			if af.verbose {
				fmt.Println("✓ All verifications passed!")
			}
			result.Success = true
			result.FixesApplied = len(currentFix.Fixes)

			// Track modified files
			for _, fix := range currentFix.Fixes {
				if !containsString(result.FilesChanged, fix.File) {
					result.FilesChanged = append(result.FilesChanged, fix.File)
				}
			}

			// Clear backups since we succeeded
			af.applier.ClearBackups()

			return result, nil
		}

		if af.verbose {
			fmt.Printf("✗ Verification failed: %s\n", verificationResult.CombinedErrors)
		}
	}

	// Max iterations exceeded - restore original files
	if af.verbose {
		fmt.Printf("Max iterations (%d) exceeded. Restoring original files.\n", af.config.MaxIterations)
	}

	err := af.applier.RestoreBackups()
	if err != nil {
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Failed to restore backups: %v", err))
		return result, fmt.Errorf("failed to restore backups: %w", err)
	}

	result.Success = false
	result.FixesFailed = len(currentFix.Fixes)
	result.ErrorMessages = append(result.ErrorMessages, "Max iterations exceeded")
	if verificationResult != nil {
		result.ErrorMessages = append(result.ErrorMessages, verificationResult.CombinedErrors)
	}

	return result, fmt.Errorf("max iterations exceeded, final errors: %s", verificationResult.CombinedErrors)
}

// GenerateFindAndFix uses the combined autofix prompt to find issues and generate fixes in one LLM call.
// Returns both the issues (for posting as comments) and fixes (for applying to code).
func (af *AutoFixer) GenerateFindAndFix(
	ctx context.Context,
	diff string,
	fileContents map[string]string,
) (*AutofixResponse, error) {
	if af.verbose {
		fmt.Println("Generating issues and fixes from LLM (combined autofix prompt)...")
	}

	prompt := af.buildAutofixPrompt(diff, fileContents)

	response, err := af.llmClient.SendFixPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Strip markdown code fences if present
	jsonStr := extractJSON(response)

	// Parse the JSON response
	var autofixResponse AutofixResponse
	err = json.Unmarshal([]byte(jsonStr), &autofixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse: %s", err, response)
	}

	// Validate and filter fixes
	validFixes := af.validateFixes(autofixResponse.Fixes)
	autofixResponse.Fixes = validFixes

	if af.verbose {
		fmt.Printf("LLM returned %d issue(s) and %d fix(es)\n", len(autofixResponse.Issues), len(autofixResponse.Fixes))
	}

	return &autofixResponse, nil
}

// buildAutofixPrompt constructs the combined find+fix prompt.
func (af *AutoFixer) buildAutofixPrompt(
	diff string,
	fileContents map[string]string,
) string {
	// Read the autofix prompt template
	promptTemplate, err := af.readPromptTemplate(af.config.AutofixPromptFile)
	if err != nil {
		if af.verbose {
			fmt.Printf("Failed to read autofix prompt template: %v, using default\n", err)
		}
		promptTemplate = getDefaultAutofixPrompt()
	}

	// Format file contents
	var fileContentsStr strings.Builder
	for file, content := range fileContents {
		fileContentsStr.WriteString(fmt.Sprintf("\n### File: %s\n\n```\n%s\n```\n", file, content))
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(promptTemplate, "{DIFF_CONTENT}", diff)
	prompt = strings.ReplaceAll(prompt, "{FILE_CONTENTS}", fileContentsStr.String())

	return prompt
}

// getDefaultAutofixPrompt returns a default combined find+fix prompt.
func getDefaultAutofixPrompt() string {
	return `# AUTO-FIX: FIND ISSUES AND GENERATE FIXES

You are a code reviewer and fix generator. Find defects and generate fixes in one response.

## OUTPUT FORMAT
Respond with ONLY a JSON object (no markdown, no explanation):

{
  "issues": [
    {
      "file": "path/to/file",
      "line": 45,
      "comment": "Description of the issue",
      "severity": "high|medium|low"
    }
  ],
  "fixes": [
    {
      "file": "path/to/file",
      "line_start": 45,
      "line_end": 47,
      "original_code": "exact code to replace",
      "fixed_code": "corrected code",
      "issue_addressed": "Brief description",
      "severity": "high|medium|low"
    }
  ],
  "summary": "Brief summary"
}

## DIFF
{DIFF_CONTENT}

## FILE CONTENTS
{FILE_CONTENTS}
`
}

// generateFixes sends review issues to LLM and parses the response.
func (af *AutoFixer) generateFixes(
	ctx context.Context,
	reviewComments []review.Comment,
	diff string,
	fileContents map[string]string,
) (*FixResponse, error) {
	if af.verbose {
		fmt.Println("Generating fixes from LLM...")
	}

	prompt := af.buildFixPrompt(reviewComments, diff, fileContents)

	response, err := af.llmClient.SendFixPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Strip markdown code fences if present
	jsonStr := extractJSON(response)

	// Parse the JSON response
	var fixResponse FixResponse
	err = json.Unmarshal([]byte(jsonStr), &fixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse: %s", err, response)
	}

	// Validate and filter fixes
	validFixes := af.validateFixes(fixResponse.Fixes)
	fixResponse.Fixes = validFixes

	if af.verbose {
		fmt.Printf("LLM returned %d fix(es)\n", len(fixResponse.Fixes))
	}

	return &fixResponse, nil
}

// applyAndVerify applies fixes and runs verification.
// Returns verification result and any errors.
func (af *AutoFixer) applyAndVerify(fixes []Fix) (*verify.VerificationResult, error) {
	if af.verbose {
		fmt.Printf("Applying %d fix(es)...\n", len(fixes))
	}

	// Apply all fixes
	modifiedFiles, err := af.applier.ApplyFixes(fixes)
	if err != nil {
		return nil, fmt.Errorf("failed to apply fixes: %w", err)
	}

	if af.verbose {
		fmt.Printf("Applied fixes to %d file(s): %v\n", len(modifiedFiles), modifiedFiles)
	}

	// Auto-format modified files before verification (if lint checking is enabled)
	if af.config.VerifyLint {
		if af.verbose {
			fmt.Println("Auto-formatting modified files...")
		}
		if err := af.autoFormatFiles(modifiedFiles); err != nil {
			if af.verbose {
				fmt.Printf("Warning: auto-format failed: %v\n", err)
			}
			// Continue anyway - verification will catch format issues
		}
	}

	if af.verbose {
		fmt.Println("Running verification...")
	}

	// Run verification
	verificationResult, err := af.verifier.RunAll()
	if err != nil {
		return verificationResult, fmt.Errorf("verification error: %w", err)
	}

	return verificationResult, nil
}

// requestFixCorrection sends verification errors to LLM for correction.
func (af *AutoFixer) requestFixCorrection(
	ctx context.Context,
	previousFix *FixResponse,
	verificationError string,
	fileContents map[string]string,
) (*FixResponse, error) {
	if af.verbose {
		fmt.Println("Requesting fix correction from LLM...")
	}

	// Parse error messages to find files that have errors
	errorFiles := af.parseErrorFiles(verificationError)

	// Combine previously modified files with error files
	allRelevantFiles := make(map[string]bool)

	// Add all files from previous fix
	for file := range fileContents {
		allRelevantFiles[file] = true
	}

	// Add all files mentioned in errors
	for _, file := range errorFiles {
		allRelevantFiles[file] = true
	}

	if af.verbose && len(errorFiles) > 0 {
		fmt.Printf("Found %d file(s) mentioned in errors: %v\n", len(errorFiles), errorFiles)
	}

	// Read contents of all relevant files
	enhancedContents := make(map[string]string)
	for file := range allRelevantFiles {
		// Use existing content if we have it
		if content, exists := fileContents[file]; exists {
			enhancedContents[file] = content
			continue
		}

		// Otherwise read from disk
		fullPath := filepath.Join(af.repoPath, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if af.verbose {
				fmt.Printf("Warning: could not read %s: %v\n", file, err)
			}
			continue
		}
		enhancedContents[file] = string(content)
	}

	if af.verbose {
		fmt.Printf("Providing %d file(s) to LLM for context\n", len(enhancedContents))
	}

	prompt := af.buildCorrectionPrompt(previousFix, verificationError, enhancedContents)

	response, err := af.llmClient.SendFixPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM correction request failed: %w", err)
	}

	// Strip markdown code fences if present
	jsonStr := extractJSON(response)

	// Parse the JSON response
	var fixResponse FixResponse
	err = json.Unmarshal([]byte(jsonStr), &fixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM correction response as JSON: %w\nResponse: %s", err, response)
	}

	// Validate and filter fixes
	validFixes := af.validateFixes(fixResponse.Fixes)
	fixResponse.Fixes = validFixes

	if af.verbose {
		fmt.Printf("LLM returned %d corrected fix(es)\n", len(fixResponse.Fixes))
	}

	return &fixResponse, nil
}

// validateFixes filters out invalid fixes and warns about them.
func (af *AutoFixer) validateFixes(fixes []Fix) []Fix {
	var valid []Fix

	for _, fix := range fixes {
		// Check required fields using getter methods (handles alternative field names)
		originalCode := fix.GetOriginalCode()
		fixedCode := fix.GetFixedCode()

		if fix.File == "" {
			if af.verbose {
				fmt.Printf("Warning: skipping fix with no file specified\n")
			}
			continue
		}

		if originalCode == "" {
			if af.verbose {
				fmt.Printf("Warning: skipping fix for %s - no original_code provided\n", fix.File)
			}
			continue
		}

		if fixedCode == "" {
			if af.verbose {
				fmt.Printf("Warning: skipping fix for %s - no fixed_code provided\n", fix.File)
			}
			continue
		}

		valid = append(valid, fix)
	}

	return valid
}

// buildFixPrompt constructs the fix generation prompt.
func (af *AutoFixer) buildFixPrompt(
	reviewComments []review.Comment,
	diff string,
	fileContents map[string]string,
) string {
	// Read the fix generation prompt template
	promptTemplate, err := af.readPromptTemplate(af.config.FixPromptFile)
	if err != nil {
		if af.verbose {
			fmt.Printf("Failed to read prompt template: %v, using default\n", err)
		}
		promptTemplate = getDefaultFixPrompt()
	}

	// Format review issues
	var reviewIssues strings.Builder
	for i, comment := range reviewComments {
		reviewIssues.WriteString(fmt.Sprintf("\n### Issue %d\n", i+1))
		reviewIssues.WriteString(fmt.Sprintf("- **File**: %s (Line %d)\n", comment.FilePath, comment.Line))
		reviewIssues.WriteString(fmt.Sprintf("- **Issue**: %s\n", comment.Text))
	}

	// Format file contents
	var fileContentsStr strings.Builder
	for file, content := range fileContents {
		fileContentsStr.WriteString(fmt.Sprintf("\n### File: %s\n\n```go\n%s\n```\n", file, content))
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(promptTemplate, "{REVIEW_ISSUES}", reviewIssues.String())
	prompt = strings.ReplaceAll(prompt, "{DIFF_CONTENT}", diff)
	prompt = strings.ReplaceAll(prompt, "{FILE_CONTENTS}", fileContentsStr.String())

	return prompt
}

// buildCorrectionPrompt constructs the fix correction prompt.
func (af *AutoFixer) buildCorrectionPrompt(
	previousFix *FixResponse,
	verificationError string,
	fileContents map[string]string,
) string {
	// Read the correction prompt template
	promptTemplate, err := af.readPromptTemplate("prompts/fix_correction.md")
	if err != nil {
		if af.verbose {
			fmt.Printf("Failed to read correction prompt template: %v, using default\n", err)
		}
		promptTemplate = getDefaultCorrectionPrompt()
	}

	// Format previous fix as JSON
	previousFixJSON, _ := json.MarshalIndent(previousFix, "", "  ")

	// Format file contents
	var fileContentsStr strings.Builder
	for file, content := range fileContents {
		fileContentsStr.WriteString(fmt.Sprintf("\n### File: %s\n\n```go\n%s\n```\n", file, content))
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(promptTemplate, "{ERROR_OUTPUT}", verificationError)
	prompt = strings.ReplaceAll(prompt, "{PREVIOUS_FIX}", string(previousFixJSON))
	prompt = strings.ReplaceAll(prompt, "{FILE_CONTENT}", fileContentsStr.String())

	return prompt
}

// readPromptTemplate reads a prompt template file.
func (af *AutoFixer) readPromptTemplate(filePath string) (string, error) {
	fullPath := filepath.Join(af.repoPath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// readModifiedFiles reads the current contents of files that were modified.
func (af *AutoFixer) readModifiedFiles(fixes []Fix) (map[string]string, error) {
	contents := make(map[string]string)

	for _, fix := range fixes {
		if _, exists := contents[fix.File]; exists {
			continue // Already read this file
		}

		fullPath := filepath.Join(af.repoPath, fix.File)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", fix.File, err)
		}

		contents[fix.File] = string(content)
	}

	return contents, nil
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// extractJSON extracts JSON from a response that may be wrapped in markdown code fences.
// Handles responses like:
//   - Plain JSON: {...}
//   - Fenced: ```json\n{...}\n```
//   - Text before fence: "Some explanation...\n```json\n{...}\n```"
func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	// Try to find JSON code fence anywhere in response
	fenceStart := strings.Index(response, "```json")
	if fenceStart == -1 {
		fenceStart = strings.Index(response, "```\n{")
	}

	if fenceStart != -1 {
		// Find the start of actual JSON content (after the fence line)
		jsonStart := strings.Index(response[fenceStart:], "\n")
		if jsonStart != -1 {
			jsonStart += fenceStart + 1 // Move past the newline

			// Find the closing fence
			closeFence := strings.Index(response[jsonStart:], "\n```")
			if closeFence != -1 {
				return strings.TrimSpace(response[jsonStart : jsonStart+closeFence])
			}
			// No closing fence found, try to extract to end
			lastFence := strings.LastIndex(response, "```")
			if lastFence > jsonStart {
				return strings.TrimSpace(response[jsonStart:lastFence])
			}
		}
	}

	// No fence found - try to find raw JSON object
	jsonStart := strings.Index(response, "{")
	if jsonStart != -1 {
		// Find matching closing brace (simple approach - find last })
		jsonEnd := strings.LastIndex(response, "}")
		if jsonEnd > jsonStart {
			return strings.TrimSpace(response[jsonStart : jsonEnd+1])
		}
	}

	return response
}

// getDefaultFixPrompt returns a default fix generation prompt.
func getDefaultFixPrompt() string {
	return `# AUTO-FIX CODE GENERATION PROMPT

You are a code fix generator. Generate precise code fixes for the review issues below.

## RULES
1. Generate minimal, surgical fixes - change only what's necessary
2. Preserve original code style and formatting
3. Do NOT introduce new features or refactorings
4. Each fix must directly address a review issue
5. Fixes must be syntactically correct

## OUTPUT FORMAT (MANDATORY)
Respond with ONLY a JSON object (no markdown, no explanation). Use this exact structure:

{
  "fixes": [
    {
      "file": "relative/path/to/file.go",
      "line_start": 45,
      "line_end": 47,
      "original_code": "the exact original code being replaced",
      "fixed_code": "the corrected code",
      "issue_addressed": "Brief description of the fix"
    }
  ],
  "summary": "Brief summary of all fixes"
}

IMPORTANT:
- line_start and line_end are 1-based line numbers
- original_code must match the exact text in the file at those lines
- fixed_code is what will replace original_code
- For single-line fixes, line_start equals line_end

## REVIEW ISSUES TO FIX
{REVIEW_ISSUES}

## ORIGINAL DIFF
{DIFF_CONTENT}

## CURRENT FILE CONTENTS
{FILE_CONTENTS}
`
}

// getDefaultCorrectionPrompt returns a default correction prompt.
func getDefaultCorrectionPrompt() string {
	return `# FIX CORRECTION PROMPT

Your previous fix caused build/test/lint errors. Analyze the errors and provide a corrected fix.

## ERROR OUTPUT
{ERROR_OUTPUT}

## YOUR PREVIOUS FIX
{PREVIOUS_FIX}

## CURRENT FILE CONTENT (all relevant files provided below)
{FILE_CONTENT}

## INSTRUCTIONS

**CRITICAL**: The errors above may be in DIFFERENT files than what you modified. Carefully read:
1. The ERROR OUTPUT to identify which files have problems
2. The FILE CONTENT section which now includes ALL files mentioned in errors
3. You may need to fix files you didn't modify in your previous attempt

**COMMON SCENARIOS**:
- Build errors in file A caused by changes in file B → Fix file A
- Undefined function errors → The function exists, check imports or method receivers
- Formatting errors → Ignore these (they will be auto-fixed)

## OUTPUT FORMAT (MANDATORY - FOLLOW EXACTLY)
Respond with ONLY a valid JSON object. No explanations, no markdown fences, just JSON:

{
  "fixes": [
    {
      "file": "relative/path/to/file.go",
      "line_start": 45,
      "line_end": 47,
      "original_code": "REQUIRED: exact code from CURRENT FILE to replace",
      "fixed_code": "the corrected replacement code",
      "issue_addressed": "Brief description"
    }
  ],
  "summary": "Brief summary"
}

CRITICAL REQUIREMENTS:
1. Look at which FILE has the error - it may NOT be the file you modified before
2. original_code is MANDATORY - copy the exact text from the CURRENT FILE CONTENT above
3. original_code must match actual text in the file character-for-character (including indentation)
4. fixed_code contains your fix for the error
5. Every fix MUST have both original_code and fixed_code fields populated
6. Return fixes for ALL files that need changes, not just the file you modified before
7. If error is "undefined: X", X probably exists somewhere - check imports or package usage
`
}

// CreateStackedPR creates a stacked pull request targeting the original PR branch.
func (af *AutoFixer) CreateStackedPR(
	ctx context.Context,
	fixBranch string,
	originalPR *bitbucket.PullRequest,
	fixResult *FixResult,
) error {
	if af.bbClient == nil {
		return fmt.Errorf("bitbucket client not configured")
	}

	if fixBranch == "" {
		return fmt.Errorf("fix branch name is required")
	}

	if originalPR == nil {
		return fmt.Errorf("original PR is required")
	}

	// Check if PR already exists for this branch
	existingPR, err := af.bbClient.GetPullRequestByBranch(ctx, fixBranch)
	if err != nil {
		return fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if existingPR != nil {
		if af.verbose {
			fmt.Printf("PR already exists for branch %s: %s\n", fixBranch, existingPR.Links.HTML.Href)
		}
		fixResult.PRCreated = true
		fixResult.PRURL = existingPR.Links.HTML.Href
		fixResult.PRNumber = existingPR.ID
		fixResult.BranchName = fixBranch
		return nil
	}

	// Verify branch exists remotely
	branchExists, err := af.bbClient.BranchExists(ctx, fixBranch)
	if err != nil {
		return fmt.Errorf("failed to check if branch exists: %w", err)
	}

	if !branchExists {
		return fmt.Errorf("branch %s does not exist remotely (push it first)", fixBranch)
	}

	// Build PR title and description
	title := af.buildPRTitle(originalPR, fixResult)
	description := af.buildPRDescription(originalPR, fixResult)

	if af.verbose {
		fmt.Printf("Creating stacked PR: %s -> %s\n", fixBranch, originalPR.SourceBranch)
	}

	// Create PR via Bitbucket API
	req := bitbucket.CreatePullRequestRequest{
		Title:             title,
		Description:       description,
		SourceBranch:      fixBranch,
		DestinationBranch: originalPR.SourceBranch,
		CloseSourceBranch: true,
	}

	prResp, err := af.bbClient.CreatePullRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create stacked PR: %w", err)
	}

	// Update fixResult with PR details
	fixResult.PRCreated = true
	fixResult.PRURL = prResp.Links.HTML.Href
	fixResult.PRNumber = prResp.ID
	fixResult.BranchName = fixBranch

	if af.verbose {
		fmt.Printf("✓ Stacked PR created: %s (#%d)\n", fixResult.PRURL, fixResult.PRNumber)
	}

	return nil
}

// buildPRTitle generates the PR title from template.
func (af *AutoFixer) buildPRTitle(originalPR *bitbucket.PullRequest, fixResult *FixResult) string {
	data := map[string]string{
		"pr_id":          strconv.Itoa(originalPR.ID),
		"original_title": originalPR.Title,
		"issue_count":    strconv.Itoa(fixResult.FixesApplied),
	}

	return bitbucket.TemplatePRTitle(af.config.PRTitleTemplate, data)
}

// buildPRDescription generates the PR description from template.
func (af *AutoFixer) buildPRDescription(originalPR *bitbucket.PullRequest, fixResult *FixResult) string {
	data := map[string]string{
		"original_pr_id":   strconv.Itoa(originalPR.ID),
		"original_pr_link": originalPR.Links.HTML.Href,
		"issue_count":      strconv.Itoa(fixResult.FixesApplied),
		"iteration_count":  strconv.Itoa(fixResult.Iterations),
		"file_list":        bitbucket.FormatFileList(fixResult.FilesChanged),
		"build_status":     bitbucket.FormatStatus(fixResult.BuildStatus),
		"test_status":      bitbucket.FormatStatus(fixResult.TestStatus),
		"lint_status":      bitbucket.FormatStatus(fixResult.LintStatus),
		"ai_explanation":   af.getAIExplanation(fixResult),
	}

	return bitbucket.TemplatePRDescription(af.config.PRDescriptionTemplate, data)
}

// getAIExplanation generates a summary of what was fixed.
func (af *AutoFixer) getAIExplanation(fixResult *FixResult) string {
	if fixResult.FixesApplied == 0 {
		return "No fixes applied."
	}

	explanation := fmt.Sprintf("Successfully applied %d fix(es) across %d file(s) after %d iteration(s).",
		fixResult.FixesApplied,
		len(fixResult.FilesChanged),
		fixResult.Iterations,
	)

	if len(fixResult.ErrorMessages) > 0 {
		explanation += "\n\n**Notes:**\n"
		for _, msg := range fixResult.ErrorMessages {
			if msg != "Max iterations exceeded" {
				explanation += fmt.Sprintf("- %s\n", msg)
			}
		}
	}

	return explanation
}

// parseErrorFiles extracts file paths from verification error messages.
// Handles Go compiler/vet error formats like:
//   - "cmd/pullreview/main.go:180:6: undefined: llm.SetVerbose"
//   - "internal/bitbucket/client.go" (from gofmt)
//   - "# pullreview/cmd/pullreview" (package header, skip)
func (af *AutoFixer) parseErrorFiles(errorOutput string) []string {
	fileSet := make(map[string]bool)
	lines := strings.Split(errorOutput, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip package headers like "# pullreview/cmd/pullreview"
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Skip lines that are just error descriptions
		if strings.Contains(line, "failed:") || strings.Contains(line, "check failed") {
			continue
		}

		// Pattern 1: "file.go:line:col: error message"
		if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
			filePath := line[:colonIdx]

			// Check if this looks like a file path (contains .go or other extension)
			if strings.Contains(filePath, ".go") ||
				strings.Contains(filePath, ".py") ||
				strings.Contains(filePath, ".js") ||
				strings.Contains(filePath, ".ts") {
				// Normalize path separators
				filePath = filepath.ToSlash(filePath)
				fileSet[filePath] = true
				continue
			}
		}

		// Pattern 2: Just a file path (from gofmt output)
		// Example: "internal\bitbucket\client.go"
		if strings.Contains(line, ".go") ||
			strings.Contains(line, ".py") ||
			strings.Contains(line, ".js") ||
			strings.Contains(line, ".ts") {
			// Check if it's a valid relative path (no spaces, no special chars)
			if !strings.Contains(line, " ") && !strings.Contains(line, ":") {
				// Normalize path separators
				filePath := filepath.ToSlash(line)
				fileSet[filePath] = true
			}
		}
	}

	// Convert set to slice
	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}

	return files
}

// autoFormatFiles runs gofmt on the specified files.
func (af *AutoFixer) autoFormatFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Run gofmt -s -w on each Go file
	for _, file := range files {
		// Only format Go files
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		absPath := filepath.Join(af.repoPath, file)

		if af.verbose {
			fmt.Printf("  Formatting: %s\n", file)
		}

		// Execute gofmt
		cmd := exec.Command("gofmt", "-s", "-w", absPath)
		cmd.Dir = af.repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("gofmt failed for %s: %w\nOutput: %s", file, err, string(output))
		}

		if af.verbose && len(output) > 0 {
			fmt.Printf("  gofmt output: %s\n", string(output))
		}
	}

	return nil
}
