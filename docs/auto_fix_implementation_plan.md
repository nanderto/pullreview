# Auto-Fix Implementation Plan

## Status Overview

| Phase | Description | Status | Completed |
|-------|-------------|--------|-----------|
| 1 | Fix Generation & Parsing | ✅ Complete | 2026-02-12 |
| 2 | Fix Application & Verification | ✅ Complete | 2026-02-12 |
| 3 | Git Operations | ✅ Complete | 2026-02-12 |
| 4 | Stacked PR Creation | ✅ Complete | 2026-02-12 |
| 5 | CLI Implementation | ✅ Complete | 2026-02-13 |
| 6 | Multi-Language Support | ✅ Complete (Go, C#) | 2026-02-14 |
| 7 | Testing & Hardening | ⏳ In Progress | - |

---

## Overview

The `pullreview` tool now supports automated code fixing with stacked PR creation. **Core functionality is complete and working.** Multi-language detection and verification support is implemented for **Go and C#**, with extensible architecture for additional languages.

---

## What's Working

✅ **Core Auto-Fix Workflow**
- LLM generates fixes from review comments (JSON format)
- Applies fixes with smart search-and-replace
- Iterative verification loop (up to N iterations)
- Creates git branches and commits
- Creates stacked PRs on Bitbucket
- CLI command: `pullreview fix-pr`

✅ **Multi-Language Verification**
- **Auto-detects** project language(s) (Go, C#)
- **Go**: `gofmt`, `go vet`, `go build`, `go test`
- **C#**: `dotnet build`, `dotnet test`
- **Configurable**: Enable/disable build, test, lint per language

✅ **Key Features**
- Backup/restore on failure
- Template-based commit messages and PR descriptions
- Dry-run mode for local testing
- Pipeline mode detection for CI/CD
- Can use existing Bitbucket comments or regenerate review

---

## Current Status

**Phase 6 Complete**: Multi-language architecture implemented for Go and C#. System is extensible for additional languages.

**Phase 7 In Progress**: Testing & hardening. See `docs/TESTING_STATUS.md` for real PR test results.

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

### Phase 6: Multi-Language Support ✅ COMPLETE (Go, C#)

**Implemented**:
- ✅ Language detector (`internal/verify/detector.go`)
  - Detects **Go**: `go.mod`, `*.go` files
  - Detects **C#**: `*.csproj`, `*.sln`, `*.cs` files
- ✅ Verifier interface (`internal/verify/verify.go`)
  - `type Verifier interface { Verify() error }`
  - `VerificationConfig` with `verify_build`, `verify_tests`, `verify_lint` flags
- ✅ Go verifier (`internal/verify/verify.go` - `runGoVerification()`)
  - Runs: `gofmt`, `go vet`, `go build`, `go test`
  - Respects config flags
- ✅ C# verifier (`internal/verify/csharp_verifier.go`)
  - Finds `.sln` file recursively
  - Runs: `dotnet build`, `dotnet test`
  - Respects config flags
- ✅ Dynamic dispatcher (`internal/verify/verify.go`)
  - `NewVerifier()` calls `DetectLanguages()` and stores results
  - `RunAll()` uses `switch` statement to call language-specific verifier
- ✅ Integration with auto-fix workflow
  - Auto-detects language and runs appropriate verifier
  - No changes needed to `AutoFixer` - it calls `verifier.RunAll()`

**Architecture**: Switch-based (no registry needed for small language set)

**Extensibility**: Adding new languages requires:
1. Add detection logic to `detector.go`
2. Create `internal/verify/{language}_verifier.go`
3. Add case to `switch` in `RunAll()`

**Future Languages** (not yet implemented):
- JavaScript/TypeScript (`javascript_verifier.go`)
  - Detects: `package.json`, `*.js`, `*.ts`, `*.jsx`, `*.tsx`
  - Runs: `npm install`, `npm run build`, `npm test`, `npm run lint`
  - **Single verifier handles all JS frameworks** (Angular, React, Svelte, Vue, etc.)
- Python (`python_verifier.go`)
  - Detects: `*.py`, `requirements.txt`, `setup.py`, `pyproject.toml`
  - Runs: `pylint`, `pytest`, `mypy`
- Rust (`rust_verifier.go`)
  - Detects: `Cargo.toml`, `*.rs`
  - Runs: `cargo fmt --check`, `cargo clippy`, `cargo build`, `cargo test`

**Tested**:
- ✅ C# verification on real PRs (menuplanning-api, bhunter repos)
- ❌ Go verification not yet tested on real Go PR

---

### Phase 7: Testing & Hardening ⏳ IN PROGRESS

**Completed**:
- ✅ Real PR testing on C# projects (see `docs/TESTING_STATUS.md`)
- ✅ 3-prompt strategy tested (review-only, combined autofix, fix existing)
- ✅ Workflows tested: `--dry-run`, `--post`, `--skip-verification`
- ✅ C# language detection and verification tested
- ✅ Stacked PR creation tested
- ✅ Iterative fix application tested

**Remaining**:
- Unit tests:
  - `internal/autofix/autofix_test.go` - Core orchestration tests
  - `internal/autofix/types_test.go` - Type tests
  - `internal/git/operations_test.go` - Git operations tests
  - `internal/verify/detector_test.go` - Language detection tests
  - `internal/verify/csharp_verifier_test.go` - C# verifier tests
- E2E tests:
  - Multi-pass iteration loop (fix fails verification → LLM correction → success)
  - Max iterations exceeded behavior
  - Pipeline mode (CI environment)
  - Go project PR (end-to-end)
  - Mixed language PR (Go + C#)
- Security: Validate LLM responses for malicious code injection
- Performance: Large diffs and many fixes
- Edge cases: Empty fixes, invalid JSON, network failures

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
| Verifier | `internal/verify/verify.go` | Orchestrates verification (multi-language) |
| Detector | `internal/verify/detector.go` | Detects project language(s) |
| Go Verifier | `internal/verify/verify.go` | Go-specific verification |
| C# Verifier | `internal/verify/csharp_verifier.go` | C#-specific verification |
| Git Ops | `internal/git/operations.go` | Branch, commit, push operations |
| BB Client | `internal/bitbucket/client.go` | PR creation and fetching |
| LLM Client | `internal/llm/client.go` | LLM communication |

### File Structure

```
internal/
├── autofix/       # Fix generation, application, orchestration
├── verify/        # Multi-language verification
│   ├── verify.go           # Orchestration + Go verifier
│   ├── detector.go         # Language detection (Go, C#)
│   ├── csharp_verifier.go  # C# verification
│   └── (future: javascript_verifier.go, python_verifier.go, etc.)
├── git/           # Git operations
├── bitbucket/     # Bitbucket API integration
└── llm/           # LLM communication

cmd/pullreview/
└── main.go        # CLI with pullreview and fix-pr commands
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

## What's Next

**Phase 7**: Complete testing & hardening
- Add unit tests for all components
- Test on real Go PRs
- Add JavaScript/TypeScript verifier for broader language support

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
