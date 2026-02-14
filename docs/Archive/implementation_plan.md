# Implementation Plan for `pullreview` CLI Tool

This plan provides a detailed, actionable, step-by-step guide for implementing the `pullreview` command-line tool, based on the requirements.

---

## Step 1: Project Initialization & Scaffolding ✅ **(Complete)**

- ✅ Initialize a new Go module for the project.
- ✅ Create the following initial files and directories:
  - ✅ `main.go` (entry point, now moved to `cmd/pullreview/main.go`)
  - ✅ `README.md` (project overview, build, and usage instructions)
  - ✅ `pullreview_requirements.md` (requirements)
  - ✅ `implementation_plan.md` (this file)
  - ✅ `pullreview.yaml` (sample config file)
  - ✅ `prompt.md` (editable LLM prompt template)
- ✅ Set up a basic directory structure for modular code:
  - `cmd/pullreview/`
  - `internal/bitbucket/`
  - `internal/llm/`
  - `internal/review/`
  - `internal/config/`
  - `internal/utils/`

**Current State:**  
The project is initialized with the required Go module, directory structure, and placeholder files for each internal package. The CLI entrypoint uses Cobra and references the internal packages. Sample configuration and prompt files are present.  

---


## Step 2: Configuration Management ✅ **(Complete)**



- ✅ Implement logic to load configuration from `pullreview.yaml`.

- ✅ Add support for overriding config values with:

  - Command-line flags (`-u`, `-p`, `--pr`)

  - Environment variables (`BITBUCKET_USERNAME`, `BITBUCKET_APP_PASSWORD`, `LLM_PROVIDER`, `LLM_API_KEY`, `LLM_ENDPOINT`, `PULLREVIEW_PROMPT_FILE`)

- ✅ Validate configuration on startup and provide helpful error messages for missing or invalid values.

**Current State:**  
The configuration system loads settings from `pullreview.yaml`, then applies overrides from environment variables, and finally from command-line flags. All required fields are validated at startup, and clear errors are shown if any are missing. The configuration precedence and supported environment variables are documented in the README.


---


## Step 3: Bitbucket Cloud Integration ✅ **(Complete)**



- ✅ Implement authentication with Bitbucket Cloud using username and app password.

- ✅ Add logic to infer the current PR from the active Git branch by default.

- ✅ Allow explicit PR ID override via command-line flag.

- ✅ Implement API calls to:

  - Fetch PR metadata (title, description, etc.)

  - Fetch the PR diff (unified diff format)

- ✅ Handle API errors gracefully and provide clear feedback to the user.



**Current State:**  
The Bitbucket client (`internal/bitbucket/client.go`) now supports:
- Authentication with Bitbucket Cloud.
- Inferring the PR ID from the current Git branch using the `GetPRIDByBranch` method.
- Fetching PR metadata and diffs via `GetPRMetadata` and `GetPRDiff`.
- All errors are surfaced to the user with clear, actionable messages.
- The CLI (`cmd/pullreview/main.go`) wires these methods together, supporting both PR inference and explicit override.

**API Usage Example:**
- `GetPRIDByBranch(branch string) (string, error)` — Finds the open PR for a branch.
- `GetPRMetadata(prID string) ([]byte, error)` — Fetches PR metadata as JSON.
- `GetPRDiff(prID string) (string, error)` — Fetches the unified diff for a PR.

---



## Step 4: LLM Integration ✅ **(Complete)**



- ✅ Implement support for at least one LLM provider (OpenAI, via Chat API).
- ✅ Load the review prompt from `prompt.md` and inject the PR diff and any relevant context.

- ✅ Send the prompt to the LLM API and receive the response.

- ✅ Make LLM provider, endpoint, and API key configurable via `pullreview.yaml`.



**Current State:**  
The LLM integration is implemented using OpenAI's Chat API. The tool loads the prompt template from `prompt.md`, injects the PR diff, and sends the prompt to the LLM. The response is printed to the console (future steps will parse and post comments). The LLM provider, endpoint, and API key are fully configurable via `pullreview.yaml`, environment variables, or CLI flags. The logic is implemented in `internal/llm/client.go` and wired up in `cmd/pullreview/main.go`.

---



## Step 5: Diff Parsing & Review Preparation ✅ **(Complete)**



- ✅ Parse the Bitbucket PR diff to extract file and line information.

- ✅ Prepare the data structure for mapping LLM responses to Bitbucket's comment API.

- ✅ Ensure the diff is presented to the LLM in a clear, context-rich format.



**Current State:**  
The `internal/review` package now includes:
- A robust unified diff parser (`ParseUnifiedDiff`) that extracts file changes, hunks, and line mappings.
- Data structures (`DiffFile`, `DiffHunk`, `HunkLine`, `LineType`) to represent parsed diff information.
- Methods on `Review` to parse the diff and format it for LLM input with clear file/hunk context.
- Unit tests for diff parsing and formatting, covering single and multiple files, additions, deletions, and edge cases.

---


## Step 6: Comment Generation & Posting ✅ **(Complete)**

- ✅ Parse the LLM response to extract:
  - Inline comments (file, line, comment text)
  - File-level comments (if any)
  - Summary review comment
- ✅ Format all comments in Markdown.
- ✅ Implement API calls to post:
  - Multiple inline comments
  - A summary comment
- ✅ Ensure comments are associated with the correct lines/files in the PR.
- ✅ Add a `--post` flag to control whether comments are posted to Bitbucket or just printed for review.
- ✅ Change output so the tool prints the parsed summary and inline comments (not the raw LLM response) by default.

**Current State:**  
- The Bitbucket client (`internal/bitbucket/client.go`) now provides:
  - `PostInlineComment(prID, filePath, line, text)` for posting inline comments to specific lines/files.
  - `PostSummaryComment(prID, text)` for posting a summary (top-level) comment.
- The review logic (`internal/review/review.go`) includes:
  - `ParseLLMResponse(llmResp string)` to extract inline and summary comments from the LLM output using a simple Markdown convention.
- The CLI (`cmd/pullreview/main.go`) now:
  - Parses the LLM response after receiving it.
  - Prints the parsed summary and all inline comments to the terminal for review.
  - Only posts comments to Bitbucket if the `--post` flag is set.
  - Prints a summary of posted comments if posting is enabled.

**How Comments Are Posted and Previewed:**  
- By default, the tool prints the summary and inline comments for review and does not post to Bitbucket.
- If the `--post` flag is provided, inline comments are posted using the Bitbucket API to the correct file and line, and the summary review is posted as a top-level PR comment.
- All comments are formatted in Markdown.
 
---

## Step 7: Robust LLM Output Parsing and Preparation for Bitbucket Posting ✅ **(Complete)**

**Status:**  
The LLM response parser now supports the new explicit section format for extracting inline comments, file-level comments, and summary from the LLM output. The parser is fully tested using realistic test files and is robust to various LLM output styles.

**What was done:**
- Implemented a parser that extracts structured comments and summary from LLM output using explicit section headers.
- Comprehensive unit tests using realistic test files ensure correct extraction of all comment types and summary.
- Legacy markdown-style parsing has been removed for clarity and maintainability.

**Goal:**  
LLM output is now parsed into structured data, ready for further processing and posting to Bitbucket.

---

### Step 7.1: Match Parsed LLM Comments to Parsed Diff/Input ✅ **(Complete & Integrated)**

**Goal:**  
Ensure that each parsed comment from the LLM output corresponds to a real file and line in the PR diff.

**What was done:**
- Implemented `MatchCommentsToDiff` in `internal/review/review.go` to match each parsed comment (file, line) to the parsed diff structure.
- Added comprehensive unit tests for valid and invalid inline and file-level comments.
- **Integrated into the main review flow:** After parsing the LLM response, only matched comments are printed and posted to Bitbucket. Unmatched comments are clearly reported and not posted.

**Additional Notes:**
- This integration ensures that only actionable, valid comments reach Bitbucket, improving reliability and user experience.
- Unmatched comments (e.g., referencing files or lines not present in the diff) are shown in a separate section for transparency and debugging.

---

### Step 7.2: Prepare Comments for Bitbucket Posting ✅ **(Complete)**

**Goal:**  
Format and organize the matched comments for Bitbucket's API.

**What was done:**
- Only matched comments are converted and posted using the Bitbucket API.
- Inline and file-level comments are handled appropriately.
- Summary comments are prepared and posted as top-level PR comments.

---

### Step 7.3: Integrate with Bitbucket Posting Logic ✅ **(Complete)**

**Goal:**  
Wire up the review flow so that only valid, matched comments are posted to Bitbucket.

**What was done:**
- The main CLI logic now uses the matching function after parsing the LLM output.
- Only matched comments are posted to Bitbucket.
- A summary of posted and unmatched (skipped) comments is printed for user awareness.

---

### Step 7.4: Testing & Validation of Matching and Posting ✅ **(Complete)**

**Goal:**  
Ensure the matching and posting logic is robust and correct.

**What was done:**
- Unit tests cover the matching function, including edge cases (missing files, out-of-range lines, etc.).
- Posting logic is exercised in integration and manual tests.
- End-to-end validation ensures only valid comments are posted, and unmatched comments are handled gracefully.

---

## Step 8: Add GitHub Copilot SDK Integration

**Goal:**
Integrate the GitHub Copilot SDK as an alternative LLM provider for code review, allowing users to leverage GitHub Copilot's AI capabilities instead of OpenAI/OpenRouter.

**Prerequisites:**
- GitHub Copilot CLI must be installed and authenticated on the user's machine
- User must have a GitHub Copilot subscription (free tier available with limited usage)
- Go 1.21+ (current project uses Go 1.24.5, which satisfies this requirement)

**SDK Reference:**
- Repository: https://github.com/github/copilot-sdk
- Go SDK: `github.com/github/copilot-sdk/go`
- Status: Technical Preview (may not be suitable for production use)

---

### Step 8.1: Add GitHub Copilot SDK Dependency

**Tasks:**
- [ ] Add the GitHub Copilot SDK Go package to the project
  ```bash
  go get github.com/github/copilot-sdk/go
  ```
- [ ] Verify the dependency is added to `go.mod` and `go.sum`
- [ ] Test that the package imports correctly

**Files Modified:**
- `go.mod`
- `go.sum`

---

### Step 8.2: Create Copilot Client Package

**Tasks:**
- [ ] Create new package `internal/copilot/` for Copilot SDK integration
- [ ] Implement `Client` struct with the following fields:
  - `Model` (string): The Copilot model to use (e.g., "gpt-4.1", "gpt-5")
  - `ReasoningEffort` (string): Processing intensity ("low", "medium", "high", "xhigh")
  - `Streaming` (bool): Whether to enable streaming responses
- [ ] Implement `NewClient(model, reasoningEffort string) *Client` constructor
- [ ] Implement `SendReviewPrompt(prompt string) (string, error)` method that:
  1. Creates a Copilot SDK client with `copilot.NewClient()`
  2. Starts the client with `client.Start(ctx)`
  3. Creates a session with `client.CreateSession(ctx, &copilot.SessionConfig{...})`
  4. Sends the prompt using `session.SendAndWait()` or streaming with event handlers
  5. Collects and returns the response content
  6. Properly cleans up with `client.Stop()`
- [ ] Implement proper error handling for:
  - Copilot CLI not installed or not in PATH
  - Authentication failures
  - Session creation failures
  - Timeout handling using context

**Files Created:**
- `internal/copilot/client.go`
- `internal/copilot/client_test.go`

**Example Implementation Pattern:**
```go
package copilot

import (
    "context"
    "errors"
    "time"

    copilot "github.com/github/copilot-sdk/go"
)

type Client struct {
    Model           string
    ReasoningEffort string
    Streaming       bool
}

func NewClient(model, reasoningEffort string) *Client {
    if model == "" {
        model = "gpt-4.1"
    }
    return &Client{
        Model:           model,
        ReasoningEffort: reasoningEffort,
    }
}

func (c *Client) SendReviewPrompt(prompt string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    client := copilot.NewClient(&copilot.ClientOptions{
        LogLevel: "error",
    })

    if err := client.Start(ctx); err != nil {
        return "", fmt.Errorf("failed to start Copilot CLI: %w", err)
    }
    defer client.Stop()

    session, err := client.CreateSession(ctx, &copilot.SessionConfig{
        Model:           c.Model,
        ReasoningEffort: c.ReasoningEffort,
    })
    if err != nil {
        return "", fmt.Errorf("failed to create Copilot session: %w", err)
    }

    response, err := session.SendAndWait(copilot.MessageOptions{
        Prompt: prompt,
    }, 0)
    if err != nil {
        return "", fmt.Errorf("failed to get Copilot response: %w", err)
    }

    if response.Data.Content == nil {
        return "", errors.New("empty response from Copilot")
    }

    return *response.Data.Content, nil
}
```

---

### Step 8.3: Update Configuration to Support Copilot Provider

**Tasks:**
- [ ] Add Copilot-specific configuration fields to `Config.LLM` struct:
  ```go
  type Config struct {
      // ... existing fields ...
      LLM struct {
          Provider        string `yaml:"provider"`        // "openai", "openrouter", or "copilot"
          APIKey          string `yaml:"api_key"`         // Not required for Copilot
          Endpoint        string `yaml:"endpoint"`        // Not required for Copilot
          Model           string `yaml:"model"`           // Model name
          ReasoningEffort string `yaml:"reasoning_effort"` // Copilot-specific: low/medium/high/xhigh
      } `yaml:"llm"`
  }
  ```
- [ ] Update `LoadConfigWithOverrides()` to:
  - Skip API key validation when provider is "copilot"
  - Add environment variable support: `LLM_REASONING_EFFORT`
  - Set default model for Copilot if not specified ("gpt-4.1")
- [ ] Update validation logic to not require `api_key` and `endpoint` when provider is "copilot"

**Files Modified:**
- `internal/config/config.go`
- `internal/config/config_test.go`

**Sample YAML Configuration for Copilot:**
```yaml
llm:
  provider: copilot
  model: gpt-4.1           # Optional, defaults to gpt-4.1
  reasoning_effort: medium  # Optional: low, medium, high, xhigh
```

---

### Step 8.4: Update LLM Client to Support Copilot Provider

**Tasks:**
- [ ] Update `internal/llm/client.go` to add Copilot as a supported provider
- [ ] Add `ReasoningEffort` field to the LLM `Client` struct
- [ ] Modify `SendReviewPrompt()` to route to Copilot client when provider is "copilot":
  ```go
  func (c *Client) SendReviewPrompt(prompt string) (string, error) {
      switch strings.ToLower(c.Provider) {
      case "openai", "openrouter":
          return c.sendOpenAI(prompt)
      case "copilot":
          return c.sendCopilot(prompt)
      default:
          return "", fmt.Errorf("unsupported LLM provider: %s", c.Provider)
      }
  }
  ```
- [ ] Implement `sendCopilot(prompt string) (string, error)` method that uses the internal/copilot package

**Files Modified:**
- `internal/llm/client.go`
- `internal/llm/client_test.go`

---

### Step 8.5: Update CLI to Support Copilot Configuration

**Tasks:**
- [ ] Update `cmd/pullreview/main.go` to pass `ReasoningEffort` to the LLM client
- [ ] Add helpful error message if Copilot CLI is not installed when using Copilot provider
- [ ] Update verbose output to show Copilot-specific configuration

**Files Modified:**
- `cmd/pullreview/main.go`

---

### Step 8.6: Update Documentation

**Tasks:**
- [ ] Update `README.md` with:
  - GitHub Copilot SDK installation instructions
  - Copilot CLI setup and authentication requirements
  - Sample configuration for using Copilot as the LLM provider
  - Explanation of `reasoning_effort` setting
  - Note about Technical Preview status
- [ ] Update `pullreview.yaml` sample config with Copilot example (commented out)

**Files Modified:**
- `README.md`
- `pullreview.yaml`

**Documentation to Add:**
```markdown
## Using GitHub Copilot as LLM Provider

### Prerequisites
1. Install GitHub Copilot CLI: https://github.com/github/copilot-cli
2. Authenticate with: `copilot auth login`
3. Verify installation: `copilot --version`

### Configuration
```yaml
llm:
  provider: copilot
  model: gpt-4.1           # Options: gpt-4.1, gpt-5, etc.
  reasoning_effort: medium  # Options: low, medium, high, xhigh
```

### Notes
- No API key required (uses your GitHub Copilot subscription)
- The Copilot SDK is in Technical Preview
- Billing follows your Copilot subscription (free tier has limited usage)
```

---

### Step 8.7: Testing

**Tasks:**
- [ ] Write unit tests for `internal/copilot/client.go`:
  - Test client construction with defaults
  - Test error handling when Copilot CLI is not available (mocked)
- [ ] Write unit tests for updated config validation (Copilot provider case)
- [ ] Write integration test (manual) to verify end-to-end Copilot flow
- [ ] Test error scenarios:
  - Copilot CLI not installed
  - Copilot CLI not authenticated
  - Invalid model specified
  - Timeout handling

**Files Created/Modified:**
- `internal/copilot/client_test.go`
- `internal/config/config_test.go`
- `internal/llm/client_test.go`

---

### Step 8.8: Error Handling and User Experience

**Tasks:**
- [ ] Implement clear error messages for common Copilot issues:
  - "Copilot CLI not found. Please install from https://github.com/github/copilot-cli"
  - "Copilot CLI not authenticated. Run 'copilot auth login' first."
  - "Copilot request timed out. Try again or use a different provider."
- [ ] Add startup check to verify Copilot CLI is available when provider is "copilot"
- [ ] Log Copilot session details in verbose mode

**Files Modified:**
- `internal/copilot/client.go`
- `cmd/pullreview/main.go`

---

**Implementation Order:**
1. Step 8.1 - Add SDK dependency
2. Step 8.2 - Create Copilot client package
3. Step 8.3 - Update configuration
4. Step 8.4 - Update LLM client routing
5. Step 8.5 - Update CLI
6. Step 8.6 - Update documentation
7. Step 8.7 - Testing
8. Step 8.8 - Error handling polish

**Estimated Complexity:** Medium
- The SDK handles most of the complexity (CLI communication, session management)
- Main work is integration and proper error handling
- Configuration changes are straightforward

**Risks and Considerations:**
- SDK is in Technical Preview - API may change
- Requires users to have Copilot CLI installed and authenticated
- Response times may vary based on Copilot service load
- Premium request quotas apply to Copilot usage

---

### Additional SDK Features Discovered (Future Enhancements)

The following features were discovered during research and could be added in future iterations:

#### 1. Custom Tools
Define tools that Copilot can invoke during review:
- `get_file_context` - Retrieve full file when diff context is insufficient
- `get_function_definition` - Look up function implementations
- `check_test_coverage` - Query test coverage for changed files
- `search_codebase` - Find similar patterns elsewhere

#### 2. Streaming Responses
Enable real-time output with `Streaming: true`:
- Show review comments as they are generated
- Display progress during long PR reviews
- Better user engagement during processing

#### 3. Session Management
- **Infinite Sessions**: Auto context management for large PRs
- **Session Persistence**: Resume interrupted reviews
- **Multi-pass reviews**: Maintain context across multiple prompts

#### 4. Reasoning Effort Levels
Configure review depth via `ReasoningEffort`:
| Level | Use Case |
|-------|----------|
| `low` | Quick formatting checks |
| `medium` | Standard code reviews |
| `high` | Security reviews, complex logic |
| `xhigh` | Architecture reviews, critical systems |

#### 5. Model Availability (Pro License)
| Model | Cost | Notes |
|-------|------|-------|
| GPT-5 mini | Free (unlimited) | Included with Pro |
| GPT-4.1 | Free (unlimited) | Included with Pro |
| GPT-4o | Free (unlimited) | Included with Pro |
| Claude Sonnet 4.5 | 1x premium request | ~300 reviews/month with Pro |
| Claude Opus 4.5 | 3x premium request | ~100 reviews/month with Pro |

#### 6. BYOK (Bring Your Own Key)
Support custom providers:
```go
Provider: &copilot.ProviderConfig{
    Type:    "openai",  // or "azure", "anthropic"
    BaseURL: "https://api.your-company.com/v1",
    APIKey:  os.Getenv("CUSTOM_API_KEY"),
}
```

#### 7. File Attachments
Attach images for UI review scenarios:
```go
Attachments: []copilot.Attachment{
    {Type: "file", Path: "/path/to/screenshot.png"},
}
```



---

## Step 9: Testing & Validation

- ✅ Write unit tests for:
  - ✅ Config loading and validation
  - ✅ Bitbucket API integration (mocked)
  - ✅ LLM API integration (mocked)
  - ✅ Diff parsing and mapping
  - ✅ CLI argument parsing
  - ✅ **LLM response parsing for summary and inline comments**
  - ✅ **Bitbucket comment posting (mocked)**
- Test end-to-end on sample PRs (small, large, multi-file).
- Validate on Windows environment.

**Current State:**  
- Unit tests exist for:
  - Diff parsing and mapping
  - LLM response parsing for summary and inline comments
  - Bitbucket comment posting (mocked HTTP)
  - Config loading and overrides
  - LLM client error handling
- All tests pass, confirming correct behavior for new features and CLI logic.

---

## Step 10: Linting, Documentation, and Polish ✅ **(Complete)**

- ✅ Set up linting (`go vet`) and ensure code passes all checks.
- ✅ Format all code with `gofmt`.
- ✅ Update `README.md` with:
  - Build instructions
  - Configuration guide
  - Usage examples
  - Customizing the prompt
- ✅ Document all public functions and modules.
- ✅ Ensure `pullreview_requirements.md` and `implementation_plan.md` are up to date with the current state of the project.

**Current State:**  
All code is linted (`go vet`), formatted (`gofmt`), and passes all tests. Documentation is current and reflects the latest features, including LLM model selection and OpenRouter support.

---

## Step 11: Review, Release, and Maintenance

- Review the codebase for completeness and clarity.
- Tag a release and provide installation instructions.
- Plan for future enhancements (multi-platform support, more LLMs, advanced review strategies, etc.).
- Solicit feedback from users and contributors.

---

**End of Step-by-Step Plan**