# Auto-Fix Implementation Plan

## Status

| Phase | Description | Status | Completed |
|-------|-------------|--------|-----------|
| 1 | Fix Generation & Parsing | ✅ Complete | 2026-02-12 |
| 2 | Fix Application & Verification | ✅ Complete | 2026-02-12 |
| 3 | Git Operations | ✅ Complete | 2026-02-12 |
| 4 | Stacked PR Creation | ✅ Complete | 2026-02-12 |
| 5 | CLI & Configuration | ⏳ Not Started | - |
| 6 | Testing & Hardening | ⏳ Not Started | - |

---

## Overview

This document outlines the detailed implementation plan for adding auto-fix capabilities with stacked pull request creation to the `pullreview` CLI tool.

---

## 1. Architecture Overview

### Current Flow
```
User → CLI → Bitbucket Client → Fetch PR Diff → LLM Client → Review → Post Comments
```

### New Flow with Auto-Fix
```
User → CLI → Bitbucket Client → Fetch PR Diff
                    ↓
            LLM Client → Review
                    ↓
            [Optional] Auto-Fix Mode
                    ↓
            LLM Client → Generate Fixes
                    ↓
            Git Operations → Clone/Checkout → Apply Fixes
                    ↓
            Verification Loop (build/test/lint)
                    ↓ (loop until pass or max iterations)
            Git Operations → Commit → Push
                    ↓
            Bitbucket Client → Create Stacked PR
```

### Integration Points

The auto-fix feature integrates with existing code at three key points:

1. **After Review Parsing**: Access review comments to generate fixes
2. **LLM Client Extension**: New method for fix generation
3. **Bitbucket Client Extension**: New method for PR creation

---

## 2. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           REVIEW PHASE                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  1. Fetch PR Metadata (existing)                                        │
│  2. Fetch PR Diff (existing)                                            │
│  3. Send to LLM for Review (existing)                                   │
│  4. Parse Review Comments (existing)                                    │
│  5. Post Review Comments (existing)                                     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          AUTO-FIX PHASE (NEW)                           │
├─────────────────────────────────────────────────────────────────────────┤
│  6. Clone/Checkout PR Branch via Git                                    │
│  7. Send Review + Diff to LLM for Fix Generation                        │
│  8. Parse Fix Response (JSON format)                                    │
│  9. Apply Fixes to Files                                                │
│ 10. Run Verification (vet, gofmt, build, test)                          │
│     └─► If fail: Send errors back to LLM → Goto step 8 (max N times)    │
│ 11. Create Fix Branch                                                   │
│ 12. Commit Fixes                                                        │
│ 13. Push Branch to Bitbucket                                            │
│ 14. Create Stacked PR via Bitbucket API                                 │
│ 15. Output Summary                                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. LLM Prompt Modifications

### 3.1 Recommended Fix Format: Structured JSON

After evaluating options, **JSON format (Option A)** is recommended because:
- Reliable parsing with Go's `encoding/json`
- Supports multiple fixes per file
- Clear structure for line ranges and replacements
- Easy to validate before applying

### 3.2 New Fix Generation Prompt

Create a new prompt file `prompt_fix.md`:

```markdown
# AUTO-FIX CODE GENERATION PROMPT

You are a **code fix generator** for a Go codebase.

Based on the code review issues identified below, generate precise code fixes.

## RULES

1. Generate **minimal, surgical fixes** - change only what's necessary
2. Preserve original code style and formatting
3. Do NOT introduce new features or refactorings
4. Each fix must directly address a review issue
5. Fixes must be syntactically correct Go code

## OUTPUT FORMAT (MANDATORY)

Respond with a JSON object containing an array of fixes:

```json
{
  "fixes": [
    {
      "file": "relative/path/to/file.go",
      "line_start": 45,
      "line_end": 47,
      "original_code": "exact code being replaced",
      "fixed_code": "the corrected code",
      "issue_addressed": "Brief description of the issue being fixed"
    }
  ],
  "summary": "Brief summary of all fixes applied"
}
```

## REVIEW ISSUES TO FIX

{REVIEW_ISSUES}

## ORIGINAL DIFF

```
{DIFF_CONTENT}
```

## CURRENT FILE CONTENTS

{FILE_CONTENTS}
```

### 3.3 Error Correction Prompt

For iterative fixing when build/test fails:

```markdown
# FIX CORRECTION PROMPT

Your previous fix caused the following error:

## ERROR OUTPUT

```
{ERROR_OUTPUT}
```

## YOUR PREVIOUS FIX

```json
{PREVIOUS_FIX}
```

## ORIGINAL FILE CONTENT

```
{FILE_CONTENT}
```

Please provide a corrected fix in the same JSON format that resolves this error.
```

---

## 4. New Packages/Files

### 4.1 Package: `internal/autofix`

```
internal/autofix/
├── autofix.go         # Main orchestration logic
├── autofix_test.go    # Unit tests
├── types.go           # Data structures (Fix, FixResult, etc.)
├── parser.go          # JSON fix parsing
├── parser_test.go     # Parser tests
├── applier.go         # File modification logic
├── applier_test.go    # Applier tests
└── testdata/          # Test fixtures
    ├── sample_fix.json
    └── sample_file.go
```

### 4.2 Package: `internal/git`

```
internal/git/
├── git.go             # Git operations wrapper
├── git_test.go        # Unit tests
└── testdata/          # Test fixtures
```

### 4.3 Package: `internal/verify`

```
internal/verify/
├── verify.go          # Build verification orchestration
├── verify_test.go     # Unit tests
└── types.go           # VerificationResult, VerificationConfig
```

### 4.4 New Prompt Files

```
prompts/
├── fix_generation.md     # Prompt for generating fixes
└── fix_correction.md     # Prompt for correcting failed fixes
```

---

## 5. Modified Packages/Files

### 5.1 `internal/config/config.go`

**Changes:**
- Add `AutoFix` struct to `Config`
- Add environment variable support for auto-fix settings
- Add pipeline mode detection

### 5.2 `internal/bitbucket/client.go`

**Changes:**
- Add `CreatePullRequest()` method
- Add `GetFileContent()` method (for fetching current file contents)
- Add `BranchExists()` method

### 5.3 `cmd/pullreview/main.go`

**Changes:**
- Add `--auto-fix` flag
- Add `fix-pr` subcommand
- Integrate auto-fix workflow after review

### 5.4 `internal/llm/client.go`

**Changes:**
- Add `SendFixPrompt()` method
- Add `SendFixCorrectionPrompt()` method

---

## 6. API Design

### 6.1 Types (`internal/autofix/types.go`)

```go
package autofix

import "time"

// Fix represents a single code fix to be applied.
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

// FixResult represents the outcome of applying fixes.
type FixResult struct {
    Success        bool
    FilesChanged   []string
    FixesApplied   int
    FixesFailed    int
    Iterations     int
    BuildStatus    string
    TestStatus     string
    LintStatus     string
    ErrorMessages  []string
    PRCreated      bool
    PRURL          string
    PRNumber       int
    BranchName     string
}

// AutoFixConfig holds configuration for auto-fix operations.
type AutoFixConfig struct {
    Enabled              bool   `yaml:"enabled"`
    AutoCreatePR         bool   `yaml:"auto_create_pr"`
    MaxIterations        int    `yaml:"max_iterations"`
    VerifyBuild          bool   `yaml:"verify_build"`
    VerifyTests          bool   `yaml:"verify_tests"`
    VerifyLint           bool   `yaml:"verify_lint"`
    PipelineMode         bool   `yaml:"pipeline_mode"`
    BranchPrefix         string `yaml:"branch_prefix"`
    CommitMessageTemplate string `yaml:"commit_message_template"`
    PRTitleTemplate      string `yaml:"pr_title_template"`
    PRDescriptionTemplate string `yaml:"pr_description_template"`
}

// PRContext holds information about the original PR.
type PRContext struct {
    ID          string
    Title       string
    Description string
    SourceBranch string
    DestBranch   string
    Author      string
    URL         string
}
```

### 6.2 AutoFix Orchestrator (`internal/autofix/autofix.go`)

```go
package autofix

import (
    "context"
    "fmt"
    "pullreview/internal/bitbucket"
    "pullreview/internal/git"
    "pullreview/internal/llm"
    "pullreview/internal/review"
    "pullreview/internal/verify"
)

// AutoFixer orchestrates the auto-fix workflow.
type AutoFixer struct {
    config     *AutoFixConfig
    bbClient   *bitbucket.Client
    llmClient  *llm.Client
    gitOps     *git.Operations
    verifier   *verify.Verifier
    prContext  *PRContext
    repoPath   string
    verbose    bool
}

// NewAutoFixer creates a new AutoFixer instance.
func NewAutoFixer(
    cfg *AutoFixConfig,
    bbClient *bitbucket.Client,
    llmClient *llm.Client,
    repoPath string,
) *AutoFixer

// Run executes the complete auto-fix workflow.
// Returns FixResult with details about the operation.
func (af *AutoFixer) Run(ctx context.Context, prID string, reviewComments []review.Comment, diff string) (*FixResult, error)

// GenerateFixes sends the review issues to LLM and parses the fix response.
func (af *AutoFixer) GenerateFixes(ctx context.Context, comments []review.Comment, diff string) (*FixResponse, error)

// ApplyFixes applies the parsed fixes to local files.
func (af *AutoFixer) ApplyFixes(fixes []Fix) error

// VerifyAndIterate runs verification and iterates on failures.
func (af *AutoFixer) VerifyAndIterate(ctx context.Context, fixes []Fix) (*FixResult, error)

// CreateFixBranch creates a new branch for the fixes.
func (af *AutoFixer) CreateFixBranch() (string, error)

// CommitAndPush commits changes and pushes to remote.
func (af *AutoFixer) CommitAndPush(result *FixResult) error

// CreateStackedPR creates a PR targeting the original PR's branch.
func (af *AutoFixer) CreateStackedPR(result *FixResult) error

// IsPipelineMode returns true if running in a CI/CD environment.
func IsPipelineMode() bool

// SetVerbose enables or disables verbose output.
func (af *AutoFixer) SetVerbose(v bool)
```

### 6.3 Git Operations (`internal/git/git.go`)

```go
package git

import (
    "context"
)

// Operations provides git operations for the auto-fix workflow.
type Operations struct {
    repoPath   string
    remoteName string
    verbose    bool
}

// NewOperations creates a new git operations handler.
func NewOperations(repoPath string) *Operations

// Checkout checks out the specified branch.
func (g *Operations) Checkout(branch string) error

// CreateBranch creates a new branch from the current HEAD.
func (g *Operations) CreateBranch(name string) error

// BranchExists checks if a branch exists locally or remotely.
func (g *Operations) BranchExists(name string) (bool, error)

// GenerateBranchName creates a unique branch name for fixes.
func (g *Operations) GenerateBranchName(baseBranch string, prefix string) string

// StageFiles stages the specified files for commit.
func (g *Operations) StageFiles(files []string) error

// Commit creates a commit with the given message.
func (g *Operations) Commit(message string) error

// Push pushes the current branch to the remote.
func (g *Operations) Push(branch string) error

// GetRemoteURL returns the remote URL for the origin.
func (g *Operations) GetRemoteURL() (string, error)

// HasUncommittedChanges checks for uncommitted changes.
func (g *Operations) HasUncommittedChanges() (bool, error)

// ResetHard resets the working directory to HEAD.
func (g *Operations) ResetHard() error

// GetCurrentBranch returns the current branch name.
func (g *Operations) GetCurrentBranch() (string, error)

// Fetch fetches updates from the remote.
func (g *Operations) Fetch() error
```

### 6.4 Verification (`internal/verify/verify.go`)

```go
package verify

import (
    "context"
)

// VerificationConfig holds settings for verification steps.
type VerificationConfig struct {
    RunVet     bool
    RunFmt     bool
    RunBuild   bool
    RunTests   bool
    RepoPath   string
    Verbose    bool
}

// VerificationResult holds the outcome of verification.
type VerificationResult struct {
    VetPassed    bool
    VetOutput    string
    FmtPassed    bool
    FmtOutput    string
    BuildPassed  bool
    BuildOutput  string
    TestsPassed  bool
    TestsOutput  string
    AllPassed    bool
    CombinedErrors string
}

// Verifier runs build verification steps.
type Verifier struct {
    config *VerificationConfig
}

// NewVerifier creates a new Verifier.
func NewVerifier(cfg *VerificationConfig) *Verifier

// RunAll executes all configured verification steps.
func (v *Verifier) RunAll(ctx context.Context) (*VerificationResult, error)

// RunVet runs `go vet ./...`.
func (v *Verifier) RunVet(ctx context.Context) (bool, string, error)

// RunFmt runs `gofmt -s -l .` and reports unformatted files.
func (v *Verifier) RunFmt(ctx context.Context) (bool, string, error)

// RunBuild runs `go build ./...`.
func (v *Verifier) RunBuild(ctx context.Context) (bool, string, error)

// RunTests runs `go test ./...`.
func (v *Verifier) RunTests(ctx context.Context) (bool, string, error)
```

### 6.5 Bitbucket PR Creation (`internal/bitbucket/client.go` - new methods)

```go
// CreatePullRequestInput holds data for creating a new PR.
type CreatePullRequestInput struct {
    Title         string
    Description   string
    SourceBranch  string
    DestBranch    string
    CloseSourceBranch bool
    Reviewers     []string
}

// CreatePullRequestOutput holds the response from PR creation.
type CreatePullRequestOutput struct {
    ID    int
    URL   string
    Title string
}

// CreatePullRequest creates a new pull request via the Bitbucket API.
func (c *Client) CreatePullRequest(input *CreatePullRequestInput) (*CreatePullRequestOutput, error)

// GetFileContent fetches the raw content of a file from a specific branch/commit.
func (c *Client) GetFileContent(branch, filePath string) (string, error)

// GetPRDetails fetches detailed PR information including source/dest branches.
func (c *Client) GetPRDetails(prID string) (*PRDetails, error)

// PRDetails holds detailed information about a PR.
type PRDetails struct {
    ID           int    `json:"id"`
    Title        string `json:"title"`
    Description  string `json:"description"`
    State        string `json:"state"`
    SourceBranch string
    DestBranch   string
    Author       string
    URL          string
}
```

---

## 7. Configuration Schema

### 7.1 Updated `pullreview.yaml`

```yaml
bitbucket:
  email: user@example.com
  api_token: ${BITBUCKET_API_TOKEN}
  workspace: myworkspace
  base_url: https://api.bitbucket.org/2.0

llm:
  provider: copilot
  model: gpt-4.1

prompt_file: prompt.md

# New auto-fix configuration
auto_fix:
  # Enable auto-fix mode (default: false)
  enabled: false
  
  # Automatically create stacked PR after fixes (default: true)
  auto_create_pr: true
  
  # Maximum fix/verify iterations before giving up (default: 5)
  max_iterations: 5
  
  # Verification steps to run
  verify_build: true   # Run `go build ./...`
  verify_tests: true   # Run `go test ./...`
  verify_lint: true    # Run `go vet ./...` and `gofmt`
  
  # Force pipeline mode (auto-detected if not set)
  # When true: no prompts, fully automated
  pipeline_mode: false
  
  # Branch prefix for fix branches (default: pullreview-fixes)
  branch_prefix: "pullreview-fixes"
  
  # Prompt file for fix generation
  fix_prompt_file: "prompts/fix_generation.md"
  
  # Commit message template
  # Available variables: {issue_summary}, {iteration_count}, {test_status}, {lint_status}
  commit_message_template: |
    🤖 Auto-fix: {issue_summary}
    
    Iterations: {iteration_count}
    Tests: {test_status}
    Lint: {lint_status}
  
  # PR title template
  # Available variables: {pr_id}, {original_title}
  pr_title_template: "🤖 Auto-fixes for PR #{pr_id}: {original_title}"
  
  # PR description template (supports multiline)
  # Available variables: {original_pr_id}, {original_pr_link}, {issue_count},
  #                      {iteration_count}, {file_list}, {build_status},
  #                      {test_status}, {lint_status}, {ai_explanation}
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

### 7.2 Environment Variable Overrides

```bash
# Auto-fix environment variables
PULLREVIEW_AUTOFIX_ENABLED=true
PULLREVIEW_AUTOFIX_MAX_ITERATIONS=5
PULLREVIEW_AUTOFIX_PIPELINE_MODE=true
PULLREVIEW_AUTOFIX_VERIFY_BUILD=true
PULLREVIEW_AUTOFIX_VERIFY_TESTS=true
PULLREVIEW_AUTOFIX_VERIFY_LINT=true
```

### 7.3 Configuration Loading Updates

```go
// In config.go - extend Config struct
type Config struct {
    Bitbucket  BitbucketConfig  `yaml:"bitbucket"`
    LLM        LLMConfig        `yaml:"llm"`
    PromptFile string           `yaml:"prompt_file"`
    AutoFix    AutoFixConfig    `yaml:"auto_fix"` // NEW
}

// AutoFixConfig holds configuration for auto-fix mode.
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

// SetAutoFixDefaults sets sensible defaults for auto-fix config.
func (c *AutoFixConfig) SetDefaults() {
    if c.MaxIterations == 0 {
        c.MaxIterations = 5
    }
    if c.BranchPrefix == "" {
        c.BranchPrefix = "pullreview-fixes"
    }
    // ... other defaults
}
```

---

## 8. CLI UX Flow

### 8.1 New CLI Flag

```bash
# Typical usage (PR auto-detected from current branch)
pullreview --auto-fix

# With explicit PR ID (for CI or reviewing other branches)
pullreview --pr=123 --auto-fix
```

**Approach: Single `--auto-fix` flag on existing command**

Reasons:
- Leverages existing PR auto-detection from current git branch
- Minimal change to CLI interface
- Natural extension of current workflow
- User is already on the branch they want to review/fix

### 8.2 Command Implementation

```go
// In main.go - add --auto-fix flag to existing command
var autoFix bool

func init() {
    rootCmd.Flags().BoolVar(&autoFix, "auto-fix", false, 
        "After review, generate fixes and create stacked PR")
    rootCmd.Flags().Bool("dry-run", false, 
        "Generate fixes but don't commit/push (preview mode)")
}
```

The existing `--pr` flag works as before - optional, auto-detected if not provided.

### 8.3 Happy Path Flow (Local Development)

```
$ pullreview --auto-fix

✅ Successfully authenticated with Bitbucket (workspace: mycompany)
🔎 Inferred branch: feature/user-auth
🔎 Inferred PR ID: 123
✅ PR Title: Add user authentication feature
✅ Source branch: feature/user-auth
✅ Destination branch: main

📋 Found 3 review issues to fix:
   1. [src/auth.go:45] Missing error context
   2. [src/auth.go:78] Potential nil pointer dereference
   3. [src/users.go:23] SQL injection vulnerability

🤖 Generating fixes via LLM...
✅ Generated 3 fixes

 Applying fixes to local files...
✅ Applied 3 fixes to 2 files

🔍 Running verification (iteration 1/5)...
   ✅ go vet ./... passed
   ✅ gofmt passed
   ✅ go build ./... passed
   ✅ go test ./... passed

🌿 Creating fix branch: pullreview-fixes/feature/user-auth-20260212-0053
✅ Branch created

📤 Committing and pushing...
✅ Committed: "🤖 Auto-fix: 3 issues resolved"
✅ Pushed to origin

🔗 Creating stacked PR...
✅ Created PR #456: "🤖 Auto-fixes for PR #123: Add user authentication feature"
   URL: https://bitbucket.org/mycompany/repo/pull-requests/456

✅ Auto-fix complete!
   - Issues fixed: 3
   - Iterations: 1
   - Stacked PR: #456
```

### 8.4 Happy Path Flow (Pipeline Mode)

```
$ pullreview --pr=123 --auto-fix

[INFO] Pipeline mode detected (CI=true)
[INFO] Authenticating with Bitbucket...
[INFO] Fetching PR #123 details...
[INFO] Found 3 review issues to fix
[INFO] Generating fixes via LLM...
[INFO] Generated 3 fixes
[INFO] Applying fixes to local files...
[INFO] Applied 3 fixes to 2 files
[INFO] Running verification (iteration 1/5)...
[INFO]   go vet: PASS
[INFO]   gofmt: PASS
[INFO]   go build: PASS
[INFO]   go test: PASS
[INFO] Creating fix branch: pullreview-fixes/feature/user-auth-20260212-0053
[INFO] Committing and pushing...
[INFO] Creating stacked PR...
[INFO] Created PR #456
[INFO] URL: https://bitbucket.org/mycompany/repo/pull-requests/456
[INFO] Auto-fix complete: 3 issues fixed in 1 iteration
```

### 8.5 Error Flow (Verification Fails, Iteration Succeeds)

```
$ pullreview --auto-fix

✅ Successfully authenticated with Bitbucket
🔎 Inferred branch: feature/user-auth
🔎 Inferred PR ID: 123
📋 Found 2 review issues to fix

🤖 Generating fixes via LLM...
✅ Generated 2 fixes

📁 Applying fixes to local files...
✅ Applied 2 fixes to 1 file

🔍 Running verification (iteration 1/5)...
   ✅ go vet ./... passed
   ✅ gofmt passed
   ❌ go build ./... FAILED

Build error:
  src/auth.go:47:15: undefined: validateUser

🤖 Sending error to LLM for correction...
✅ Received corrected fix

📁 Applying corrected fixes...
✅ Applied 1 fix

🔍 Running verification (iteration 2/5)...
   ✅ go vet ./... passed
   ✅ gofmt passed
   ✅ go build ./... passed
   ✅ go test ./... passed

🌿 Creating fix branch...
✅ Auto-fix complete in 2 iterations!
```

### 8.6 Error Flow (Max Iterations Exceeded)

```
$ pullreview --auto-fix

✅ Successfully authenticated with Bitbucket
🔎 Inferred branch: feature/user-auth
🔎 Inferred PR ID: 123
📋 Found 1 review issue to fix

🤖 Generating fixes via LLM...
📁 Applying fixes...

🔍 Running verification (iteration 1/5)...
   ❌ go test ./... FAILED

🤖 Sending error to LLM for correction...
🔍 Running verification (iteration 2/5)...
   ❌ go test ./... FAILED

🤖 Sending error to LLM for correction...
🔍 Running verification (iteration 3/5)...
   ❌ go test ./... FAILED

🤖 Sending error to LLM for correction...
🔍 Running verification (iteration 4/5)...
   ❌ go test ./... FAILED

🤖 Sending error to LLM for correction...
🔍 Running verification (iteration 5/5)...
   ❌ go test ./... FAILED

❌ Auto-fix failed: Maximum iterations (5) exceeded without passing verification

Last error:
  --- FAIL: TestUserAuth (0.00s)
      auth_test.go:45: expected token, got empty string

🧹 Cleaning up...
✅ Restored original branch state

Exit code: 1
```

---

## 9. Testing Approach

### 9.1 Unit Tests

#### `internal/autofix/parser_test.go`
```go
func TestParseFixResponse_ValidJSON(t *testing.T)
func TestParseFixResponse_InvalidJSON(t *testing.T)
func TestParseFixResponse_EmptyFixes(t *testing.T)
func TestParseFixResponse_MalformedFix(t *testing.T)
```

#### `internal/autofix/applier_test.go`
```go
func TestApplyFix_SingleLine(t *testing.T)
func TestApplyFix_MultiLine(t *testing.T)
func TestApplyFix_FileNotFound(t *testing.T)
func TestApplyFix_LineOutOfRange(t *testing.T)
func TestApplyFix_OriginalCodeMismatch(t *testing.T)
func TestApplyMultipleFixes_SameFile(t *testing.T)
func TestApplyMultipleFixes_DifferentFiles(t *testing.T)
```

#### `internal/git/git_test.go`
```go
func TestGenerateBranchName(t *testing.T)
func TestBranchExists(t *testing.T)
func TestCheckout(t *testing.T)
func TestCreateBranch(t *testing.T)
func TestCommit(t *testing.T)
func TestPush(t *testing.T) // May need to mock
```

#### `internal/verify/verify_test.go`
```go
func TestRunVet_Pass(t *testing.T)
func TestRunVet_Fail(t *testing.T)
func TestRunFmt_Pass(t *testing.T)
func TestRunFmt_Fail(t *testing.T)
func TestRunBuild_Pass(t *testing.T)
func TestRunBuild_Fail(t *testing.T)
func TestRunTests_Pass(t *testing.T)
func TestRunTests_Fail(t *testing.T)
func TestRunAll(t *testing.T)
```

### 9.2 Integration Tests

#### `internal/bitbucket/client_integration_test.go`
```go
// Use httptest or VCR to mock Bitbucket API
func TestCreatePullRequest_Success(t *testing.T)
func TestCreatePullRequest_Unauthorized(t *testing.T)
func TestCreatePullRequest_BranchNotFound(t *testing.T)
func TestGetFileContent(t *testing.T)
```

### 9.3 End-to-End Tests

```go
// e2e/autofix_test.go
func TestAutoFixWorkflow_HappyPath(t *testing.T)
func TestAutoFixWorkflow_VerificationFails(t *testing.T)
func TestAutoFixWorkflow_MaxIterations(t *testing.T)
func TestAutoFixWorkflow_PipelineMode(t *testing.T)
```

### 9.4 Test Fixtures

```
internal/autofix/testdata/
├── sample_fix_response.json
├── sample_fix_invalid.json
├── sample_file_before.go
├── sample_file_after.go
└── sample_diff.txt

internal/verify/testdata/
├── valid_go_file.go
├── invalid_go_file.go
└── unformatted_go_file.go
```

---

## 10. Phased Rollout

### Phase 1: Fix Generation & Parsing (Week 1-2)

**Scope:**
- Implement `internal/autofix/types.go`
- Implement `internal/autofix/parser.go`
- Create `prompts/fix_generation.md`
- Add LLM method `SendFixPrompt()`
- Unit tests for parsing

**Deliverable:**
- Can generate and parse fix responses from LLM
- CLI flag `--generate-fixes` to show proposed fixes without applying

### Phase 2: Fix Application & Verification (Week 3-4)

**Scope:**
- Implement `internal/autofix/applier.go`
- Implement `internal/verify/verify.go`
- Iterative fix loop with error feedback
- Unit tests for applier and verifier

**Deliverable:**
- Can apply fixes to files
- Can run verification and iterate on failures
- CLI flag `--dry-run` to apply fixes locally without committing

### Phase 3: Git Operations (Week 5)

**Scope:**
- Implement `internal/git/git.go`
- Branch creation, commit, push
- Rollback on failure
- Unit tests for git operations

**Deliverable:**
- Can create fix branch, commit, and push
- Handles branch name collisions
- Graceful rollback on errors

### Phase 4: Stacked PR Creation (Week 6) ✅ COMPLETED (2026-02-12)

**Scope:**
- Implement `CreatePullRequest()` in Bitbucket client
- PR title/description templating
- Integration with auto-fix workflow

**Deliverable:**
- Can create stacked PRs via Bitbucket API
- Full end-to-end workflow works

**Implementation Summary:**
- ✅ Added `CreatePullRequest()`, `GetFileContent()`, `BranchExists()`, `GetPullRequestByBranch()`, and `GetPullRequest()` methods to Bitbucket client
- ✅ Created `internal/bitbucket/templates.go` with PR title/description templating
- ✅ Added markdown escaping to prevent injection in user-provided content
- ✅ Integrated PR creation into `AutoFixer` via `CreateStackedPR()` method
- ✅ Checks for existing PRs to prevent duplicates
- ✅ Verifies branch exists remotely before creating PR
- ✅ Formats status with emojis (✅/❌) for better readability
- ✅ 26 tests in bitbucket package, all passing
- ✅ Code formatted and passes `go vet`

**Files Modified:**
- `internal/bitbucket/client.go` - Extended with PR creation methods
- `internal/autofix/autofix.go` - Added `CreateStackedPR()` and helper methods
- `internal/bitbucket/client_test.go` - Added 10 new tests

**Files Created:**
- `internal/bitbucket/templates.go` - PR template helpers with default templates
- `internal/bitbucket/templates_test.go` - 7 test suites for templates

### Phase 5: CLI & Configuration Polish (Week 7)

**Scope:**
- Add `fix-pr` subcommand
- Add `--auto-fix` flag to main command
- Configuration schema in `pullreview.yaml`
- Pipeline mode detection
- Documentation

**Deliverable:**
- Complete feature ready for release
- Documentation and examples

### Phase 6: Testing & Hardening (Week 8)

**Scope:**
- Integration tests
- E2E tests
- Edge case handling
- Performance optimization
- Bug fixes

**Deliverable:**
- Production-ready feature

---

## 11. Risk Assessment

### 11.1 High Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM generates incorrect/unsafe fixes | Code breaks, security issues | Mandatory verification loop, human review on stacked PR |
| Git operations corrupt repo state | Lost work, broken state | Always work on new branch, implement rollback, never force push |
| API rate limits (Bitbucket/LLM) | Workflow interruption | Implement retry with backoff, cache where possible |
| Max iterations never converge | Infinite loops, hung pipeline | Hard limit (default 5), clear failure messaging |

### 11.2 Medium Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Fix format parsing failures | Fixes not applied | Strict JSON validation, fallback to manual mode |
| Branch name collisions | Push failures | Timestamp in branch name, check before create |
| Merge conflicts | Can't apply fixes | Detect early, abort gracefully, notify user |
| Network failures mid-workflow | Inconsistent state | Checkpoint progress, support resume |

### 11.3 Low Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Config file backwards compatibility | Old configs fail | Sensible defaults, validation with clear errors |
| Cross-platform path issues | Windows/Unix differences | Use `filepath` package consistently |
| Large diffs overwhelm LLM | Incomplete fixes | Chunk large diffs, prioritize critical issues |

### 11.4 Mitigations Summary

1. **Never commit unverified code** - All fixes must pass build/test/lint
2. **Work on isolated branches** - Never modify original PR branch
3. **Stacked PR for review** - Author reviews fixes before merging to their branch
4. **Fail safe** - On any unexpected error, abort and restore state
5. **Clear logging** - Full audit trail for debugging

---

## 12. Effort Estimation

### 12.1 Development Time

| Phase | Duration | Complexity |
|-------|----------|------------|
| Phase 1: Fix Generation & Parsing | 2 weeks | Medium |
| Phase 2: Fix Application & Verification | 2 weeks | High |
| Phase 3: Git Operations | 1 week | Medium |
| Phase 4: Stacked PR Creation | 1 week | Low |
| Phase 5: CLI & Configuration | 1 week | Low |
| Phase 6: Testing & Hardening | 1 week | Medium |
| **Total** | **8 weeks** | - |

### 12.2 Lines of Code Estimate

| Component | New Lines | Modified Lines |
|-----------|-----------|----------------|
| `internal/autofix/` | ~800 | - |
| `internal/git/` | ~400 | - |
| `internal/verify/` | ~300 | - |
| `internal/bitbucket/` | - | ~150 |
| `internal/config/` | - | ~100 |
| `internal/llm/` | - | ~50 |
| `cmd/pullreview/main.go` | - | ~200 |
| Tests | ~1500 | - |
| Prompts & Docs | ~500 | - |
| **Total** | **~3500** | **~500** |

### 12.3 Dependencies

- **go-git** (optional): Could use for more robust git operations, but shelling to `git` is simpler and more reliable
- No new major dependencies required

---

## 13. Answers to Questions

### Q1: LLM interaction - Should fixes be generated during initial review or second call?

**Answer: Second call (separate prompt)**

Reasons:
- Keeps review prompt focused on defect detection
- Fix generation requires different prompt structure (JSON output)
- Allows user to decide whether to generate fixes
- More flexible (can re-generate fixes without re-reviewing)

### Q2: Fix format - Most reliable parsing method?

**Answer: Structured JSON**

```json
{
  "fixes": [{
    "file": "path/to/file.go",
    "line_start": 45,
    "line_end": 47,
    "original_code": "...",
    "fixed_code": "...",
    "issue_addressed": "..."
  }]
}
```

Reasons:
- Unambiguous parsing with `encoding/json`
- Easy validation
- Supports multi-line fixes
- Can include metadata (explanation)

### Q3: Git operations - go-git library or shell out?

**Answer: Shell out to `git` command**

Reasons:
- Simpler implementation
- Git is already a dependency (used in utils)
- More reliable handling of auth (SSH keys, credential helpers)
- Easier debugging (same commands user would run)
- go-git has known issues with some edge cases

### Q4: User confirmation points?

**Answer: None required**

Running `fix-pr` or `--auto-fix` is the user's explicit consent. The stacked PR serves as the final approval mechanism - the author can review, accept, or reject the fixes there.

This simplifies the workflow:
- No interactive prompts needed
- Same behavior in local and pipeline mode
- User can always abandon the stacked PR if they don't want the fixes

### Q5: Rollback strategy if push succeeds but PR creation fails?

**Answer: Leave branch, report error**

- Don't auto-delete the pushed branch
- User can manually create PR or re-run
- Branch name includes timestamp, so retrying creates new branch
- Document the orphaned branch in error message

### Q6: Multi-fix batching?

**Answer: One commit for all fixes**

Reasons:
- Atomic change (all or nothing)
- Cleaner git history
- Single PR review
- Simpler rollback

### Q7: Config schema - Configurable vs hard-coded?

**Configurable:**
- enabled, max_iterations
- verify_build, verify_tests, verify_lint
- branch_prefix
- Templates (commit message, PR title, PR description)

**Hard-coded:**
- Timestamp format for branch names
- Verification commands (`go vet`, `gofmt`, etc.)
- Bitbucket API endpoints

### Q8: Backwards compatibility?

**Answer:**
- All new config is under `auto_fix:` key
- Old configs without this key work fine (auto_fix disabled by default)
- Validation only checks new fields if auto_fix is enabled

### Q9: Iterative fix loop - How to feed errors to LLM?

**Answer:**

```
Your previous fix caused this error:

BUILD ERROR:
  src/auth.go:47:15: undefined: validateUser

PREVIOUS FIX:
  {json of the fix that failed}

CURRENT FILE STATE:
  {full file content}

Please provide a corrected fix.
```

### Q10: Max iterations exceeded - What happens?

**Answer:**
- Exit with code 1
- Log the last error clearly
- Restore working directory to clean state
- Don't leave partial changes

Same behavior in both local and pipeline mode - fail fast and fail clearly.

### Q11: Pipeline vs Local auth?

**Answer: No changes needed**

Existing auth mechanisms work:
- Config file: `pullreview.yaml`
- Environment: `BITBUCKET_API_TOKEN`, etc.
- CLI flags: `--token`

Pipeline auto-detection via:
- `CI=true`
- `TF_BUILD=true`
- `GITHUB_ACTIONS=true`

### Q12: Branch name collisions?

**Answer:** Timestamp suffix makes collisions extremely unlikely

Format: `pullreview-fixes/{original-branch}-{YYYYMMDD-HHMM}`

If collision detected:
1. Add random suffix: `-{YYYYMMDD-HHMM}-{random4}`
2. Retry once
3. If still fails, error out

### Q13: Verification performance?

**Answer:** Run in sequence, fail fast

1. `gofmt -s -l .` - fastest, catches formatting
2. `go vet ./...` - quick static analysis
3. `go build ./...` - medium, catches compile errors
4. `go test ./...` - slowest, only if others pass

Skip remaining steps on first failure to speed up iteration.

### Q14: Partial success - 3 of 5 fixes pass?

**Answer: All or nothing**

Reasons:
- Partial fixes may be incomplete/incorrect
- Simpler mental model for users
- If some fixes fail verification, the entire set may need reconsideration
- LLM can re-generate with knowledge of which issues remain

---

## 14. Implementation Checklist

### Phase 1 Checklist ✅ COMPLETED (2026-02-12)
- [x] Create `internal/autofix/types.go`
- [x] Create `internal/autofix/parser.go`
- [x] Create `internal/autofix/parser_test.go`
- [x] Create `prompts/fix_generation.md`
- [x] Create `prompts/fix_correction.md` (bonus)
- [x] Add `SendFixPrompt()` to LLM client
- [x] Unit tests passing
- [ ] Add `--generate-fixes` flag to CLI (deferred to Phase 5)

### Phase 2 Checklist ✅ COMPLETED (2026-02-12)
- [x] Create `internal/autofix/applier.go`
- [x] Create `internal/autofix/applier_test.go`
- [x] Create `internal/verify/verify.go`
- [x] Create `internal/verify/verify_test.go`
- [x] Implement iterative fix loop in `internal/autofix/autofix.go`
- [x] Unit tests passing

### Phase 3 Checklist ✅ COMPLETED (2026-02-12)
- [x] Create `internal/git/git.go`
- [x] Create `internal/git/git_test.go`
- [x] Implement branch creation
- [x] Implement commit and push
- [x] Implement rollback

### Phase 4 Checklist ✅ COMPLETED (2026-02-12)
- [x] Add `CreatePullRequest()` to Bitbucket client
- [x] Add `GetFileContent()` to Bitbucket client
- [x] Add `BranchExists()` to Bitbucket client
- [x] Add `GetPullRequestByBranch()` to Bitbucket client
- [x] Add `GetPullRequest()` to Bitbucket client
- [x] Implement PR title templating (`internal/bitbucket/templates.go`)
- [x] Implement PR description templating (`internal/bitbucket/templates.go`)
- [x] Add `CreateStackedPR()` to AutoFixer
- [x] Add 10 new tests for Bitbucket client methods
- [x] Add 7 test suites for template functions
- [x] All tests passing (26 tests in bitbucket package)
- [x] Code formatted and passes `go vet`

### Phase 5 Checklist
- [ ] Add `fix-pr` subcommand
- [ ] Add `--auto-fix` flag to main command
- [ ] Add `AutoFix` config section
- [ ] Implement pipeline mode detection
- [ ] Update README
- [ ] Update AGENTS.md if needed

### Phase 6 Checklist
- [ ] E2E tests
- [ ] Edge case handling
- [ ] Performance testing
- [ ] Security review
- [ ] Documentation review
- [ ] Release notes

---

## 15. Future Enhancements (Out of Scope)

- **Fix caching:** Store fix responses to avoid re-generating
- **Incremental fixes:** Apply fixes that pass individually
- **Custom verification commands:** Support non-Go projects
- **Fix suggestions UI:** Web interface for reviewing fixes
- **Multi-PR batching:** Fix multiple PRs in one run
- **Fix approval workflow:** Require approval before creating PR
- **Analytics:** Track fix success rates by issue type
