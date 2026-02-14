# Implementation Prompt: Language Detector

## Task
Create `internal/verify/detector.go` to detect programming languages in a repository.

## Requirements

### Function Signature
```go
package verify

// DetectLanguages scans the repository and returns a list of detected languages.
// Returns languages in priority order (most prevalent first).
func DetectLanguages(repoPath string) ([]string, error)
```

### Detection Logic

Scan the repository for these indicators:

**1. Configuration Files** (highest priority):
- `go.mod` or `go.sum` → "go"
- `package.json` → "javascript" (check for TypeScript in devDependencies)
- `tsconfig.json` → "typescript"
- `pyproject.toml`, `setup.py`, `requirements.txt`, `Pipfile` → "python"
- `pom.xml`, `build.gradle` → "java"
- `Cargo.toml` → "rust"
- `Gemfile` → "ruby"
- `composer.json` → "php"

**2. File Extensions** (count occurrences):
- `.go` → "go"
- `.py` → "python"
- `.js`, `.jsx` → "javascript"
- `.ts`, `.tsx` → "typescript"
- `.java` → "java"
- `.rs` → "rust"
- `.rb` → "ruby"
- `.php` → "php"

**3. Directory Markers** (presence indicates language):
- `node_modules/` → "javascript" or "typescript"
- `venv/`, `__pycache__/`, `.venv/` → "python"
- `target/` (with pom.xml) → "java"

### Rules

1. **Ignore**: `vendor/`, `node_modules/`, `.git/`, `dist/`, `build/`, `__pycache__/`
2. **Multi-language**: Return all detected languages, sorted by prevalence
3. **Minimum threshold**: Only include language if ≥ 5 files of that type found
4. **Config files override**: If config file found, always include that language

### Return Value
```go
// Examples:
["go"]                          // Single language repo
["go", "python"]                // Multi-language repo (Go primary)
["javascript", "typescript"]    // JS/TS mixed project
[]                             // No recognized languages (error condition)
```

## Implementation Notes

- Use `filepath.Walk` to traverse the repository
- Track file counts per language
- Check config files first, then count extensions
- Return error if no languages detected
- Keep it simple - no need for complex heuristics initially

## Test Cases Required

Create `internal/verify/detector_test.go`:

```go
// Test scenarios:
func TestDetectLanguages_GoOnly(t *testing.T)
func TestDetectLanguages_Python(t *testing.T)
func TestDetectLanguages_JavaScript(t *testing.T)
func TestDetectLanguages_MultiLanguage(t *testing.T)
func TestDetectLanguages_NoLanguages(t *testing.T)
func TestDetectLanguages_IgnoresVendor(t *testing.T)
func TestDetectLanguages_ConfigFileOverride(t *testing.T)
```

## Success Criteria

- [ ] `DetectLanguages()` function implemented
- [ ] Detects Go, Python, JavaScript, TypeScript
- [ ] Returns languages in priority order
- [ ] Ignores common ignore directories
- [ ] Returns error if no languages found
- [ ] Unit tests pass
- [ ] Works with current pullreview repository (should detect "go")

## Reference
See `docs/multi_language_architecture.md` for overall design.
