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
**Next:** Proceed to Step 2: Configuration Management.

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

**Next:** Proceed to Step 4: LLM Integration.

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



## Step 7.5: Append Unmatched Comments as Bullet Points Under the File Summary ✅ **(Complete)**



**Goal:**  
When inserting the file summary, any unmatched comments should be added as plain bullet points directly underneath the file summary (with no heading), both in the console output and when posting the summary to Bitbucket.

**What was done:**  
- The CLI logic was updated so that after matching comments to the diff, any unmatched comments are appended as plain Markdown bullet points directly under the summary, with no heading.
- This composed summary (original summary + unmatched comments as bullets) is printed in the summary section and posted as the summary comment to Bitbucket.
- Unmatched comments are formatted as `- [file:line] comment` or `- [file] comment` for file-level comments.

- The previous separate "Unmatched Comments" section was removed from the output, as unmatched comments now appear directly under the summary.

- This ensures all reviewer feedback is visible in one place, improving transparency and usability.

**Status:**  
✅ Implemented and tested. The summary now always includes unmatched comments as plain bullet points (with no heading) when present.

---

## Step 8: Testing & Validation


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

## Step 9: Linting, Documentation, and Polish ✅ **(Complete)**

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

## Step 10: Review, Release, and Maintenance

- Review the codebase for completeness and clarity.
- Tag a release and provide installation instructions.
- Plan for future enhancements (multi-platform support, more LLMs, advanced review strategies, etc.).
- Solicit feedback from users and contributors.

---

**End of Step-by-Step Plan**