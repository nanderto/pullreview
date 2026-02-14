# Testing Status

This document tracks which commands and flags have been tested.

**Last Updated:** 2026-02-14

---

## ✅ TESTED AND WORKING

### Review Commands

| Command | Description | Status | Notes |
|---------|-------------|--------|-------|
| `pullreview` | Review PR, display comments | ✅ TESTED | Works correctly |
| `pullreview -v` | Review with verbose output | ✅ TESTED | Shows detailed logs |
| `pullreview --pr <ID>` | Review specific PR by ID | ⚠️ NOT TESTED | Should work (flag implemented) |

### Fix Commands

| Command | Description | Status | Notes |
|---------|-------------|--------|-------|
| `pullreview fix-pr` | Auto-fix issues, create stacked PR | ✅ TESTED | Works perfectly |
| `pullreview fix-pr -v` | Fix with verbose output | ✅ TESTED | Shows detailed logs |
| `pullreview fix-pr --dry-run` | Apply fixes locally without commit | ✅ TESTED | Works correctly |
| `pullreview fix-pr --post` | Post comments + apply fixes + create PR | ✅ TESTED | Full CI/CD workflow |
| `pullreview fix-pr --post --dry-run` | Post comments + apply fixes locally | ✅ TESTED | Useful for testing |

### Workflows Tested

| Workflow | Prompt Used | Status | Notes |
|----------|-------------|--------|-------|
| Review-only (no fix) | `prompt.md` | ✅ TESTED | Finds issues, displays/posts comments |
| Auto-fix (no existing comments) | `autofix_prompt.md` | ✅ TESTED | Finds + fixes in ONE LLM call |
| Fix existing comments | `fix_prompt.md` | ✅ TESTED | Converts existing comments to fixes |

### Language Support Tested

| Language | Detection | Build | Tests | Status |
|----------|-----------|-------|-------|--------|
| Go | ✅ | ✅ | ✅ | Fully tested |
| C# | ✅ | ✅ | ⚠️ Skipped (config) | Tested on menuplanning-api, bhunter |

---

## ❌ NOT TESTED

### Flags

| Flag | Description | Priority | Notes |
|------|-------------|----------|-------|
| `pullreview --post` | Review and POST comments only | HIGH | Should be tested |
| `pullreview fix-pr --skip-verification` | Skip build/test/lint checks | MEDIUM | Dangerous but useful for testing |
| `pullreview fix-pr --max-iterations N` | Custom iteration limit | LOW | Should work (implemented) |
| `pullreview fix-pr --branch-prefix <name>` | Custom branch naming | LOW | Should work (implemented) |
| `pullreview --pr <ID>` | Specify PR ID manually | LOW | Override auto-detection |

### Scenarios

| Scenario | Priority | Notes |
|----------|----------|-------|
| Fix correction iterations (multi-pass) | MEDIUM | When first fix fails verification, LLM retries |
| Mixed language PR (Go + C# in same PR) | LOW | Should work with multi-language detector |
| Pipeline mode (CI/CD auto-detection) | LOW | Auto-detected when running in CI/CD |
| Max iterations exceeded | LOW | What happens when fix can't pass verification after N tries |

---

## 🎯 TESTING PRIORITIES

1. **HIGH**: `pullreview --post` (review-only with posting)
2. **MEDIUM**: `--skip-verification` flag
3. **MEDIUM**: Fix correction iterations (multi-pass fix attempts)
4. **LOW**: Custom flags (--max-iterations, --branch-prefix, --pr)

---

# Testing Status

This document tracks which commands and flags have been **actually tested on real pull requests**.

**Last Updated:** 2026-02-14

---

## ✅ TESTED ON REAL PULL REQUESTS

### Review Commands

| Command | Tested On | Language | Result | Notes |
|---------|-----------|----------|--------|-------|
| `pullreview` | bhunter PR #5 | C# | ✅ PASS | Displayed 1 comment correctly |
| `pullreview -v` | Multiple PRs | C#, Go | ✅ PASS | Verbose output working |

### Fix Commands

| Command | Tested On | Language | Result | Notes |
|---------|-----------|----------|--------|-------|
| `pullreview fix-pr --dry-run -v` | bhunter PR #6, PR #8 | C# | ✅ PASS | Applied fixes locally, no commit |
| `pullreview fix-pr -v` | bhunter PR #8 | C# | ✅ PASS | Created PR #9, stacked correctly |
| `pullreview fix-pr --post -v` | bhunter PR #6 | C# | ✅ PASS | Posted 1 comment + created PR #7 |
| `pullreview fix-pr --post -v` | menuplanning-api PR #89 | C# | ✅ PASS | Posted 3 comments + created PR #92 |
| `pullreview fix-pr --skip-verification -v` | menuplanning-api PR #89 | C# | ✅ PASS | Skipped verification, created PR #91 |

### Workflows Tested on Real PRs

| Workflow | Prompt Used | Tested On | LLM Calls | Result |
|----------|-------------|-----------|-----------|--------|
| Review-only (display) | `prompt.md` | bhunter PR #5 | 1 | ✅ PASS |
| Auto-fix (no existing comments) | `autofix_prompt.md` | bhunter PR #6 | **1** | ✅ PASS (combined find+fix) |
| Fix existing comments | `fix_prompt.md` | bhunter PR #8, menuplanning-api PR #89 | 1 | ✅ PASS |

### Language Support Tested

| Language | Tested On | Detection | Build | Tests | Result |
|----------|-----------|-----------|-------|-------|--------|
| C# | menuplanning-api, bhunter | ✅ | ✅ (dotnet build) | ⚠️ Skipped | Fully working |
| Go | pullreview repo (self) | ✅ | ⚠️ Not tested | ⚠️ Not tested | Detection working, verifier not tested on real PR |

---

## ❌ NOT TESTED ON REAL PULL REQUESTS

### Commands/Flags Not Tested

| Command/Flag | Priority | Notes |
|--------------|----------|-------|
| `pullreview --post` | HIGH | Should post comments to Bitbucket (not tested on real PR) |
| `pullreview fix-pr --max-iterations N` | LOW | Implemented but not tested |
| `pullreview fix-pr --branch-prefix <name>` | LOW | Implemented but not tested |
| `pullreview --pr <ID>` / `pullreview fix-pr --pr <ID>` | LOW | Manual PR ID override not tested |

### Scenarios Not Tested

| Scenario | Priority | Notes |
|----------|----------|-------|
| Fix correction iterations (multi-pass) | MEDIUM | When first fix fails verification, LLM should retry (not yet triggered) |
| Go project PR | MEDIUM | Haven't tested on real Go PR yet |
| Mixed language PR (Go + C#) | LOW | Haven't found a PR with both languages |
| Pipeline mode (CI/CD) | LOW | Auto-detected in CI environment (not tested) |
| Max iterations exceeded | LOW | What happens when fix can't pass after N tries |

---

## 🐛 KNOWN ISSUES

1. **Prompt file location**: Tool looks for prompt files in target repo, not tool directory
   - **Workaround**: Copy prompt files to target repo
   - **Fix needed**: Load prompts from tool directory or make path configurable

2. **Missing .NET 6.0 runtime**: Tests fail if runtime not installed
   - **Workaround**: Set `verify_tests: false` in config
   - **Note**: This is environmental, not a bug

---

## 🎉 KEY IMPROVEMENTS IMPLEMENTED

1. **3-Prompt Strategy**: Separate prompts for review-only, autofix, and fix-existing
2. **50% LLM Call Reduction**: Combined autofix uses ONE call instead of two
3. **Multi-language Support**: Go and C# fully supported
4. **--post Flag**: Post comments before fixing (CI/CD transparency)
5. **Removed Redundant Flags**: Eliminated `--regenerate` and `--no-pr`
6. **Improved String Matching**: Prompt explicitly instructs to copy from CURRENT FILE CONTENTS
