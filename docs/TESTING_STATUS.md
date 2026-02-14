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

## 📊 RECENT TEST RESULTS

### bhunter PR #8 (2026-02-14)
- **Language**: C#
- **Command**: `pullreview fix-pr -v`
- **Workflow**: Fix existing comments
- **Result**: ✅ SUCCESS
- **Details**: 1 comment → 1 fix → dotnet build passed → PR #9 created
- **Verification**: Build only (tests skipped per config)

### bhunter PR #6 (2026-02-14)
- **Language**: C#
- **Command**: `pullreview fix-pr --post -v`
- **Workflow**: Combined autofix with posting (ONE LLM call)
- **Result**: ✅ SUCCESS
- **Details**: 1 issue + 1 fix in ONE call → comment posted → dotnet build passed → PR #7 created
- **Verification**: Build only (tests skipped per config)

### menuplanning-api PR #89 (2026-02-14)
- **Language**: C#
- **Command**: `pullreview fix-pr --post -v`
- **Workflow**: Fix existing comments with posting
- **Result**: ✅ SUCCESS
- **Details**: 5 comments → 1 fix → dotnet build passed → PR #92 created
- **Verification**: Build only (tests skipped per config)

### menuplanning-api PR #89 - Earlier Test (2026-02-13)
- **Language**: C#
- **Command**: `pullreview fix-pr --skip-verification -v`
- **Workflow**: Fix existing comments
- **Result**: ✅ SUCCESS
- **Details**: 3 comments → 1 fix → verification skipped → PR #91 created
- **Verification**: Skipped via flag

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
