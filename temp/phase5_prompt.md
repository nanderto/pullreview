# Phase 5: CLI & Configuration Integration

## Objective

Integrate the auto-fix workflow into the CLI with dedicated subcommands, configuration options, and pipeline mode detection, completing the user-facing interface for the auto-fix feature.

---

## Context

**Project:** Go CLI tool for code review automation  
**Current Status:** Phases 1-4 complete  
- ✅ Phase 1: Fix generation and parsing (LLM integration, JSON parsing)
- ✅ Phase 2: Fix application and verification (file modifications, build/test verification, iterative loop)
- ✅ Phase 3: Git operations (branch creation, commits, push)
- ✅ Phase 4: Stacked PR creation (Bitbucket API integration)
- 🎯 Phase 5: CLI & Configuration (this phase)

**What exists:**
- `internal/autofix/` - Complete fix generation, application, and orchestration
- `internal/verify/` - Build/test/lint verification
- `internal/git/` - Git operations (branch, commit, push)
- `internal/bitbucket/` - Bitbucket API client (fetch PR, diff, metadata, post comments, create PRs)
- `internal/llm/` - LLM client for fix generation
- `internal/config/` - Configuration management
- `cmd/pullreview/main.go` - Existing CLI with review subcommand

**What's needed:**
CLI integration to make auto-fix functionality accessible to users with proper configuration and pipeline mode detection.

---

## Scope

### Files to Modify

1. **`cmd/pullreview/main.go`** - Add `fix-pr` subcommand and `--auto-fix` flag
2. **`internal/config/config.go`** - Add AutoFix configuration section
3. **`pullreview.yaml`** - Add example AutoFix configuration
4. **`README.md`** - Document new commands and configuration

### Files to Create (Optional)

1. **`docs/CLI_USAGE.md`** - Detailed CLI usage guide
2. **`examples/pullreview-autofix.yaml`** - Example configuration for auto-fix

---

## Detailed Requirements

### 1. CLI Structure (`cmd/pullreview/main.go`)

Add a new `fix-pr` subcommand alongside the existing root command:

```
pullreview                    # Existing: Review PR and post comments
pullreview --auto-fix         # New: Review + apply fixes + create stacked PR
pullreview fix-pr             # New: Apply fixes to current PR (dedicated command)
pullreview fix-pr --pr 123    # New: Apply fixes to specific PR
```

#### Command Structure

```go
rootCmd := &cobra.Command{
	Use:   "pullreview [flags]",
	Short: "Automated code review for Bitbucket Cloud PRs using LLMs",
	Long:  "pullreview fetches Bitbucket Cloud PR diffs, sends them to an LLM for review, and posts AI-generated comments back to Bitbucket.",
	RunE:  runPullReview,
}

fixPRCmd := &cobra.Command{
	Use:   "fix-pr [flags]",
	Short: "Auto-fix issues in a pull request and create stacked PR",
	Long: `Fetches a Bitbucket Cloud PR, generates fixes using LLM, applies them,
verifies build/test/lint, and creates a stacked pull request with the fixes.`,
	RunE:  runFixPR,
}

rootCmd.AddCommand(fixPRCmd)
```

#### Flags for `fix-pr` subcommand

```go
// Shared flags (inherited from root)
fixPRCmd.Flags().StringVarP(&cfgFile, "config", "c", defaultConfig, "Path to config file")
fixPRCmd.Flags().StringVar(&prID, "pr", "", "Bitbucket Pull Request ID (overrides branch inference)")
fixPRCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

// Auto-fix specific flags
fixPRCmd.Flags().Bool("dry-run", false, "Apply fixes locally without committing or creating PR")
fixPRCmd.Flags().Bool("skip-verification", false, "Skip build/test/lint verification (dangerous)")
fixPRCmd.Flags().Int("max-iterations", 0, "Maximum fix iterations (0 = use config default)")
fixPRCmd.Flags().String("branch-prefix", "", "Branch name prefix (default: from config)")
fixPRCmd.Flags().Bool("no-pr", false, "Don't create stacked PR (just fix locally)")
```

#### Add `--auto-fix` flag to root command

```go
rootCmd.Flags().Bool("auto-fix", false, "Apply fixes after review and create stacked PR")
```

### 2. Configuration Schema (`internal/config/config.go`)

Add AutoFix configuration section:

```go
type Config struct {
	Bitbucket struct {
		Email     string `yaml:"email"`
		APIToken  string `yaml:"api_token"`
		Workspace string `yaml:"workspace"`
		RepoSlug  string `yaml:"repo_slug"`
		BaseURL   string `yaml:"base_url"`
	} `yaml:"bitbucket"`

	LLM struct {
		Provider string `yaml:"provider"`
		APIKey   string `yaml:"api_key"`
		Endpoint string `yaml:"endpoint"`
		Model    string `yaml:"model"`
	} `yaml:"llm"`

	PromptFile string `yaml:"prompt_file"`

	// NEW: AutoFix configuration
	AutoFix struct {
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
	} `yaml:"autofix"`
}
```

Add method to convert to `autofix.AutoFixConfig`:

```go
// ToAutoFixConfig converts Config.AutoFix to autofix.AutoFixConfig.
func (c *Config) ToAutoFixConfig() *autofix.AutoFixConfig {
	cfg := &autofix.AutoFixConfig{
		Enabled:               c.AutoFix.Enabled,
		AutoCreatePR:          c.AutoFix.AutoCreatePR,
		MaxIterations:         c.AutoFix.MaxIterations,
		VerifyBuild:           c.AutoFix.VerifyBuild,
		VerifyTests:           c.AutoFix.VerifyTests,
		VerifyLint:            c.AutoFix.VerifyLint,
		PipelineMode:          c.AutoFix.PipelineMode,
		BranchPrefix:          c.AutoFix.BranchPrefix,
		FixPromptFile:         c.AutoFix.FixPromptFile,
		CommitMessageTemplate: c.AutoFix.CommitMessageTemplate,
		PRTitleTemplate:       c.AutoFix.PRTitleTemplate,
		PRDescriptionTemplate: c.AutoFix.PRDescriptionTemplate,
	}
	cfg.SetDefaults()
	return cfg
}
```

### 3. Pipeline Mode Detection

Detect if running in CI/CD pipeline and adjust behavior:

```go
// DetectPipelineMode checks environment variables to determine if running in CI/CD.
func DetectPipelineMode() bool {
	ciEnvVars := []string{
		"CI",                    // Generic CI indicator
		"BITBUCKET_PIPELINE",    // Bitbucket Pipelines
		"GITHUB_ACTIONS",        // GitHub Actions
		"GITLAB_CI",             // GitLab CI
		"JENKINS_HOME",          // Jenkins
		"CIRCLECI",              // CircleCI
		"TRAVIS",                // Travis CI
		"AZURE_PIPELINES",       // Azure Pipelines
		"BUDDY_WORKSPACE_ID",    // Buddy
		"TEAMCITY_VERSION",      // TeamCity
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}
```

**Pipeline Mode Behavior:**
- Auto-enable `--verbose` for better logs
- Disable interactive prompts
- Always exit with non-zero code on failure
- Output machine-readable summary at end

### 4. Implementation: `runFixPR` function

```go
func runFixPR(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfigWithOverrides(cfgFile, bbEmail, bbAPIToken)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI flag overrides to AutoFix config
	autoFixCfg := cfg.ToAutoFixConfig()
	
	if maxIter, _ := cmd.Flags().GetInt("max-iterations"); maxIter > 0 {
		autoFixCfg.MaxIterations = maxIter
	}
	
	if prefix, _ := cmd.Flags().GetString("branch-prefix"); prefix != "" {
		autoFixCfg.BranchPrefix = prefix
	}
	
	if skipVerify, _ := cmd.Flags().GetBool("skip-verification"); skipVerify {
		autoFixCfg.VerifyBuild = false
		autoFixCfg.VerifyTests = false
		autoFixCfg.VerifyLint = false
	}
	
	if noPR, _ := cmd.Flags().GetBool("no-pr"); noPR {
		autoFixCfg.AutoCreatePR = false
	}

	// Detect pipeline mode
	if config.DetectPipelineMode() {
		autoFixCfg.PipelineMode = true
		verbose = true
		fmt.Println("🤖 Pipeline mode detected")
	}

	// Validate configuration
	if !autoFixCfg.Enabled && !cmd.Flags().Changed("dry-run") {
		fmt.Fprintln(os.Stderr, "⚠️  AutoFix is disabled in config. Use --dry-run to override.")
	}

	// Get repo path
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}

	// Initialize clients
	bbClient := bitbucket.NewClient(
		cfg.Bitbucket.Email,
		cfg.Bitbucket.APIToken,
		cfg.Bitbucket.Workspace,
		cfg.Bitbucket.RepoSlug,
		cfg.Bitbucket.BaseURL,
	)

	if err := bbClient.Authenticate(); err != nil {
		return fmt.Errorf("bitbucket authentication failed: %w", err)
	}

	llmClient, err := llm.NewClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Endpoint, cfg.LLM.Model)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	// Determine PR ID
	finalPRID, err := determinePRID(prID, repoPath, bbClient)
	if err != nil {
		return err
	}

	fmt.Printf("🔧 Auto-fixing PR #%s...\n", finalPRID)

	// Fetch PR details
	originalPR, err := bbClient.GetPullRequest(context.Background(), finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR: %w", err)
	}

	// Fetch PR diff
	diff, err := bbClient.GetPRDiff(finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	// Fetch review comments (re-use review logic)
	reviewComments, err := getReviewComments(cfg, llmClient, finalPRID, diff)
	if err != nil {
		return fmt.Errorf("failed to generate review: %w", err)
	}

	if len(reviewComments) == 0 {
		fmt.Println("✅ No issues found - nothing to fix!")
		return nil
	}

	fmt.Printf("📝 Found %d issue(s) to fix\n", len(reviewComments))

	// Initialize AutoFixer
	autofixer := autofix.NewAutoFixer(autoFixCfg, llmClient, repoPath)
	autofixer.SetVerbose(verbose)
	autofixer.SetBitbucketClient(bbClient)

	// Get file contents for context
	fileContents, err := getFileContents(repoPath, reviewComments)
	if err != nil {
		return fmt.Errorf("failed to read file contents: %w", err)
	}

	// Apply fixes
	ctx := context.Background()
	fixResult, err := autofixer.GenerateAndApplyFixes(ctx, reviewComments, diff, fileContents)
	if err != nil {
		return fmt.Errorf("fix generation failed: %w", err)
	}

	if !fixResult.Success {
		fmt.Fprintln(os.Stderr, "❌ Fixes failed verification")
		for _, errMsg := range fixResult.ErrorMessages {
			fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
		}
		if autoFixCfg.PipelineMode {
			os.Exit(1)
		}
		return fmt.Errorf("fixes did not pass verification")
	}

	fmt.Printf("✅ Applied %d fix(es) to %d file(s)\n", fixResult.FixesApplied, len(fixResult.FilesChanged))

	// Check if dry-run
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Println("🏁 Dry-run mode: Fixes applied locally. Review changes with 'git diff'")
		return nil
	}

	// Create branch, commit, push
	gitOps := git.NewOperations(repoPath)
	currentBranch, err := gitOps.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	fixBranch := gitOps.GenerateBranchName(currentBranch, autoFixCfg.BranchPrefix)
	
	if err := gitOps.CreateBranch(fixBranch); err != nil {
		return fmt.Errorf("failed to create fix branch: %w", err)
	}

	if err := gitOps.Checkout(fixBranch); err != nil {
		return fmt.Errorf("failed to checkout fix branch: %w", err)
	}

	if err := gitOps.StageFiles(fixResult.FilesChanged); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	commitMsg := buildCommitMessage(autoFixCfg.CommitMessageTemplate, fixResult)
	if err := gitOps.Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit fixes: %w", err)
	}

	if err := gitOps.Push(fixBranch); err != nil {
		return fmt.Errorf("failed to push fix branch: %w", err)
	}

	fmt.Printf("✅ Pushed fixes to branch: %s\n", fixBranch)

	// Create stacked PR if enabled
	if autoFixCfg.AutoCreatePR {
		err := autofixer.CreateStackedPR(ctx, fixBranch, originalPR, fixResult)
		if err != nil {
			return fmt.Errorf("failed to create stacked PR: %w", err)
		}

		fmt.Printf("✅ Stacked PR created: %s\n", fixResult.PRURL)
	} else {
		fmt.Println("ℹ️  Stacked PR creation disabled. Push branch manually if needed.")
	}

	// Output summary
	printFixSummary(fixResult, autoFixCfg.PipelineMode)

	return nil
}
```

### 5. Helper Functions

```go
// determinePRID resolves the PR ID from CLI arg or git branch
func determinePRID(cliPRID, repoPath string, bbClient *bitbucket.Client) (string, error) {
	if cliPRID != "" {
		return cliPRID, nil
	}

	branch, err := utils.GetCurrentGitBranch(repoPath)
	if err != nil {
		return "", fmt.Errorf("could not infer git branch: %w", err)
	}

	prID, err := bbClient.GetPRIDByBranch(branch)
	if err != nil {
		return "", fmt.Errorf("could not find open PR for branch %q: %w", branch, err)
	}

	return prID, nil
}

// buildCommitMessage generates commit message from template
func buildCommitMessage(template string, fixResult *autofix.FixResult) string {
	msg := template
	msg = strings.ReplaceAll(msg, "{issue_summary}", fmt.Sprintf("Applied %d fix(es)", fixResult.FixesApplied))
	msg = strings.ReplaceAll(msg, "{iteration_count}", fmt.Sprintf("%d", fixResult.Iterations))
	msg = strings.ReplaceAll(msg, "{test_status}", fixResult.TestStatus)
	msg = strings.ReplaceAll(msg, "{lint_status}", fixResult.LintStatus)
	return msg
}

// printFixSummary outputs a summary of the fix operation
func printFixSummary(fixResult *autofix.FixResult, pipelineMode bool) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("FIX SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Fixes Applied:  %d\n", fixResult.FixesApplied)
	fmt.Printf("Files Changed:  %d\n", len(fixResult.FilesChanged))
	fmt.Printf("Iterations:     %d\n", fixResult.Iterations)
	fmt.Printf("Build Status:   %s\n", fixResult.BuildStatus)
	fmt.Printf("Test Status:    %s\n", fixResult.TestStatus)
	fmt.Printf("Lint Status:    %s\n", fixResult.LintStatus)
	
	if fixResult.PRCreated {
		fmt.Printf("Stacked PR:     %s\n", fixResult.PRURL)
	}
	
	fmt.Println(strings.Repeat("=", 60))

	// Machine-readable output for CI/CD
	if pipelineMode {
		output := map[string]interface{}{
			"success":       fixResult.Success,
			"fixes_applied": fixResult.FixesApplied,
			"files_changed": fixResult.FilesChanged,
			"iterations":    fixResult.Iterations,
			"pr_url":        fixResult.PRURL,
			"pr_number":     fixResult.PRNumber,
			"branch_name":   fixResult.BranchName,
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println("\nMACHINE_READABLE_OUTPUT:")
		fmt.Println(string(jsonBytes))
	}
}

// getFileContents reads file contents for review context
func getFileContents(repoPath string, comments []review.Comment) (map[string]string, error) {
	contents := make(map[string]string)
	
	for _, comment := range comments {
		if _, exists := contents[comment.FilePath]; exists {
			continue
		}
		
		fullPath := filepath.Join(repoPath, comment.FilePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", comment.FilePath, err)
		}
		
		contents[comment.FilePath] = string(data)
	}
	
	return contents, nil
}

// getReviewComments generates review comments using LLM
func getReviewComments(cfg *config.Config, llmClient *llm.Client, prID, diff string) ([]review.Comment, error) {
	// Read prompt template
	promptData, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}
	
	prompt := strings.ReplaceAll(string(promptData), "{DIFF}", diff)
	prompt = strings.ReplaceAll(prompt, "{PR_ID}", prID)
	
	// Send to LLM
	response, err := llmClient.SendPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	
	// Parse review comments
	comments, err := review.ParseReviewComments(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}
	
	return comments, nil
}
```

### 6. Update `pullreview.yaml` Configuration

```yaml
bitbucket:
  email: user@example.com
  api_token: ${BITBUCKET_API_TOKEN}
  workspace: my-workspace
  repo_slug: my-repo  # Auto-detected if not specified
  base_url: https://api.bitbucket.org/2.0

llm:
  provider: copilot  # or openai, openrouter, github
  api_key: ${LLM_API_KEY}  # Not needed for copilot
  endpoint: https://api.openai.com/v1/chat/completions
  model: gpt-4

prompt_file: prompt.md

# Auto-fix configuration
autofix:
  enabled: true
  auto_create_pr: true
  max_iterations: 5
  verify_build: true
  verify_tests: true
  verify_lint: true
  pipeline_mode: false  # Auto-detected in CI/CD
  branch_prefix: pullreview-fixes
  fix_prompt_file: prompts/fix_generation.md
  commit_message_template: |
    🤖 Auto-fix: {issue_summary}
    
    Iterations: {iteration_count}
    Tests: {test_status}
    Lint: {lint_status}
  pr_title_template: "🤖 Auto-fixes for PR #{pr_id}: {original_title}"
  pr_description_template: |
    ## Auto-generated fixes for PR #{original_pr_id}
    
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
    {ai_explanation}
```

### 7. Update README.md

Add documentation for new commands:

````markdown
## Auto-Fix Usage

### Apply Fixes to Current PR

```bash
# Auto-fix issues and create stacked PR
pullreview fix-pr

# Specify PR ID explicitly
pullreview fix-pr --pr 123

# Dry-run mode (apply fixes locally without committing)
pullreview fix-pr --dry-run

# Skip verification (dangerous - not recommended)
pullreview fix-pr --skip-verification

# Apply fixes without creating PR
pullreview fix-pr --no-pr
```

### Combined Review + Auto-Fix

```bash
# Review PR and auto-fix in one command
pullreview --auto-fix --pr 123
```

### Configuration

Enable auto-fix in `pullreview.yaml`:

```yaml
autofix:
  enabled: true
  auto_create_pr: true
  max_iterations: 5
  verify_build: true
  verify_tests: true
  verify_lint: true
```

### Pipeline Mode

Auto-fix works seamlessly in CI/CD pipelines:

```yaml
# Bitbucket Pipelines example
pipelines:
  pull-requests:
    '**':
      - step:
          name: Auto-fix PR
          script:
            - ./pullreview fix-pr
```

Pipeline mode is auto-detected and provides:
- Verbose logging
- Machine-readable JSON output
- Non-zero exit codes on failure
- No interactive prompts
````

---

## Test Requirements

### CLI Tests

Create integration test script `test/test_cli.sh`:

```bash
#!/bin/bash

# Test fix-pr command
echo "Testing: pullreview fix-pr --help"
./pullreview fix-pr --help || exit 1

# Test dry-run mode
echo "Testing: pullreview fix-pr --dry-run"
./pullreview fix-pr --dry-run --pr 123 || exit 1

# Test configuration loading
echo "Testing: pullreview fix-pr --config test/fixtures/test-config.yaml"
./pullreview fix-pr --config test/fixtures/test-config.yaml --dry-run || exit 1
```

### Configuration Tests

Add tests to `internal/config/config_test.go`:

```go
func TestToAutoFixConfig(t *testing.T)
func TestDetectPipelineMode(t *testing.T)
func TestConfigWithAutoFixDefaults(t *testing.T)
```

---

## Validation Commands

After implementation, run:

```bash
go vet ./...
gofmt -s -w .
go build ./...
go test ./...
./pullreview --help
./pullreview fix-pr --help
```

All must pass before Phase 5 is considered complete.

---

## Success Criteria

- [ ] `pullreview fix-pr` command works
- [ ] `pullreview --auto-fix` flag works
- [ ] Configuration loads from `pullreview.yaml`
- [ ] CLI flags override config values
- [ ] Pipeline mode auto-detected in CI/CD
- [ ] Dry-run mode applies fixes without committing
- [ ] `--help` documentation is clear and complete
- [ ] README updated with usage examples
- [ ] All tests pass
- [ ] Code is formatted and passes vet

---

## Example Usage Scenarios

### Scenario 1: Developer Local Workflow

```bash
# Developer creates PR
git checkout -b feature/add-logger
git push origin feature/add-logger

# PR created in Bitbucket

# Auto-fix issues found in review
cd /path/to/repo
pullreview fix-pr

# Output:
# 🔧 Auto-fixing PR #456...
# 📝 Found 3 issue(s) to fix
# ✅ Applied 3 fix(es) to 2 file(s)
# ✅ Pushed fixes to branch: pullreview-fixes-feature-add-logger-20260212T153042Z
# ✅ Stacked PR created: https://bitbucket.org/...

# Developer reviews stacked PR, merges to their branch
```

### Scenario 2: CI/CD Pipeline

```yaml
# bitbucket-pipelines.yml
pipelines:
  pull-requests:
    '**':
      - step:
          name: Code Review & Auto-Fix
          script:
            - curl -L https://github.com/.../pullreview/releases/download/v1.0.0/pullreview-linux-amd64 -o pullreview
            - chmod +x pullreview
            - ./pullreview fix-pr
          after-script:
            - echo "Fix summary available in pipeline logs"
```

### Scenario 3: Review Without Auto-Fix

```bash
# Just review (existing behavior)
pullreview --pr 123 --post

# Review and show proposed fixes (but don't apply)
pullreview --pr 123 --generate-fixes
```

---

## Anti-Patterns to Avoid

- ❌ Don't make auto-fix enabled by default (opt-in only)
- ❌ Don't commit unverified fixes (always require verification unless explicitly skipped)
- ❌ Don't modify the original PR branch (always create stacked PR)
- ❌ Don't hide errors in pipeline mode (must exit with non-zero code)
- ❌ Don't require interactive input in pipeline mode
- ❌ Don't hardcode configuration values (use config file + env vars + CLI flags)

---

## API Documentation References

- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Viper Configuration](https://github.com/spf13/viper) (optional enhancement)
- [Go Flag Package](https://pkg.go.dev/flag)

---

## Reference

See [docs/auto_fix_implementation_plan.md](../docs/auto_fix_implementation_plan.md) for full context.

Project follows Go best practices as defined in [AGENTS.md](../AGENTS.md).

---

## Notes

- CLI follows POSIX conventions (flags before args)
- Configuration precedence: CLI flags > env vars > config file > defaults
- Pipeline mode detection covers all major CI/CD platforms
- Dry-run mode useful for testing locally before committing
- Machine-readable output in pipeline mode enables downstream tools
- All operations are idempotent (safe to re-run)
