package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"pullreview/internal/autofix"
	"pullreview/internal/bitbucket"
	"pullreview/internal/config"
	"pullreview/internal/git"
	"pullreview/internal/llm"
	"pullreview/internal/review"
	"pullreview/internal/utils"
	"strings"
)

var (
	cfgFile     string
	prID        string
	bbEmail     string
	bbAPIToken  string
	showVersion bool
	verbose     bool
	postToBB    bool
	version     = "0.1.0"
)

func main() {
	// Determine default config path: executable directory
	defaultConfig := "pullreview.yaml"
	if exePath, err := os.Executable(); err == nil {
		exeDir := ""
		if exePath != "" {
			exeDir = filepath.Dir(exePath)
			defaultConfig = filepath.Join(exeDir, "pullreview.yaml")
		}
	}

	rootCmd := &cobra.Command{
		Use:   "pullreview",
		Short: "Automated code review for Bitbucket Cloud PRs using LLMs",
		Long:  "pullreview fetches Bitbucket Cloud PR diffs, sends them to an LLM for review, and posts AI-generated comments back to Bitbucket.",
		RunE:  runPullReview,
	}

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", defaultConfig, "Path to config file")
	rootCmd.Flags().StringVar(&prID, "pr", "", "Bitbucket Pull Request ID (overrides branch inference)")
	rootCmd.Flags().StringVar(&bbEmail, "email", "", "Bitbucket account email (overrides config/env)")
	rootCmd.Flags().StringVar(&bbAPIToken, "token", "", "Bitbucket API token (overrides config/env)")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show version and exit")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&postToBB, "post", false, "Post comments to Bitbucket (default: false, just print comments)")

	// Add fix-pr subcommand
	fixPRCmd := &cobra.Command{
		Use:   "fix-pr [flags]",
		Short: "Auto-fix issues in a pull request and create stacked PR",
		Long: `Fetches a Bitbucket Cloud PR, generates fixes using LLM, applies them,
verifies build/test/lint, and creates a stacked pull request with the fixes.`,
		RunE: runFixPR,
	}

	// Share some flags with root command
	fixPRCmd.Flags().StringVarP(&cfgFile, "config", "c", defaultConfig, "Path to config file")
	fixPRCmd.Flags().StringVar(&prID, "pr", "", "Bitbucket Pull Request ID (overrides branch inference)")
	fixPRCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Auto-fix specific flags
	fixPRCmd.Flags().Bool("dry-run", false, "Apply fixes locally without committing or creating PR")
	fixPRCmd.Flags().Bool("skip-verification", false, "Skip build/test/lint verification (dangerous)")
	fixPRCmd.Flags().Int("max-iterations", 0, "Maximum fix iterations (0 = use config default)")
	fixPRCmd.Flags().String("branch-prefix", "", "Branch name prefix (default: from config)")
	fixPRCmd.Flags().Bool("no-pr", false, "Don't create stacked PR (just fix locally)")
	fixPRCmd.Flags().Bool("regenerate", false, "Generate new review instead of using existing comments")

	rootCmd.AddCommand(fixPRCmd)

	cobra.OnInitialize(initConfig)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	// Placeholder: could load config here if needed before command runs
}

func runPullReview(cmd *cobra.Command, args []string) error {
	// Handle version flag
	if showVersion {
		fmt.Printf("pullreview version %s\n", version)
		return nil
	}

	// Load configuration
	cfg, err := config.LoadConfigWithOverrides(cfgFile, bbEmail, bbAPIToken)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get repo path
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}

	// Initialize Bitbucket client
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

	// Initialize LLM client
	llm.SetVerbose(verbose)
	llmClient := llm.NewClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Endpoint)
	llmClient.Model = cfg.LLM.Model

	// Determine PR ID
	finalPRID, err := determinePRID(prID, repoPath, bbClient)
	if err != nil {
		return err
	}

	fmt.Printf("🔍 Reviewing PR #%s...\n", finalPRID)

	// Fetch PR details
	ctx := context.Background()
	pr, err := bbClient.GetPullRequest(ctx, finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR: %w", err)
	}

	if verbose {
		fmt.Printf("PR Title: %s\n", pr.Title)
		if pr.Author != "" {
			fmt.Printf("Author: %s\n", pr.Author)
		}
	}

	// Fetch PR diff
	diff, err := bbClient.GetPRDiff(finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	// Generate review comments
	fmt.Println("🤖 Generating review using LLM...")
	reviewComments, err := getReviewComments(cfg, llmClient, finalPRID, diff)
	if err != nil {
		return fmt.Errorf("failed to generate review: %w", err)
	}

	// Display review results
	if len(reviewComments) == 0 {
		fmt.Println("✅ No issues found - looks good!")
		return nil
	}

	fmt.Printf("\n📝 Found %d comment(s):\n", len(reviewComments))
	fmt.Println(strings.Repeat("=", 60))

	for i, comment := range reviewComments {
		fmt.Printf("\n[Comment %d]\n", i+1)
		if comment.IsFileLevel {
			fmt.Printf("File: %s (file-level comment)\n", comment.FilePath)
		} else {
			fmt.Printf("File: %s, Line: %d\n", comment.FilePath, comment.Line)
		}
		fmt.Printf("Comment:\n%s\n", comment.Text)
		fmt.Println(strings.Repeat("-", 60))
	}

	// Ask user if they want to post to Bitbucket
	if postToBB {
		// --post flag was used, automatically post
		fmt.Println("\n📤 Posting comments to Bitbucket (--post flag enabled)...")
		return postCommentsToBitbucket(bbClient, finalPRID, reviewComments)
	}

	// Interactive mode - ask user
	fmt.Print("\n❓ Do you want to post these comments to Bitbucket? (yes/no): ")
	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" || response == "y" {
		fmt.Println("\n📤 Posting comments to Bitbucket...")
		return postCommentsToBitbucket(bbClient, finalPRID, reviewComments)
	}

	fmt.Println("\n✅ Comments not posted. Review complete.")
	return nil
}

// runFixPR implements the fix-pr subcommand.
func runFixPR(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfigWithOverrides(cfgFile, bbEmail, bbAPIToken)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert config to AutoFixConfig
	autoFixCfg := &autofix.AutoFixConfig{
		Enabled:               cfg.AutoFix.Enabled,
		AutoCreatePR:          cfg.AutoFix.AutoCreatePR,
		MaxIterations:         cfg.AutoFix.MaxIterations,
		VerifyBuild:           cfg.AutoFix.VerifyBuild,
		VerifyTests:           cfg.AutoFix.VerifyTests,
		VerifyLint:            cfg.AutoFix.VerifyLint,
		PipelineMode:          cfg.AutoFix.PipelineMode,
		BranchPrefix:          cfg.AutoFix.BranchPrefix,
		FixPromptFile:         cfg.AutoFix.FixPromptFile,
		CommitMessageTemplate: cfg.AutoFix.CommitMessageTemplate,
		PRTitleTemplate:       cfg.AutoFix.PRTitleTemplate,
		PRDescriptionTemplate: cfg.AutoFix.PRDescriptionTemplate,
	}

	// Apply CLI flag overrides
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

	// Initialize Bitbucket client
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

	// Initialize LLM client
	llm.SetVerbose(verbose)
	llmClient := llm.NewClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Endpoint)
	llmClient.Model = cfg.LLM.Model

	// Determine PR ID
	finalPRID, err := determinePRID(prID, repoPath, bbClient)
	if err != nil {
		return err
	}

	fmt.Printf("🔧 Auto-fixing PR #%s...\n", finalPRID)

	// Fetch PR details
	ctx := context.Background()
	originalPR, err := bbClient.GetPullRequest(ctx, finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR: %w", err)
	}

	// Fetch PR diff
	diff, err := bbClient.GetPRDiff(finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	var reviewComments []review.Comment

	// Check if we should regenerate review or use existing comments
	regenerate, _ := cmd.Flags().GetBool("regenerate")

	if regenerate {
		// Generate new review comments
		fmt.Println("🤖 Generating new review comments...")
		reviewComments, err = getReviewComments(cfg, llmClient, finalPRID, diff)
		if err != nil {
			return fmt.Errorf("failed to generate review: %w", err)
		}
	} else {
		// Fetch existing review comments from Bitbucket
		fmt.Println("📥 Fetching existing review comments from Bitbucket...")
		bbComments, err := bbClient.GetPRComments(finalPRID)
		if err != nil {
			return fmt.Errorf("failed to fetch PR comments: %w", err)
		}

		// Convert Bitbucket comments to review.Comment format
		reviewComments = convertBitbucketCommentsToReviewComments(bbComments)
	}

	if len(reviewComments) == 0 {
		if regenerate {
			fmt.Println("✅ No issues found - nothing to fix!")
		} else {
			fmt.Println("✅ No existing review comments found - nothing to fix!")
			fmt.Println("💡 Tip: Use --regenerate to generate a new review instead")
		}
		return nil
	}

	fmt.Printf("📝 Found %d comment(s) to fix\n", len(reviewComments))

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

// determinePRID resolves the PR ID from CLI arg or git branch.
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

// buildCommitMessage generates commit message from template.
func buildCommitMessage(template string, fixResult *autofix.FixResult) string {
	msg := template
	msg = strings.ReplaceAll(msg, "{issue_summary}", fmt.Sprintf("Applied %d fix(es)", fixResult.FixesApplied))
	msg = strings.ReplaceAll(msg, "{iteration_count}", fmt.Sprintf("%d", fixResult.Iterations))
	msg = strings.ReplaceAll(msg, "{test_status}", fixResult.TestStatus)
	msg = strings.ReplaceAll(msg, "{lint_status}", fixResult.LintStatus)
	return msg
}

// printFixSummary outputs a summary of the fix operation.
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

// getFileContents reads file contents for review context.
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

// getReviewComments generates review comments using LLM.
func getReviewComments(cfg *config.Config, llmClient *llm.Client, prID, diff string) ([]review.Comment, error) {
	// Resolve prompt file path relative to config file location if not absolute
	promptPath := cfg.PromptFile
	if !filepath.IsAbs(promptPath) && cfgFile != "" {
		cfgDir := filepath.Dir(cfgFile)
		promptPath = filepath.Join(cfgDir, promptPath)
	}

	// Read prompt template
	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	prompt := strings.ReplaceAll(string(promptData), "(DIFF_CONTENT_HERE)", diff)

	// Send to LLM
	response, err := llmClient.SendReviewPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse review comments
	r := review.NewReview(prID, diff)
	if err := r.ParseDiff(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse diff for comment mapping: %v\n", err)
	}
	r.ParseLLMResponse(response)

	return r.Comments, nil
}

// convertBitbucketCommentsToReviewComments converts Bitbucket API comments to review.Comment format.
func convertBitbucketCommentsToReviewComments(bbComments []bitbucket.BitbucketComment) []review.Comment {
	var comments []review.Comment

	for _, bbComment := range bbComments {
		// Extract the raw text content
		var text string
		if content, ok := bbComment.Content["raw"].(string); ok {
			text = content
		} else {
			continue // Skip if no text content
		}

		comment := review.Comment{
			Text: text,
		}

		// Check if it's an inline comment
		if bbComment.Inline != nil && bbComment.Inline.Path != "" {
			comment.FilePath = bbComment.Inline.Path
			comment.Line = bbComment.Inline.To
		} else {
			// Top-level comment - skip for auto-fix (only fix inline comments)
			continue
		}

		comments = append(comments, comment)
	}

	return comments
}

// postCommentsToBitbucket posts review comments to Bitbucket PR.
func postCommentsToBitbucket(bbClient *bitbucket.Client, prID string, comments []review.Comment) error {
	successCount := 0
	errorCount := 0
	var errors []string

	for _, comment := range comments {
		if comment.IsFileLevel {
			// Post as file-level comment (not currently supported by simple inline API)
			// For now, we'll post inline comments only
			if verbose {
				fmt.Printf("⚠️  Skipping file-level comment for %s (inline comments only)\n", comment.FilePath)
			}
			continue
		}

		// Post inline comment
		err := bbClient.PostInlineComment(prID, comment.FilePath, comment.Line, comment.Text)
		if err != nil {
			errorCount++
			errMsg := fmt.Sprintf("Failed to post comment to %s:%d: %v", comment.FilePath, comment.Line, err)
			errors = append(errors, errMsg)
			if verbose {
				fmt.Fprintf(os.Stderr, "❌ %s\n", errMsg)
			}
		} else {
			successCount++
			if verbose {
				fmt.Printf("✅ Posted comment to %s:%d\n", comment.FilePath, comment.Line)
			}
		}
	}

	fmt.Printf("\n📊 Results: %d posted, %d failed\n", successCount, errorCount)

	if errorCount > 0 {
		fmt.Fprintln(os.Stderr, "\n❌ Errors encountered:")
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
		}
		return fmt.Errorf("failed to post %d comment(s)", errorCount)
	}

	fmt.Println("✅ All comments posted successfully!")
	return nil
}
