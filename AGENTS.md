# agents.md

## Project Overview
This project is a Go (Golang) command-line application.

The goal is to produce:
- A fast, small, cross-platform CLI
- Idiomatic, readable Go code
- Predictable command behavior and output
- Clear error handling and helpful UX

AI agents should prioritize correctness, simplicity, and Go best practices.

---

## Tech Stack
- Language: Go (latest stable)
- CLI framework: `cobra` (preferred) or standard `flag` package if simpler
- Build system: `go build`
- Dependency management: Go modules
- Target platforms: macOS, Linux, Windows

---

## Project Structure

pullreview/
├── cmd/
│   └── pullreview/
│       └── main.go
├── internal/
│   ├── bitbucket/
│   │   └── client.go
│   ├── llm/
│   │   └── client.go
│   ├── review/
│   │   └── review.go
│   ├── config/
│   │   └── config.go
│   └── utils/
│       └── utils.go
├── prompt.md
├── pullreview.yaml
├── go.mod
└── README.md


Rules:
- `cmd/` contains CLI entrypoints only
- Business logic lives in `/internal`
- Avoid putting logic in `main.go`
- Keep packages small and focused

---

## Coding Guidelines
- Follow idiomatic Go style
- Run `gofmt` on all files
- Prefer explicit, readable code over cleverness
- Avoid global state
- Keep functions short and focused
- Use meaningful variable and function names

Do NOT:
- Over-engineer abstractions
- Introduce unnecessary interfaces
- Use reflection unless required

---

## Error Handling
- Always return errors instead of panicking
- Wrap errors with context using `fmt.Errorf("...: %w", err)`
- CLI-facing errors should be human-readable
- Exit codes must be meaningful:
  - `0` for success
  - `1` for general failure
  - Avoid silent failures

---

## CLI Behavior
- Commands should be predictable and composable
- Prefer flags over positional arguments when ambiguity exists
- Support `--help` and `--version`
- Output should be:
  - Human-friendly by default
  - Machine-friendly only when explicitly requested (e.g. `--json`)

---

## Logging & Output
- Do not log to stdout unless it’s intentional output
- Use stderr for errors and logs
- Avoid verbose logging unless `--verbose` or `--debug` is enabled

---

## Testing
- Use Go’s standard `testing` package
- Write unit tests for:
  - Core business logic
  - Command behavior where practical
- Avoid testing CLI frameworks themselves
- Tests should be deterministic and fast

---

## Dependencies
- Keep dependencies minimal
- Prefer Go standard library
- Every dependency must justify its existence

---

## Documentation
- Exported functions must have comments
- CLI commands and flags must be self-documented via help text
- Keep README focused on usage, not implementation details

---

## AI Agent Instructions
When modifying or generating code:
1. Preserve existing behavior unless explicitly told otherwise
2. Match existing style and patterns
3. Ask before large refactors
4. Prefer small, incremental changes
5. Ensure code builds and passes tests
6. Complete Linting
7. Ensure code is formatted correctly
8. Ensure code is secure and follows best practices
9. Ensure code is well-documented and easy to understand
10. Ensure code is tested and covered by unit tests

When in doubt, choose the simplest working solution.
