# pullreview

`pullreview` is a command-line tool that automates code review for Bitbucket Cloud pull requests using a Large Language Model (LLM). It fetches PR diffs, sends them to an LLM for review, and posts AI-generated comments (inline and summary) back to Bitbucket. The tool is designed for Windows and is highly configurable.

Usage example

```
# Windows
pullreview.exe --repo menuplanning-api --pr 89 --verbose

# Linux
./pullreview --repo menuplanning-api --pr 89 --verbose
```

---

## Features

- **Bitbucket Cloud Integration:** Fetches pull request diffs and posts comments using Bitbucket’s REST API.
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
go build -o pullreview.exe ./cmd/pullrequest
```

---

## Automated Comment Posting

After the LLM review is generated, `pullreview` will:

- **Parse the LLM response** for both inline and summary comments.
- **Print the summary and all inline comments** to the terminal for review.
- **Optionally post comments to Bitbucket** using the `--post` flag.
  - If `--post` is not set, no comments are posted (preview mode).
  - If `--post` is set, all inline and summary comments are posted to the PR.
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

**Configuration Precedence:**  
1. Values from `pullreview.yaml` are loaded first.
2. Environment variables override values from the config file.
3. Command-line flags override both environment variables and config file values.

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

- `--token` Bitbucket API token
- `--pr` Pull request ID (optional; inferred from branch by default)


## Usage

### Basic Usage (Preview Mode)

```sh
./pullreview.exe
```

By default, `pullreview` will:

- Infer the current PR from the active Git branch.
- Fetch the PR diff from Bitbucket Cloud.
- Load the review prompt from `prompt.md`, inject the PR diff, and send it to the configured LLM (e.g., OpenAI).
- Print the parsed summary and all inline comments to the terminal.
- **No comments are posted to Bitbucket unless you use the `--post` flag.**

### Post Comments to Bitbucket

To actually post the summary and inline comments to Bitbucket, use the `--post` flag:

```sh
./pullreview.exe --post
```

or

```sh
./pullreview.exe --post=true
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


---

## The --post Flag

- `--post` (default: false):  
  If set, posts the parsed summary and inline comments to Bitbucket.  
  If not set, only prints the comments for review.

---

## Customizing the AI Review




Edit the `prompt.md` file to change the instructions or review style sent to the LLM. This allows you to tailor the AI’s feedback to your team’s needs.

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
