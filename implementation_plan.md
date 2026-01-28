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


## Step 5: Diff Parsing & Review Preparation

- Parse the Bitbucket PR diff to extract file and line information.
- Prepare the data structure for mapping LLM responses to Bitbucket's comment API.
- Ensure the diff is presented to the LLM in a clear, context-rich format.

---

## Step 6: Comment Generation & Posting

- Parse the LLM response to extract:
  - Inline comments (file, line, comment text)
  - File-level comments (if any)
  - Summary review comment
- Format all comments in Markdown.
- Implement API calls to post:
  - Multiple inline comments
  - A summary comment
- Ensure comments are associated with the correct lines/files in the PR.

---

## Step 7: Command-Line Interface & User Experience

- Implement a user-friendly CLI with clear help and usage messages.
- Support the following usage patterns:
  - Default (infer PR from branch): `pullreview`
  - Specify PR ID: `pullreview --pr <id>`
  - Override credentials: `pullreview -u <username> -p <app_password>`
- Print a summary of actions (e.g., number of comments posted, summary text) after execution.

---

## Step 8: Testing & Validation

- Write unit tests for:
  - Config loading and validation
  - Bitbucket API integration (mocked)
  - LLM API integration (mocked)
  - Diff parsing and mapping
  - CLI argument parsing
- Test end-to-end on sample PRs (small, large, multi-file).
- Validate on Windows environment.

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