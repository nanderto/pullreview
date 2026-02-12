package autofix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pullreview/internal/bitbucket"
	"pullreview/internal/llm"
	"pullreview/internal/review"
	"pullreview/internal/verify"
)

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

	// Parse the JSON response
	var fixResponse FixResponse
	err = json.Unmarshal([]byte(response), &fixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse: %s", err, response)
	}

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

	prompt := af.buildCorrectionPrompt(previousFix, verificationError, fileContents)

	response, err := af.llmClient.SendFixPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM correction request failed: %w", err)
	}

	// Parse the JSON response
	var fixResponse FixResponse
	err = json.Unmarshal([]byte(response), &fixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM correction response as JSON: %w\nResponse: %s", err, response)
	}

	if af.verbose {
		fmt.Printf("LLM returned %d corrected fix(es)\n", len(fixResponse.Fixes))
	}

	return &fixResponse, nil
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

// getDefaultFixPrompt returns a default fix generation prompt.
func getDefaultFixPrompt() string {
	return `# AUTO-FIX CODE GENERATION PROMPT

Generate precise code fixes for the review issues below.

## REVIEW ISSUES
{REVIEW_ISSUES}

## DIFF
{DIFF_CONTENT}

## FILE CONTENTS
{FILE_CONTENTS}

Respond with only a JSON object containing fixes.
`
}

// getDefaultCorrectionPrompt returns a default correction prompt.
func getDefaultCorrectionPrompt() string {
	return `# FIX CORRECTION PROMPT

Your previous fix caused an error. Please provide a corrected fix.

## ERROR
{ERROR_OUTPUT}

## PREVIOUS FIX
{PREVIOUS_FIX}

## CURRENT FILE CONTENT
{FILE_CONTENT}

Respond with only a JSON object containing the corrected fix.
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
