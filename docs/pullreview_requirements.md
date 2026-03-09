# pullreview CLI Tool – Requirements

## Overview

`pullreview` is a command-line tool designed to automate code review for Bitbucket Cloud pull requests using a Large Language Model (LLM). It fetches PR diffs, sends them to an LLM for review, and posts the AI-generated comments back to Bitbucket. The tool is intended for use on Windows and is configurable via a single YAML file.

---

## Functional Requirements

### 1. Bitbucket Integration

- **Bitbucket Cloud Support:** The tool must work with Bitbucket Cloud (not Bitbucket Server).
- **Authentication:** Authenticate using a Bitbucket Cloud username and app password.
  - Credentials can be provided via:
    - A config file (`bhunter.yaml`)
    - Command-line arguments (`-u` for username, `-p` for app password)
    - Environment variables (`BITBUCKET_USERNAME`, `BITBUCKET_APP_PASSWORD`)
- **PR Identification:**
  - By default, infer the target PR from the current Git branch.
  - Optionally, accept a PR ID as a command-line flag to override branch inference.
- **Fetch PR Diff:** Use Bitbucket’s REST API to fetch the diff for the identified pull request.

### 2. LLM Integration

- **Model Support:** The tool must support integration with an LLM (e.g., OpenAI, Claude, Copilot via GitHub APIs).
- **API Key Management:** LLM API keys/configuration are provided in a config file (see `Answers.md` for details).
- **Prompt Customization:** The review prompt sent to the LLM must be loaded from an editable file (`prompt.md`) so users can customize the review style and instructions.

### 3. Review and Commenting

- **Diff Analysis:** The tool must parse the PR diff and prepare it for LLM review.
- **Comment Types:**
  - **Inline Comments:** The tool must support posting comments on specific lines and files.
  - **Summary Comment:** The tool must also post a summary review as a single comment.
- **Comment Format:** All comments must be posted in Markdown format.
- **Multiple Comments:** The tool must be able to post multiple inline comments as well as a summary comment in a single run.

### 4. Configuration & Usability

- **Single Config File:** All settings (Bitbucket, LLM, etc.) are managed via a single YAML config file.
- **Windows Compatibility:** The tool must run on Windows, with all paths, commands, and dependencies compatible.
- **Command-Line Usage:** The tool should be invoked as a single command (e.g., `pullreview bitbucket-pr <pr-id>`), with sensible defaults and override options.

---

## Non-Functional Requirements

- **No Rate Limit Handling:** The tool does not need to handle API rate limits.
- **No Sensitive Data Redaction:** The tool does not need to redact secrets or sensitive information from diffs before sending to the LLM.
- **Extensibility:** The tool should be designed to allow easy swapping of LLM providers and prompt templates.

---

## Files and Artifacts

- `main.go` – Entry point for the CLI tool.
- `bhunter.yaml` – Main configuration file.
- `prompt.md` – Editable prompt template for LLM review.
- `Answers.md` – (Not committed) For storing API keys/secrets.
- Other Go source files for modular functionality (Bitbucket API, LLM integration, etc.).

---

## Out of Scope

- Bitbucket Server (self-hosted) support.
- Handling of API rate limits or retries.
- Redaction of sensitive data from code diffs.
- Support for platforms other than Windows (initially).

---

## Summary

The `pullreview` CLI tool will streamline code review for Bitbucket Cloud PRs by leveraging LLMs, providing flexible configuration, and supporting both inline and summary comments in Markdown. It is designed for ease of use, extensibility, and Windows compatibility.
