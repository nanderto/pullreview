package autofix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AutoFixConfig holds configuration for auto-fix operations.
type AutoFixConfig struct {
	Enabled               bool   `yaml:"enabled"`
	AutoCreatePR          bool   `yaml:"auto_create_pr"`
	MaxIterations         int    `yaml:"max_iterations"`
	VerifyBuild           bool   `yaml:"verify_build"`
	VerifyTests           bool   `yaml:"verify_tests"`
	VerifyLint            bool   `yaml:"verify_lint"`
	PipelineMode          bool   `yaml:"pipeline_mode"`
	BranchPrefix          string `yaml:"branch_prefix"`
	AutofixPromptFile     string `yaml:"autofix_prompt_file"` // Combined find+fix prompt
	FixPromptFile         string `yaml:"fix_prompt_file"`     // Fix existing comments prompt
	CommitMessageTemplate string `yaml:"commit_message_template"`
	PRTitleTemplate       string `yaml:"pr_title_template"`
	PRDescriptionTemplate string `yaml:"pr_description_template"`
}

// SetDefaults sets sensible defaults for auto-fix config.
func (c *AutoFixConfig) SetDefaults() {
	if c.MaxIterations == 0 {
		c.MaxIterations = 5
	}
	if c.BranchPrefix == "" {
		c.BranchPrefix = "pullreview-fixes"
	}
	if c.AutofixPromptFile == "" {
		c.AutofixPromptFile = "autofix_prompt.md"
	}
	if c.FixPromptFile == "" {
		c.FixPromptFile = "fix_prompt.md"
	}
	if c.CommitMessageTemplate == "" {
		c.CommitMessageTemplate = `🤖 Auto-fix: {issue_summary}

Iterations: {iteration_count}
Tests: {test_status}
Lint: {lint_status}`
	}
	if c.PRTitleTemplate == "" {
		c.PRTitleTemplate = "🤖 Auto-fixes for PR #{pr_id}: {original_title}"
	}
	if c.PRDescriptionTemplate == "" {
		c.PRDescriptionTemplate = `## Auto-generated fixes for PR #{original_pr_id}

**Original PR:** {original_pr_link}
**Issues Fixed:** {issue_count}
**Iterations Required:** {iteration_count}

### Changes Made:
{file_list}

### Build Verification:
- Build: {build_status}
- Tests: {test_status}
- Lint: {lint_status}

### AI Summary:
{ai_explanation}`
	}
}

// FixResult represents the outcome of applying fixes.
type FixResult struct {
	Success       bool
	FilesChanged  []string
	FixesApplied  int
	FixesFailed   int
	Iterations    int
	BuildStatus   string
	TestStatus    string
	LintStatus    string
	ErrorMessages []string
	PRCreated     bool
	PRURL         string
	PRNumber      int
	BranchName    string
}

// Fix represents a single code fix.
type Fix struct {
	File           string `json:"file"`
	LineStart      int    `json:"line_start"`
	LineEnd        int    `json:"line_end"`
	OriginalCode   string `json:"original_code"`
	FixedCode      string `json:"fixed_code"`
	IssueAddressed string `json:"issue_addressed"`
	// Alternative field names some LLMs use
	OldCode string `json:"old_code,omitempty"`
	NewCode string `json:"new_code,omitempty"`
}

// GetOriginalCode returns the original code, checking alternative field names.
func (f *Fix) GetOriginalCode() string {
	if f.OriginalCode != "" {
		return f.OriginalCode
	}
	return f.OldCode
}

// GetFixedCode returns the fixed code, checking alternative field names.
func (f *Fix) GetFixedCode() string {
	if f.FixedCode != "" {
		return f.FixedCode
	}
	return f.NewCode
}

// FixResponse represents the LLM's response containing fixes.
type FixResponse struct {
	Fixes   []Fix  `json:"fixes"`
	Summary string `json:"summary"`
}

// Applier applies fixes to files.
type Applier struct {
	repoPath string
	verbose  bool
	backups  map[string][]byte
}

// NewApplier creates a new Applier instance.
func NewApplier(repoPath string) *Applier {
	return &Applier{
		repoPath: repoPath,
		verbose:  false,
		backups:  make(map[string][]byte),
	}
}

// SetVerbose enables debug output.
func (a *Applier) SetVerbose(v bool) {
	a.verbose = v
}

// RestoreBackups restores all backed up files.
func (a *Applier) RestoreBackups() error {
	var errs []string

	for file, content := range a.backups {
		fullPath := filepath.Join(a.repoPath, file)
		if a.verbose {
			fmt.Printf("Restoring backup: %s\n", file)
		}

		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			errs = append(errs, fmt.Sprintf("failed to restore %s: %v", file, err))
		}
	}

	// Clear backups after restore attempt
	a.backups = make(map[string][]byte)

	if len(errs) > 0 {
		return fmt.Errorf("restore errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// ClearBackups clears backup storage.
func (a *Applier) ClearBackups() {
	a.backups = make(map[string][]byte)
}

// ApplyFixes applies a list of fixes to files.
// Returns the list of modified file paths.
// Uses search-and-replace approach (more robust than line numbers since LLMs often get those wrong).
func (a *Applier) ApplyFixes(fixes []Fix) ([]string, error) {
	if len(fixes) == 0 {
		return nil, nil
	}

	modifiedFiles := make(map[string]bool)

	// Group fixes by file
	fixesByFile := make(map[string][]Fix)
	for _, fix := range fixes {
		fixesByFile[fix.File] = append(fixesByFile[fix.File], fix)
	}

	for file, fileFixes := range fixesByFile {
		fullPath := filepath.Join(a.repoPath, file)

		// Read file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Backup original content (only on first modification)
		if _, backed := a.backups[file]; !backed {
			a.backups[file] = content
		}

		fileContent := string(content)

		// Apply each fix using search-and-replace
		for _, fix := range fileFixes {
			if a.verbose {
				fmt.Printf("Applying fix to %s: %s\n", file, fix.IssueAddressed)
			}

			originalCode := fix.GetOriginalCode()
			fixedCode := fix.GetFixedCode()

			if originalCode == "" {
				return nil, fmt.Errorf("fix for %s has no original_code - cannot apply", file)
			}

			// Try to find and replace the original code
			newContent, found := a.searchAndReplace(fileContent, originalCode, fixedCode)
			if !found {
				// Try with normalized whitespace
				newContent, found = a.searchAndReplaceNormalized(fileContent, originalCode, fixedCode)
			}

			if !found {
				return nil, fmt.Errorf("could not find original code in %s\nSearching for:\n%s",
					file, originalCode)
			}

			fileContent = newContent
		}

		// Write back to file
		if err := os.WriteFile(fullPath, []byte(fileContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", file, err)
		}

		modifiedFiles[file] = true

		if a.verbose {
			fmt.Printf("✓ Modified %s\n", file)
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(modifiedFiles))
	for f := range modifiedFiles {
		result = append(result, f)
	}

	return result, nil
}

// searchAndReplace tries to find originalCode in content and replace with fixedCode.
// Returns the new content and whether the replacement was made.
func (a *Applier) searchAndReplace(content, originalCode, fixedCode string) (string, bool) {
	// Direct match
	if strings.Contains(content, originalCode) {
		return strings.Replace(content, originalCode, fixedCode, 1), true
	}
	return content, false
}

// searchAndReplaceNormalized tries to find originalCode with flexible whitespace matching.
func (a *Applier) searchAndReplaceNormalized(content, originalCode, fixedCode string) (string, bool) {
	// Try trimming leading/trailing whitespace from each line
	originalLines := strings.Split(originalCode, "\n")
	contentLines := strings.Split(content, "\n")

	// Build a pattern with trimmed lines
	var trimmedOriginal []string
	for _, line := range originalLines {
		trimmedOriginal = append(trimmedOriginal, strings.TrimSpace(line))
	}

	// Search for the pattern in content
	for i := 0; i <= len(contentLines)-len(originalLines); i++ {
		match := true
		for j, origLine := range trimmedOriginal {
			if strings.TrimSpace(contentLines[i+j]) != origLine {
				match = false
				break
			}
		}

		if match {
			// Found match at line i - replace those lines
			// Preserve indentation from the first matched line
			indent := getLeadingWhitespace(contentLines[i])

			// Apply indent to fixed code
			fixedLines := strings.Split(fixedCode, "\n")
			var indentedFixed []string
			for k, line := range fixedLines {
				if k == 0 {
					// First line gets the original indent
					indentedFixed = append(indentedFixed, indent+strings.TrimSpace(line))
				} else if strings.TrimSpace(line) == "" {
					indentedFixed = append(indentedFixed, "")
				} else {
					// Subsequent lines: try to preserve relative indentation
					indentedFixed = append(indentedFixed, indent+strings.TrimSpace(line))
				}
			}

			// Build new content
			newLines := make([]string, 0, len(contentLines)-len(originalLines)+len(fixedLines))
			newLines = append(newLines, contentLines[:i]...)
			newLines = append(newLines, indentedFixed...)
			newLines = append(newLines, contentLines[i+len(originalLines):]...)

			return strings.Join(newLines, "\n"), true
		}
	}

	return content, false
}

// getLeadingWhitespace returns the leading whitespace of a string.
func getLeadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}
