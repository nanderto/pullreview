package autofix

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
	FixPromptFile         string `yaml:"fix_prompt_file"`
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
	if c.FixPromptFile == "" {
		c.FixPromptFile = "prompts/fix_generation.md"
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
	// Implementation placeholder
	return nil
}

// ClearBackups clears backup storage.
func (a *Applier) ClearBackups() {
	a.backups = make(map[string][]byte)
}

// ApplyFixes applies a list of fixes to files.
// Returns the list of modified file paths.
func (a *Applier) ApplyFixes(fixes []Fix) ([]string, error) {
	// Placeholder implementation
	modifiedFiles := make([]string, 0)
	for _, fix := range fixes {
		modifiedFiles = append(modifiedFiles, fix.File)
	}
	return modifiedFiles, nil
}
