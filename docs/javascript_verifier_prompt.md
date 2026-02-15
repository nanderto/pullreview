# JavaScript/TypeScript Verifier Implementation Prompt

## Goal

Add JavaScript/TypeScript verification support to `pullreview` tool, enabling it to verify fixes for any JavaScript/TypeScript project (React, Angular, Svelte, Vue, Next.js, etc.) using standard npm/yarn/pnpm workflows.

## Background

The multi-language verification system is already implemented and working for Go and C#. The architecture uses:
- **Language Detector** (`internal/verify/detector.go`) - Detects project languages
- **Language-specific Verifiers** - Implement `Verify() error` interface
- **Dynamic Dispatcher** (`internal/verify/verify.go`) - Routes to correct verifier based on detected language

Reference implementations:
- **Go Verifier**: See `runGoVerification()` in `internal/verify/verify.go`
- **C# Verifier**: See `internal/verify/csharp_verifier.go`

## Requirements

### 1. Update Language Detector

**File**: `internal/verify/detector.go`

Add JavaScript/TypeScript detection:

```go
func DetectLanguages(repoPath string) ([]string, error) {
    // Existing: go, csharp
    // Add: javascript
    
    // Detection criteria:
    // - package.json exists
    // - One or more: *.js, *.jsx, *.ts, *.tsx files
    // - Optional: tsconfig.json, node_modules/, package-lock.json, yarn.lock, pnpm-lock.yaml
}
```

**Language identifier**: Return `"javascript"` for both JS and TS projects (TypeScript is a superset, same tooling).

### 2. Create JavaScript Verifier

**File**: `internal/verify/javascript_verifier.go`

**Design Principles**:
- **Framework-agnostic**: One verifier handles ALL JavaScript frameworks (React, Angular, Svelte, Vue, Next.js, SvelteKit, etc.)
- **Package manager detection**: Auto-detect npm, yarn, or pnpm
- **Script-based**: Run standard npm scripts defined in `package.json`
- **Respects config**: Honor `verify_build`, `verify_tests`, `verify_lint` flags

**Implementation**:

```go
package verify

import (
    "os"
    "os/exec"
    "path/filepath"
)

type JavaScriptVerifier struct {
    repoPath        string
    packageManager  string // "npm", "yarn", or "pnpm"
    config          *VerificationConfig
}

func NewJavaScriptVerifier(repoPath string, config *VerificationConfig) (*JavaScriptVerifier, error) {
    // 1. Verify package.json exists
    // 2. Detect package manager:
    //    - pnpm-lock.yaml → pnpm
    //    - yarn.lock → yarn
    //    - package-lock.json → npm
    //    - default → npm
    // 3. Return verifier instance
}

func (v *JavaScriptVerifier) Verify() error {
    // IMPORTANT: Install dependencies first (if node_modules doesn't exist or is outdated)
    if err := v.installDependencies(); err != nil {
        return fmt.Errorf("dependency installation failed: %w", err)
    }

    // Run verification steps (fail fast on first error)
    if v.config.VerifyLint {
        if err := v.runLint(); err != nil {
            return err
        }
    }

    if v.config.VerifyBuild {
        if err := v.runBuild(); err != nil {
            return err
        }
    }

    if v.config.VerifyTests {
        if err := v.runTests(); err != nil {
            return err
        }
    }

    return nil
}

func (v *JavaScriptVerifier) installDependencies() error {
    // Run: npm install / yarn install / pnpm install
    // Only if needed (check node_modules existence)
}

func (v *JavaScriptVerifier) runLint() error {
    // Check if "lint" script exists in package.json
    // If yes: run `npm run lint` / `yarn lint` / `pnpm lint`
    // If no: skip silently (not all projects have lint configured)
}

func (v *JavaScriptVerifier) runBuild() error {
    // Check if "build" script exists in package.json
    // If yes: run `npm run build` / `yarn build` / `pnpm build`
    // If no: return error (build should exist for production projects)
}

func (v *JavaScriptVerifier) runTests() error {
    // Check if "test" script exists in package.json
    // If yes: run `npm test` / `yarn test` / `pnpm test`
    // If no: skip silently (some projects don't have tests yet)
}

func (v *JavaScriptVerifier) hasScript(scriptName string) (bool, error) {
    // Parse package.json and check if scripts[scriptName] exists
}
```

### 3. Update Dispatcher

**File**: `internal/verify/verify.go`

Add JavaScript case to `RunAll()`:

```go
func (v *Verifier) RunAll() error {
    for _, lang := range v.languages {
        switch lang {
        case "go":
            if err := v.runGoVerification(); err != nil {
                return err
            }
        case "csharp":
            if err := v.runCSharpVerification(); err != nil {
                return err
            }
        case "javascript":
            if err := v.runJavaScriptVerification(); err != nil {
                return err
            }
        default:
            return fmt.Errorf("unsupported language: %s", lang)
        }
    }
    return nil
}

func (v *Verifier) runJavaScriptVerification() error {
    verifier, err := NewJavaScriptVerifier(v.repoPath, v.config)
    if err != nil {
        return fmt.Errorf("failed to create JavaScript verifier: %w", err)
    }
    return verifier.Verify()
}
```

## Technical Details

### Package Manager Detection Priority

1. **pnpm**: Check for `pnpm-lock.yaml`
2. **yarn**: Check for `yarn.lock`
3. **npm**: Check for `package-lock.json` or default

### Script Execution

Use Go's `exec.Command()` to run package manager commands:

```go
cmd := exec.Command(v.packageManager, "run", "build")
cmd.Dir = v.repoPath
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
if err := cmd.Run(); err != nil {
    return fmt.Errorf("build failed: %w", err)
}
```

### Handling Missing Scripts

- **lint**: Optional - skip if not defined
- **build**: Required - return error if not defined (or skip with warning for dev-only projects)
- **test**: Optional - skip if not defined or if script is placeholder like `"test": "echo \"Error: no test specified\" && exit 1"`

### Error Messages

Provide clear, actionable errors:
- "JavaScript project detected but package.json not found"
- "npm install failed - check network connection and package.json dependencies"
- "Build script not found in package.json - add a 'build' script or disable verify_build"
- "Tests failed - see output above for details"

### Configuration

Use existing `VerificationConfig` flags:

```yaml
# pullreview.yaml
auto_fix:
  verify_build: true   # Runs npm run build
  verify_tests: true   # Runs npm test
  verify_lint: true    # Runs npm run lint
```

## Framework Support Matrix

This single verifier handles:

| Framework | Build Command | Test Command | Lint Command |
|-----------|--------------|--------------|--------------|
| **React (CRA)** | `npm run build` | `npm test` | `npm run lint` (if configured) |
| **React (Vite)** | `npm run build` | `npm test` | `npm run lint` |
| **Next.js** | `npm run build` | `npm test` | `npm run lint` |
| **Angular** | `npm run build` | `npm test` | `npm run lint` |
| **Svelte/SvelteKit** | `npm run build` | `npm test` | `npm run lint` |
| **Vue** | `npm run build` | `npm test` | `npm run lint` |
| **Nuxt** | `npm run build` | `npm test` | `npm run lint` |

**Key Insight**: All modern JS frameworks expose their CLI tools (ng, vite, next, etc.) via npm scripts in `package.json`. The verifier doesn't need to know which framework - it just runs the scripts.

## Testing Strategy

### Unit Tests

**File**: `internal/verify/javascript_verifier_test.go`

Test cases:
- ✅ Detect npm (package-lock.json present)
- ✅ Detect yarn (yarn.lock present)
- ✅ Detect pnpm (pnpm-lock.yaml present)
- ✅ Parse package.json to check script existence
- ✅ Handle missing package.json
- ✅ Handle missing build script
- ✅ Handle missing test script (should not fail)
- ✅ Respect config flags (skip build/test/lint if disabled)

### Integration Tests

Test on real projects:
- React project with Vite
- Angular project
- Next.js project
- Svelte project

## Success Criteria

1. ✅ Language detector identifies JavaScript/TypeScript projects correctly
2. ✅ Verifier auto-detects package manager (npm/yarn/pnpm)
3. ✅ Runs build successfully on real React/Angular/Svelte projects
4. ✅ Runs tests successfully (or skips if no tests)
5. ✅ Runs lint successfully (or skips if no lint script)
6. ✅ Respects `verify_build`, `verify_tests`, `verify_lint` config flags
7. ✅ Clear error messages when verification fails
8. ✅ Integration with auto-fix workflow (apply fix → verify → pass/fail)

## Implementation Checklist

- [ ] Update `internal/verify/detector.go` - Add JavaScript detection
- [ ] Create `internal/verify/javascript_verifier.go` - Implement verifier
- [ ] Update `internal/verify/verify.go` - Add JavaScript case to `RunAll()`
- [ ] Add unit tests: `internal/verify/javascript_verifier_test.go`
- [ ] Test on real JavaScript project (e.g., React app)
- [ ] Update `docs/auto_fix_implementation_plan.md` - Mark JavaScript support complete
- [ ] Update `docs/TESTING_STATUS.md` - Add JavaScript test results

## Edge Cases to Handle

1. **node_modules missing**: Run install before verification
2. **package.json missing**: Return clear error
3. **No build script**: Warn or error (configurable)
4. **Test script is placeholder**: Detect `"test": "echo \"Error: no test specified\" && exit 1"` and skip
5. **TypeScript compilation errors**: Will be caught by build script
6. **ESLint errors**: Will be caught by lint script
7. **Mixed JS/TS project**: Same verifier handles both (TypeScript projects use same npm scripts)

## Future Enhancements (Out of Scope)

- Custom script names (e.g., `build:prod` instead of `build`)
- Parallel build/test/lint execution for speed
- Cache node_modules to avoid repeated installs
- Support for Deno or Bun runtimes
- Framework-specific optimizations (e.g., Angular `ng lint` directly)

## References

- Existing C# verifier: `internal/verify/csharp_verifier.go`
- Existing detector: `internal/verify/detector.go`
- Multi-language architecture: `docs/multi_language_architecture.md`
- Testing status: `docs/TESTING_STATUS.md`

---

**TL;DR**: Add JavaScript/TypeScript support by creating a framework-agnostic verifier that detects the package manager and runs standard npm scripts (`build`, `test`, `lint`). Single verifier handles all JS frameworks (React, Angular, Svelte, etc.).
