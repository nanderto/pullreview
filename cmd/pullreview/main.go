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

	"io/ioutil"
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

	if showVersion {

		fmt.Printf("pullreview version %s\n", version)

		return nil

	}

	// Load configuration with overrides from CLI flags

	cfg, err := config.LoadConfigWithOverrides(cfgFile, bbEmail, bbAPIToken)

	if err != nil {

		return fmt.Errorf("failed to load config: %w", err)

	}

	// Initialize Bitbucket client and attempt authentication

	bbClient := bitbucket.NewClient(
		cfg.Bitbucket.Email,
		cfg.Bitbucket.APIToken,
		cfg.Bitbucket.Workspace,
		cfg.Bitbucket.RepoSlug,
		cfg.Bitbucket.BaseURL,
	)

	if err := bbClient.Authenticate(); err != nil {

		fmt.Fprintf(os.Stderr, "❌ Bitbucket login failed: %v\n", err)

		if cfg.Bitbucket.APIToken == "" {

			fmt.Fprintln(os.Stderr, "  - Missing Bitbucket API token (set in config, env, or CLI flag)")

		}

		if cfg.Bitbucket.Workspace == "" {

			fmt.Fprintln(os.Stderr, "  - Missing Bitbucket workspace (set in config, env, or CLI flag)")

		}

		return fmt.Errorf("could not authenticate with Bitbucket")

	}

	fmt.Printf("✅ Successfully authenticated with Bitbucket (workspace: %s)\n", cfg.Bitbucket.Workspace)

	// Determine PR ID: use CLI flag if provided, else infer from git branch
	finalPRID := prID
	if finalPRID == "" {
		// Try to infer from git branch
		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		branch, err := utils.GetCurrentGitBranch(repoPath)
		if err != nil {
			return fmt.Errorf("could not infer git branch: %w", err)
		}
		fmt.Printf("🔎 Inferred branch: %s\n", branch)
		finalPRID, err = bbClient.GetPRIDByBranch(branch)
		if err != nil {
			return fmt.Errorf("could not find open PR for branch %q: %w", branch, err)

		}
		fmt.Printf("🔎 Inferred PR ID: %s\n", finalPRID)
	} else {
		fmt.Printf("ℹ️ Using provided PR ID: %s\n", finalPRID)
	}

	// Fetch PR metadata
	prMetaBytes, err := bbClient.GetPRMetadata(finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR metadata: %w", err)
	}
	fmt.Printf("✅ Fetched PR metadata for PR #%s\n", finalPRID)

	// Parse and print PR title and description
	type prMetaStruct struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	var prMeta prMetaStruct
	if err := json.Unmarshal(prMetaBytes, &prMeta); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse PR metadata JSON: %v\n", err)
	} else {
		fmt.Printf("🔖 PR Title: %s\n", prMeta.Title)
		fmt.Printf("📝 PR Description: %s\n", prMeta.Description)
	}

	// Fetch PR diff
	diff, err := bbClient.GetPRDiff(finalPRID)
	if err != nil {
		return fmt.Errorf("failed to fetch PR diff: %w", err)
	}
	fmt.Printf("✅ Fetched PR diff for PR #%s (length: %d bytes)\n", finalPRID, len(diff))

	if verbose {
		fmt.Println("------ BEGIN PR DIFF ------")
		fmt.Println(diff)
		fmt.Println("------- END PR DIFF -------")
	}

	// Initialize LLM client
	llm.SetVerbose(verbose)
	llmClient := llm.NewClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Endpoint)
	llmClient.Model = cfg.LLM.Model

	// Resolve prompt file path relative to config file location if not absolute
	promptPath := cfg.PromptFile
	if !filepath.IsAbs(promptPath) && cfgFile != "" {
		cfgDir := filepath.Dir(cfgFile)
		promptPath = filepath.Join(cfgDir, promptPath)
	}

	// Load prompt template
	promptBytes, err := ioutil.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt file %q: %w", promptPath, err)
	}
	promptTemplate := string(promptBytes)

	// Inject diff into prompt
	finalPrompt := strings.Replace(promptTemplate, "(DIFF_CONTENT_HERE)", diff, 1)

	// Send prompt to LLM
	fmt.Println("🤖 Sending review prompt to LLM...")
	llmResp, err := llmClient.SendReviewPrompt(finalPrompt)
	if err != nil {
		return fmt.Errorf("failed to get response from LLM: %w", err)
	}

	// Parse LLM response and print summary and inline comments
	r := review.NewReview(finalPRID, diff)
	if err := r.ParseDiff(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse diff for comment mapping: %v\n", err)
	}
	r.ParseLLMResponse(llmResp)

	// Filter comments: only keep those that match the diff, and report unmatched
	matched, unmatched := review.MatchCommentsToDiff(r.Comments, r.Files)

	// Compose summary with unmatched comments as bullet points (no heading)
	summaryWithUnmatched := r.Summary
	if len(unmatched) > 0 {
		var b strings.Builder
		if summaryWithUnmatched != "" {
			b.WriteString(summaryWithUnmatched)
			b.WriteString("\n\n")
		}
		for _, cmt := range unmatched {
			if cmt.IsFileLevel {
				b.WriteString(fmt.Sprintf("- [%s] %s\n", cmt.FilePath, cmt.Text))
			} else {
				b.WriteString(fmt.Sprintf("- [%s:%d] %s\n", cmt.FilePath, cmt.Line, cmt.Text))
			}
		}
		summaryWithUnmatched = b.String()
	}

	fmt.Println("------ AI Review Summary ------")
	if summaryWithUnmatched != "" {
		fmt.Println(summaryWithUnmatched)
	} else {
		fmt.Println("(No summary comment found in LLM output.)")
	}
	fmt.Println("------ Inline Comments ------")
	if len(matched) == 0 {
		fmt.Println("(No valid inline or file-level comments found in LLM output.)")
	} else {
		for _, cmt := range matched {
			if cmt.IsFileLevel {
				fmt.Printf("[File: %s]\n%s\n\n", cmt.FilePath, cmt.Text)
			} else {
				fmt.Printf("[%s:%d]\n%s\n\n", cmt.FilePath, cmt.Line, cmt.Text)
			}
		}
	}

	if !postToBB {
		fmt.Println("ℹ️  Not posting comments to Bitbucket (use --post to enable).")
		return nil
	}

	// Bitbucket posting output section
	fmt.Fprintf(os.Stderr, "==============================================================================================================================\n")
	fmt.Fprintf(os.Stderr, "[bitbucket] Posting comments to Bitbucket:\n")
	fmt.Fprintf(os.Stderr, "==============================================================================================================================\n\n")

	// Post inline and file-level comments (only matched)
	inlineCount := 0
	for _, cmt := range matched {
		if cmt.IsFileLevel {
			err := bbClient.PostSummaryComment(finalPRID, cmt.Text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to post file-level comment to %s: %v\n", cmt.FilePath, err)
				if verbose {
					fmt.Fprintf(os.Stderr, "    [bitbucket] Error details: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "✅ Posted file-level comment to %s\n", cmt.FilePath)
			}
		} else {
			err := bbClient.PostInlineComment(finalPRID, cmt.FilePath, cmt.Line, cmt.Text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to post inline comment to %s:%d: %v\n", cmt.FilePath, cmt.Line, err)
				if verbose {
					fmt.Fprintf(os.Stderr, "    [bitbucket] Error details: %v\n", err)
				}
			} else {
				inlineCount++
				fmt.Fprintf(os.Stderr, "✅ Posted inline comment to %s:%d\n", cmt.FilePath, cmt.Line)
			}
		}
	}

	// Post summary comment (with unmatched comments as bullet points)
	summaryPosted := false
	if summaryWithUnmatched != "" {
		err := bbClient.PostSummaryComment(finalPRID, summaryWithUnmatched)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to post summary comment: %v\n", err)
			if verbose {
				fmt.Fprintf(os.Stderr, "    [bitbucket] Error details: %v\n", err)
			}
		} else {
			summaryPosted = true
			fmt.Fprintln(os.Stderr, "✅ Posted summary comment to PR")
		}
	}

	fmt.Fprintf(os.Stderr, "\n==============================================================================================================================\n")
	fmt.Fprintf(os.Stderr, "==============================================================================================================================\n")

	fmt.Printf("✅ Posted %d inline comment(s)%s to PR #%s\n", inlineCount,
		func() string {
			if summaryPosted {
				return " and summary"
			}
			return ""
		}(), finalPRID)

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
