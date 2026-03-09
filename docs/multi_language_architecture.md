# Multi-Language Auto-Fix Architecture

## Overview
To support auto-fix and verification for multiple languages, the architecture must:
- Detect the project's primary language(s)
- Dynamically select and run the correct build/lint/test tools
- Be extensible for new languages/toolchains

---

## Components

### 1. LanguageDetector
- Scans the project directory
- Identifies language(s) by file extensions, config files, etc.
- Returns a list of detected languages (e.g., ["go", "python", "js"])

### 2. Verifier Interface
- Defines methods: `Verify()`, `Build()`, `Lint()`, `Test()`
- Each language implements its own verifier (e.g., GoVerifier, PythonVerifier)

```go
// Example Go interface
 type Verifier interface {
     Verify() error
     Build() error
     Lint() error
     Test() error
 }
```

### 3. Verifier Registry
- Maps language names to verifier implementations
- Allows easy addition of new languages

### 4. Orchestrator
- Uses LanguageDetector to find languages
- Instantiates the correct verifier(s)
- Runs verification steps for each language
- Aggregates results

---

## Example Flow
1. LanguageDetector scans repo → returns ["go", "python"]
2. Orchestrator creates GoVerifier and PythonVerifier
3. For each verifier:
    - Run Build, Lint, Test
    - Collect errors
4. If all verifiers pass, fix is complete
5. If any fail, send errors to LLM for correction

---

## Extensibility
- To add a new language:
    - Implement a new Verifier (e.g., JavaScriptVerifier)
    - Register it in the Verifier Registry
    - Update LanguageDetector if needed

---

## Example Directory Structure
```
internal/
    verify/
        go_verifier.go
        python_verifier.go
        js_verifier.go
        registry.go
        detector.go
```

---

## Notes
- Each verifier runs language-specific commands (e.g., `go build`, `python -m unittest`, `npm test`)
- The architecture supports multiple languages in one repo
- Results are aggregated for unified reporting

---

## Next Steps
- Implement LanguageDetector
- Refactor Verifier logic to use registry and interface
- Add verifiers for each supported language
- Update orchestration logic
