# Testing Status

This document tracks which commands and flags have been **actually tested on real pull requests**.

**Last Updated:** 2026-02-14

---

## 📋 TESTING MATRIX

| Command | Flags | Tested On | Language | Result | Details |
|---------|-------|-----------|----------|--------|---------|
| `pullreview` | (none) | bhunter PR #5 | C# | ✅ PASS | Displayed 1 comment |
| `pullreview` | `-v` | Multiple PRs | C# | ✅ PASS | Verbose output working |
| `pullreview` | `--post` | - | - | ❌ NOT TESTED | Need to test review+post on real PR |
| `pullreview fix-pr` | `-v` | bhunter PR #8 | C# | ✅ PASS | Fix existing (1 comment) → PR #9 |
| `pullreview fix-pr` | `--dry-run -v` | bhunter PR #6, #8 | C# | ✅ PASS | Combined autofix, local only |
| `pullreview fix-pr` | `--post -v` | bhunter PR #6 | C# | ✅ PASS | Combined autofix: 1 LLM call → posted → PR #7 |
| `pullreview fix-pr` | `--post -v` | menuplanning-api PR #89 | C# | ✅ PASS | Fix existing (5 comments) → posted → PR #92 |
| `pullreview fix-pr` | `--skip-verification -v` | menuplanning-api PR #89 | C# | ✅ PASS | Skipped verification → PR #91 |
| `pullreview fix-pr` | `--max-iterations N` | - | - | ❌ NOT TESTED | Flag exists but not tested |
| `pullreview fix-pr` | `--branch-prefix <name>` | - | - | ❌ NOT TESTED | Flag exists but not tested |
| `pullreview fix-pr` | `--pr <ID>` | - | - | ❌ NOT TESTED | Manual PR ID not tested |

---

## 🎯 WORKFLOWS VERIFIED ON REAL PRS

| Workflow | Prompt File | LLM Calls | Tested | Result |
|----------|-------------|-----------|--------|--------|
| **Review-only** | `prompt.md` | 1 | ✅ YES (bhunter #5) | Finds issues, displays comments |
| **Auto-fix (no comments)** | `autofix_prompt.md` | **1** | ✅ YES (bhunter #6) | Finds + fixes in ONE call |
| **Fix existing comments** | `fix_prompt.md` | 1+ | ✅ YES (bhunter #8, menuplanning #89) | Converts comments to fixes |

**Key Achievement:** Combined autofix reduces LLM calls from 2 → 1 (50% reduction)

---

## 🌐 LANGUAGE SUPPORT

| Language | Detection | Build | Tests | Tested On Real PRs | Status |
|----------|-----------|-------|-------|-------------------|--------|
| **C#** | ✅ | ✅ dotnet build | ✅ dotnet test | menuplanning-api, bhunter | ✅ FULLY WORKING |
| **Go** | ✅ | go build | go test | Not tested on real Go PR | ⚠️ NOT TESTED |

---

## ❌ NOT TESTED ON REAL PRS

| Scenario | Priority | Notes |
|----------|----------|-------|
| Fix correction iterations (multi-pass) | MEDIUM | When first fix fails, LLM retries (not yet triggered) |
| Go project PR | MEDIUM | Need to test on real Go PR |
| Mixed language PR (Go + C#) | LOW | Detector implemented but not tested |
| Pipeline mode (CI/CD) | LOW | Auto-detected in CI environment |
| Max iterations exceeded | LOW | What happens after N failed attempts |