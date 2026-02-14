# Auto-Fix Implementation Plan

## Status Overview

| Phase | Description | Status | Completed |
|-------|-------------|--------|-----------|
| 1 | Fix Generation & Parsing | ✅ Complete | 2026-02-12 |
| 2 | Fix Application & Verification | ✅ Complete | 2026-02-12 |
| 3 | Git Operations | ✅ Complete | 2026-02-12 |
| 4 | Stacked PR Creation | ✅ Complete | 2026-02-12 |
| 5 | CLI Implementation | ✅ Complete | 2026-02-13 |
| 6 | Multi-Language Support | ⏳ In Progress | - |
| 7 | Testing & Hardening | ⏳ Not Started | - |

---

## Overview

The `pullreview` tool now supports automated code fixing with stacked PR creation. **Core functionality is complete and working.** Current focus is adding multi-language detection and verification support.

---

## What's Working

✅ **Core Auto-Fix Workflow**
- LLM generates fixes from review comments (JSON format)
- Applies fixes with smart search-and-replace
- Iterative verification loop (up to N iterations)
- Creates git branches and commits
- Creates stacked PRs on Bitbucket
- CLI command: `pullreview fix-pr`

✅ **Verification** (currently Go-only)
- `gofmt` - code formatting check
- `go vet` - static analysis
- `go build` - compilation
- `go test` - test execution

✅ **Key Features**
- Backup/restore on failure
- Template-based commit messages and PR descriptions
- Dry-run mode for local testing
- Pipeline mode detection for CI/CD
- Can use existing Bitbucket comments or regenerate review

---

## Current Focus: Multi-Language Support

**Problem**: Verification is hardcoded to Go tools. Need to detect project language(s) and run appropriate verifiers.

**Solution**: See `docs/multi_language_architecture.md`

**Progress**: Architecture designed → **Next: Implement detector**

---

## Implementation Phases

### Phase 1: Fix Generation & Parsing ✅

**Deliverables**:
- JSON-based fix format from LLM
- Robust JSON parsing with markdown fence stripping
- Support for alternative field names

**Key Files**:
- `internal/autofix/types.go` - Fix, FixResponse, FixResult types
- `internal/autofix/autofix.go` - Fix generation logic

---

### Phase 2: Fix Application & Verification ✅

**Deliverables**:
- Smart search-and-replace fix application (handles whitespace variations)
- Backup/restore mechanism for failed fixes
- Iterative verification loop (max N iterations)
- Error feedback to LLM for correction

**Key Files**:
- `internal/autofix/types.go` - Applier implementation
- `internal/verify/verify.go` - Verification orchestration
- `internal/verify/verify_test.go` - Unit tests

---

### Phase 3: Git Operations ✅

**Deliverables**:
- Branch creation with timestamp-based naming
- File staging and commits
- Push to remote
- Branch existence checks

**Key Files**:
- `internal/git/operations.go` - Git wrapper functions

**Branch Naming**: `{prefix}/{source-branch}-{YYYYMMDD-HHMM}`

---

### Phase 4: Stacked PR Creation ✅

**Deliverables**:
- Bitbucket API integration for PR creation
- Template-based PR titles and descriptions
- Markdown escaping for security
- Duplicate PR detection
- Remote branch verification

**Key Files**:
- `internal/bitbucket/client.go` - CreatePullRequest, GetPullRequestByBranch, BranchExists
- `internal/bitbucket/templates.go` - PR title/description templating
- `internal/autofix/autofix.go` - CreateStackedPR method

**Tests**: 26 tests passing in bitbucket package

---

### Phase 5: CLI Implementation ✅

**Command**: `pullreview fix-pr [flags]`

**Key Flags**:
- `--pr <id>` - Specify PR ID (or auto-detect from branch)
- `--dry-run` - Apply fixes locally without committing
- `--regenerate` - Generate new review instead of using existing comments
- `--no-pr` - Skip stacked PR creation
- `--skip-verification` - Skip build/test (dangerous)
- `--max-iterations <n>` - Override max iterations
- `--verbose` - Enable debug output

**Features**:
- Auto-detects PR from current git branch
- Can use existing Bitbucket comments or regenerate review
- Pipeline mode detection for CI/CD
- Machine-readable JSON output in pipeline mode

**Key Files**:
- `cmd/pullreview/main.go` - CLI implementation (runFixPR function)

---

### Phase 6: Multi-Language Support ⏳ IN PROGRESS

**What's Done**:
- ✅ Architecture designed (see `docs/multi_language_architecture.md`)

**Next**: Create `internal/verify/detector.go` - Language detection logic

**Remaining**:
- Create `internal/verify/interface.go` - Verifier interface
- Create `internal/verify/registry.go` - Verifier registry
- Refactor to `internal/verify/go_verifier.go` - Extract Go logic
- Create `internal/verify/python_verifier.go` - Python support
- Create `internal/verify/javascript_verifier.go` - JS/TS support
- Update `AutoFixer.applyAndVerify()` to use detector + registry
- Add tests

---

### Phase 7: Testing & Hardening ⏳ NOT STARTED

**Remaining**:
- `internal/autofix/autofix_test.go` - Core orchestration tests
- `internal/autofix/types_test.go` - Type tests
- `internal/git/operations_test.go` - Git operations tests
- E2E test: Happy path (fix → verify → PR)
- E2E test: Iteration loop (fix fails → correction → success)
- E2E test: Max iterations exceeded
- E2E test: Pipeline mode
- Security: Validate LLM responses
- Performance: Large diffs and many fixes

---

## Architecture

### Data Flow

```
User → Fetch PR → LLM Generate Fixes → Apply Fixes → 
Detect Language(s) → Verify → [Pass/Fail Loop] → Commit → Push → Create Stacked PR
```

### Key Components

| Component | File | Responsibility |
|-----------|------|----------------|
| AutoFixer | `internal/autofix/autofix.go` | Orchestrates entire workflow |
| Applier | `internal/autofix/types.go` | Applies fixes with backup/restore |
| Verifier | `internal/verify/verify.go` | Runs build/lint/test (Go-only currently) |
| Git Ops | `internal/git/operations.go` | Branch, commit, push operations |
| BB Client | `internal/bitbucket/client.go` | PR creation and fetching |
| LLM Client | `internal/llm/client.go` | LLM communication |

### File Structure

```
internal/
├── autofix/       # Fix generation, application, orchestration (728 lines)
├── verify/        # Build/lint/test verification (268 lines) [Multi-language next]
├── git/           # Git operations
├── bitbucket/     # Bitbucket API integration
└── llm/           # LLM communication

cmd/pullreview/
└── main.go        # CLI with fix-pr subcommand (467 lines)
```

---

## Configuration

Configuration in `pullreview.yaml`:

```yaml
auto_fix:
  enabled: true
  auto_create_pr: true
  max_iterations: 5
  verify_build: true
  verify_tests: true
  verify_lint: true
  pipeline_mode: false  # Auto-detected from CI env vars
  branch_prefix: "pullreview-fixes"
  fix_prompt_file: "prompts/fix_generation.md"
  commit_message_template: "🤖 Auto-fix: {issue_summary}..."
  pr_title_template: "🤖 Auto-fixes for PR #{pr_id}: {original_title}"
  pr_description_template: "..."
```

Environment variable overrides:
- `PULLREVIEW_AUTOFIX_ENABLED`
- `PULLREVIEW_AUTOFIX_MAX_ITERATIONS`
- `PULLREVIEW_AUTOFIX_PIPELINE_MODE`

---

## Usage Examples

**Basic auto-fix** (detects PR from current branch):
```bash
pullreview fix-pr
```

**Specify PR ID**:
```bash
pullreview fix-pr --pr 123
```

**Dry-run** (apply locally, don't commit):
```bash
pullreview fix-pr --dry-run
```

**Generate new review** (instead of using existing comments):
```bash
pullreview fix-pr --regenerate
```

**Skip PR creation** (just commit to fix branch):
```bash
pullreview fix-pr --no-pr
```

**Pipeline mode** (auto-detected in CI):
```bash
CI=true pullreview fix-pr --pr 123
```

---

## Key Design Decisions

**Fix Format**: Structured JSON
- Reliable parsing with `encoding/json`
- Supports multiple fixes per file
- Clear structure for line ranges and replacements

**Git Operations**: Shell out to `git` command (not go-git library)
- Simpler, more reliable
- Better auth handling
- Easier debugging

**User Confirmation**: None required
- Running `fix-pr` is explicit consent
- Stacked PR serves as approval mechanism

**Fix Batching**: One commit for all fixes
- Atomic change (all or nothing)
- Cleaner git history
- Simpler rollback

**Verification**: Run in sequence, fail fast
- Order: gofmt → vet → build → test
- Skip remaining on first failure

**Max Iterations**: Exit with code 1, restore original state
- Clear error messaging
- No partial changes

---

## What's Next After Current Step

**Complete Phase 6**: 7 more tasks (interface, registry, verifiers)  
**Then Phase 7**: Testing & hardening

---

## Appendix: Fix Format Details

**LLM Fix Format** (JSON):
```json
{
  "fixes": [
    {
      "file": "relative/path/to/file.go",
      "line_start": 45,
      "line_end": 47,
      "original_code": "exact code being replaced",
      "fixed_code": "the corrected code",
      "issue_addressed": "Brief description"
    }
  ],
  "summary": "Brief summary of all fixes"
}
```

**Prompts**:
1. **Fix Generation** (`prompts/fix_generation.md`) - Initial fix generation
2. **Fix Correction** (`prompts/fix_correction.md`) - Iterative error correction

Both embedded as defaults in `autofix.go`, overridable via config file.

---

## Future Enhancements (Out of Scope)

- Fix caching to avoid re-generation
- Incremental fixes (apply fixes that pass individually)
- Custom verification commands for non-standard projects
- Fix suggestions UI (web interface)
- Multi-PR batching
- Analytics on fix success rates
