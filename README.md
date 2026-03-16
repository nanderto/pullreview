# pullreview

`pullreview` is a command-line tool that automates code review for Bitbucket Cloud pull requests using a Large Language Model (LLM). It fetches PR diffs, sends them to an LLM for review, and posts AI-generated comments (inline and summary) back to Bitbucket. The tool is designed for Windows and is highly configurable.

Usage example

```
# Windows
pullreview.exe --repo bitbucket_repo_name --pr 89 --verbose

# Linux
./pullreview --repo bitbucket_repo_name --pr 89 --verbose
```

---

## Features

- **Bitbucket Cloud Integration:** Fetches pull request diffs and posts comments using Bitbucket’s REST API.
- **Local Branch Review:** Review local branch changes against a target branch (e.g., `main`) without needing a Bitbucket PR, using the `--local` flag.
- **LLM-Powered Reviews:** Sends diffs to a configurable LLM (OpenAI, Claude, Copilot, etc.) for automated code review.
- **Customizable Prompts:** Users can edit the prompt template to control the review style and focus.
- **Flexible Authentication:** Supports credentials via config file, command-line flags, or environment variables.
- **Markdown Comments:** Posts both inline and summary comments in Markdown format.
- **Windows Compatibility:** Designed to work seamlessly on Windows systems.

---

## Prerequisites

- **Go 1.24.5 or later** ([Download Go](https://golang.org/dl/))
- **Git** (for branch detection and PR inference)
- **Bitbucket Cloud account** with an API token
- **LLM API key** (e.g., OpenAI, Claude, etc.)

---

## Installation

Clone the repository and build the executable:

```sh
git clone https://your.repo.url/pullreview.git
cd pullreview
go build ./cmd/pullrequest
```

---

## Automated Comment Posting

After the LLM review is generated, `pullreview` will:

- **Parse the LLM response** for both inline and summary comments.
- **Print the summary and all inline comments** to the terminal for review.
- **Prompt you to confirm** before posting to Bitbucket (unless `--skip-inline` is set).
  - Default behavior: Shows review, then asks "Should I post this review to Bitbucket? [y/N]"
  - If you confirm (y/yes), all inline and summary comments are posted to the PR.
  - If you decline (n/no or Enter), no comments are posted.
  - Use `--skip-inline` flag for non-interactive mode (no prompt).
- All comments are posted in Markdown format.


### LLM Response Format



The tool supports two formats for inline comments in the LLM output:

- **1. Code block format (legacy, still supported):**  
  Use code blocks with the `inline` tag and specify the file and line, e.g.:

  ```

  ```inline path/to/file.go:42

  This is an inline comment for file.go at line 42.

  ```

  ```


- **2. Natural language format (recommended):**  
  Write inline comments in a human-friendly way, referencing the file and line(s) at the start of the line, e.g.:
  ```
  path/to/file.go Line 42: This is an inline comment for file.go at line 42.
  path/to/other.go Lines 10-12: These lines could use more descriptive variable names.
  ```
  - Both `Line N:` and `Lines N-M:` are supported.
  - The tool will post one inline comment per referenced line.

- **Summary comment:**  

  Any text outside of inline comment blocks or natural language inline comment lines is treated as the summary and posted as a top-level PR comment.



#### Example LLM Output (Both Formats Supported)



```

Overall, this PR improves code clarity. See inline comments for details.



foo.go Line 10: Consider renaming this variable for clarity.


bar.go Lines 25-26: Possible off-by-one error here.


Thank you for your work!
```


Or, using the code block format:

```

Overall, this PR improves code clarity. See inline comments for details.


```inline foo.go:10
Consider renaming this variable for clarity.
```
```

```inline bar.go:25
Possible off-by-one error here.
```

In both examples, inline comments will be posted to the specified files/lines, and the summary will be posted as a top-level comment.


## Configuration


All settings are managed via a single YAML config file (`pullreview.yaml`). You can also provide credentials via command-line flags or environment variables.

**Config File Resolution:**
The tool searches for `pullreview.yaml` in this order (first found wins):
1. Current working directory
2. Next to the executable (e.g., `C:\tools\pullreview.yaml`)
3. Explicitly via `--config` flag (highest priority)

No config file is required for `--local` reviews if LLM settings are provided via environment variables.

**Value Precedence (highest to lowest):**
1. Command-line flags
2. Environment variables
3. Values from `pullreview.yaml`

All required configuration fields must be set by one of these methods, or the tool will exit with an error.


### Example `pullreview.yaml`

- Copy the example [pullreview.yaml.example](pullreview.yaml.example) configuration file and rename it to `pullreview.yaml`
- Update the values

### Environment Variables



The following environment variables are supported and override values from the config file:

- `BITBUCKET_API_TOKEN` – Bitbucket API token
- `LLM_PROVIDER` – LLM provider (e.g., openai, openrouter, copilot)
- `LLM_API_KEY` – LLM API key (not required for copilot provider)
- `LLM_ENDPOINT` – LLM API endpoint (not required for copilot provider)
- `LLM_MODEL` – LLM model name
- `PULLREVIEW_PROMPT_FILE` – Path to the prompt file


### Command-Line Flags

- `--config`, `-c` - Path to config file (default: `pullreview.yaml`)
- `--pr` - Pull request ID (optional; inferred from branch by default)
- `--email` - Bitbucket account email (overrides config/env)
- `--token` - Bitbucket API token (overrides config/env)
- `--post` - Enable posting to Bitbucket when used with `--skip-inline` (default: false)
- `--skip-inline` - Skip interactive confirmation prompt (non-interactive mode)
- `--local` - Review local branch changes against target branch (no Bitbucket PR required; optional positional arg: path to repo folder)
- `--target` - Target branch to diff against when using `--local` (default: `main`)
- `--verbose`, `-v` - Enable verbose output (shows full diff and API details)
- `--version` - Show version and exit


## Usage

### Basic Usage (Interactive Mode - Default)

```sh
./pullreview.exe
```

By default, `pullreview` will:

- Infer the current PR from the active Git branch.
- Fetch the PR diff from Bitbucket Cloud.
- Load the review prompt from `prompt.md`, inject the PR diff, and send it to the configured LLM (e.g., OpenAI).
- Print the parsed summary and all inline comments to the terminal.
- **Prompt you to confirm before posting:** "Should I post this review to Bitbucket? [y/N]"
  - Type `y` or `yes` to post comments
  - Type `n`, `no`, or press Enter to cancel (default)

### Preview Mode (No Posting)

To review without posting and skip the prompt:

```sh
./pullreview.exe --skip-inline
```

### Auto-Post Mode (Non-Interactive)

To automatically post without prompting (useful for CI/CD):

```sh
./pullreview.exe --post --skip-inline
```

### Local Branch Review (No PR Required)

Review your current branch against `main` without a Bitbucket PR:

```sh
./pullreview --local
```

Review a specific repo folder:

```sh
./pullreview --local /path/to/repo
```

Review against a different target branch:

```sh
./pullreview --local --target develop
```

### Specify a PR ID

```sh
./pullreview.exe --pr 123
```

### Override Credentials

```sh
./pullreview.exe --token your_bitbucket_api_token
```

---

## Auto-Fix Usage

The `pullreview` tool includes automated fix generation and application capabilities. After reviewing a PR, it can generate code fixes, apply them, verify they work, and create a stacked pull request with the fixes.

### Apply Fixes to Current PR

```sh
# Auto-fix issues and create stacked PR (uses existing PR comments)
./pullreview.exe fix-pr

# Generate new review instead of using existing comments
./pullreview.exe fix-pr --regenerate

# Specify PR ID explicitly
./pullreview.exe fix-pr --pr 123

# Dry-run mode (apply fixes locally without committing)
./pullreview.exe fix-pr --dry-run

# Skip verification (dangerous - not recommended)
./pullreview.exe fix-pr --skip-verification

# Apply fixes without creating PR
./pullreview.exe fix-pr --no-pr
```

**Note:** By default, `fix-pr` fetches existing review comments from the Bitbucket PR, ensuring deterministic fixes for the same issues that were reviewed. Use `--regenerate` if you want to generate a fresh review instead (may produce different issues).

### Auto-Fix Configuration

Enable auto-fix in `pullreview.yaml`:

```yaml
autofix:
  enabled: true
  auto_create_pr: true
  max_iterations: 5
  verify_build: true
  verify_tests: true
  verify_lint: true
  branch_prefix: pullreview-fixes
  fix_prompt_file: fix_prompt.md
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

### How Auto-Fix Works

1. **Review Generation:** Analyzes the PR and identifies issues
2. **Fix Generation:** LLM generates code fixes for identified issues
3. **Fix Application:** Applies fixes to local files
4. **Verification:** Runs build/test/lint to verify fixes work
5. **Iteration:** If verification fails, requests corrected fixes (up to max_iterations)
6. **Git Operations:** Creates branch, commits, and pushes fixes
7. **Stacked PR:** Creates a new PR targeting the original PR's branch

### Example Auto-Fix Workflow

```sh
# Developer creates PR
git checkout -b feature/add-logger
git push origin feature/add-logger

# PR created in Bitbucket

# Auto-fix issues found in review
cd /path/to/repo
./pullreview fix-pr

# Output:
# 🔧 Auto-fixing PR #456...
# 📝 Found 3 issue(s) to fix
# ✅ Applied 3 fix(es) to 2 file(s)
# ✅ Pushed fixes to branch: pullreview-fixes-feature-add-logger-20260212T153042Z
# ✅ Stacked PR created: https://bitbucket.org/...

# Developer reviews stacked PR, merges to their branch
```

---

## Flag Behavior Summary

| Command | Behavior |
|---------|----------|
| `pullreview` | Shows review, prompts for confirmation, posts if confirmed |
| `pullreview --skip-inline` | Shows review only, no prompt, no posting |
| `pullreview --post --skip-inline` | Shows review and auto-posts (no prompt) |
| `pullreview --pr 123` | Review specific PR #123 |
| `pullreview --local` | Review current branch against main (no Bitbucket needed) |
| `pullreview --local /path/to/repo` | Review a specific repo's branch against main |
| `pullreview --local --target develop` | Review current branch against develop |
| `pullreview --verbose` | Show full diff and detailed API output |

---

## Customizing the AI Review

Edit the `prompt.md` file to change the instructions or review style sent to the LLM. This allows you to tailor the AI’s feedback to your team’s needs.

### Prompt File Resolution

The tool searches for `prompt.md` in the following order (first found wins):

1. **Explicit path** from `prompt_file` in config or `PULLREVIEW_PROMPT_FILE` env var (absolute paths used as-is; relative paths resolved against the config file’s directory)
2. **Current working directory** — place a `prompt.md` in your repo for project-specific review instructions
3. **Executable directory** — the installation default (e.g., `C:\tools\prompt.md`)

This means you can override the default prompt on a per-repo basis by dropping a `prompt.md` in your repository root, or set a global default next to the installed binary. No configuration is needed if `prompt.md` exists in either location.
### Custom Project-Specific Instructions

You can now add custom project-specific instructions that take **highest priority** over the base prompt!

To add custom instructions for a specific project:

1. Create a file named `.pullreview-custom.md` or `pullreview-custom.md` in your project repository root (the repo being reviewed)
2. Add your project-specific review guidelines, coding standards, or framework-specific rules
3. The tool will automatically detect and apply these instructions with **absolute precedence**

**Example custom instructions:**
```markdown
1. This project uses TypeScript 2.9.2. Do not recommend syntax that is not supported in this version.
2. Ignore the return type in method `MyMethod` at line 20 in abc.ts - this is an intentional design choice.
```

**Priority Behavior:**
- Custom instructions override base prompt rules when there's a conflict
- Custom instructions are clearly marked as `[CUSTOM_PROJECT_INSTRUCTIONS]` to the AI
- The AI evaluates custom instructions first before applying base review rules

**Note:** The custom prompt file should only exist in your target project repositories (the repos being reviewed), not in the pullreview tool directory.

---

## Diff Parsing & Review Mapping

The `pullreview` tool parses the Bitbucket PR diff using a robust unified diff parser. This enables:

- Extraction of file changes, hunks, and line mappings from the diff.
- Mapping of LLM responses (inline comments, summary) to the correct files and lines in Bitbucket.
- Presentation of the diff to the LLM in a clear, context-rich format, including file names and hunk headers.

### How Diff Parsing Works

- The tool parses the unified diff into structured data: files, hunks, and line mappings.
- Each hunk includes context lines, additions, and deletions, with accurate line numbers for both the old and new files.
- This structure allows precise association of LLM-generated comments with the correct lines in the PR.

### LLM Review Context: Diff vs. Full File

By default, the LLM only sees the unified diff for each pull request. This means:

- The LLM receives only the lines that were added, removed, or modified, plus a small amount of surrounding context (the “hunks” in the diff).
- The LLM does **not** see the entire file before or after the change, nor unchanged code outside the diff hunks.
- This approach is efficient and focuses the review on what changed, but may miss issues that require broader file or project context.

**Future Option:**  
A future version may allow sending the full file (before or after changes) to the LLM for deeper context. This will be configurable.

---



## Example: Diff Formatting for LLM

The tool formats the parsed diff for LLM input with clear file and hunk context. For example:

```
File: foo.go
  @@ -1,6 +1,7 @@
      package main
    - func hello() {
    -     println("Hello, world!")
    + func hello(name string) {
    +     println("Hello,", name)
      }
```

This helps the LLM understand exactly what changed and where, improving the quality and relevance of its review comments.

---

## LLM Integration

The tool supports multiple LLM providers:
- **OpenAI** - Direct OpenAI API
- **OpenRouter** - Access to multiple models via OpenRouter
- **Copilot** - GitHub Copilot via the Copilot SDK (requires Copilot CLI)

---

### GitHub Copilot SDK Integration

The tool supports GitHub Copilot as an LLM provider via the [GitHub Copilot SDK](https://github.com/github/copilot-sdk). This allows you to use your existing GitHub Copilot subscription for code reviews.

#### Prerequisites for Copilot

1. **GitHub Copilot Subscription** - You need an active Copilot subscription (Individual, Pro, Business, or Enterprise)
2. **Copilot CLI** - Install from https://github.com/github/copilot-cli
3. **Authentication** - Run `copilot auth login` to authenticate
4. **Organization Policy** (for Business/Enterprise) - Your organization admin must enable the Copilot CLI policy in settings

#### Copilot Configuration

```yaml
llm:
  provider: copilot
  model: gpt-4.1           # Optional, defaults to gpt-4.1
```

**Note:** No API key is required when using Copilot - authentication is handled through the Copilot CLI.

#### Available Copilot Models

- `gpt-4.1` (default)
- `gpt-5`
- Other models as supported by your Copilot subscription

#### Environment Variables for Copilot

- `LLM_PROVIDER=copilot` - Set provider to Copilot
- `LLM_MODEL` - Override the model

#### Important Notes

- The Copilot SDK is currently in **Technical Preview** and may not be suitable for production use
- Billing follows your Copilot subscription (free tier has limited usage)
- Premium request quotas apply based on your subscription level

---

### OpenAI & OpenRouter

The tool supports sending review prompts to an LLM provider using the OpenAI-compatible API format. This includes OpenAI, OpenRouter, and other compatible services.

- The prompt template is loaded from `prompt.md`.

- The PR diff is injected into the prompt at the `(DIFF_CONTENT_HERE)` placeholder.

- The prompt is sent to the LLM API (e.g., OpenAI, OpenRouter).
- The LLM's response is printed to the console.

- You can select the model via the `model` field in your config.


**Example OpenAI Configuration:**



```yaml

llm:

  provider: openai

  api_key: your_openai_api_key

  endpoint: https://api.openai.com/v1/chat/completions

  model: gpt-3.5-turbo
```



**Example OpenRouter Configuration:**

```yaml
llm:
  provider: openai
  api_key: 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
  endpoint: https://openrouter.ai/api/v1/chat/completions
  model: arcee-ai/trinity-large-preview:free
```
You can use any model supported by OpenRouter by specifying its name in the `model` field.

**How it works:**


1. The tool loads your prompt template and replaces `(DIFF_CONTENT_HERE)` with the actual PR diff.
2. It sends the prompt to the OpenAI Chat API using your API key and endpoint.
3. The LLM's review (in Markdown) is printed to the terminal.

**Sample output:**

```
🤖 Sending review prompt to LLM...
✅ Received LLM review response:
------ BEGIN LLM REVIEW ------
[AI-generated review content here]
------- END LLM REVIEW -------
```

Support for additional LLM providers can be added by extending `internal/llm/client.go`.


---

## Bitbucket API Usage & PR Inference

The tool uses the Bitbucket Cloud REST API to authenticate, infer the pull request (PR) from your current Git branch, and fetch PR metadata and diffs.

### How PR Inference Works

- By default, `pullreview` detects your current Git branch using `git rev-parse --abbrev-ref HEAD`.
- It queries Bitbucket for open PRs where the source branch matches your current branch.
- If a matching PR is found, its ID is used for review. You can override this by specifying `--pr <id>`.

### Bitbucket API Methods Used

- **Authentication:** Validates credentials via the `/user` endpoint.
- **PR Lookup:** Finds open PRs for a branch using `/repositories/{workspace}/pullrequests?q=source.branch.name="branch"`.
- **PR Metadata:** Fetches PR details from `/repositories/{workspace}/pullrequests/{id}`.
- **PR Diff:** Retrieves the unified diff from `/repositories/{workspace}/pullrequests/{id}/diff`.

### Error Handling

- All API errors (authentication, PR lookup, metadata, diff) are reported with clear, actionable messages.
- If no PR is found for your branch, you’ll be prompted to check your branch or specify a PR ID manually.

### Example Output

```
✅ Successfully authenticated with Bitbucket (workspace: your_workspace)
🔎 Inferred branch: feature/my-feature
🔎 Inferred PR ID: 123
✅ Fetched PR metadata for PR #123
✅ Fetched PR diff for PR #123 (length: 2048 bytes)
```

---


---

## Sample Workflow

1. Ensure your config (`pullreview.yaml`) and prompt (`prompt.md`) are set up.
2. Open a terminal in your repository.
3. Build and run `pullreview.exe`.
4. Review the AI-generated comments on your Bitbucket PR.

---

## Contributing

Contributions are welcome! Please open issues or submit pull requests for bug fixes, new features, or improvements.

---

## License

[MIT License](LICENSE)

---

## Disclaimer

This tool is provided as-is. Use at your own risk. Ensure you do not expose sensitive information in your prompts or configuration files.
