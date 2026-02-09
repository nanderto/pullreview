package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"pullreview/internal/bitbucket"

	"pullreview/internal/config"

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
	repoSlug    string
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
	rootCmd.Flags().StringVar(&repoSlug, "repo", "", "Bitbucket repository slug (overrides config/env)")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show version and exit")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&postToBB, "post", false, "Post comments to Bitbucket (default: false, just print comments)")

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

	cfg, err := config.LoadConfigWithOverrides(cfgFile, bbEmail, bbAPIToken, repoSlug)

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
