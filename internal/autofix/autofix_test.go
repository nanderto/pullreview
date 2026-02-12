package autofix

import (
	"testing"

	"pullreview/internal/llm"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"fixes": []}`,
			expected: `{"fixes": []}`,
		},
		{
			name:     "JSON with json fence",
			input:    "```json\n{\"fixes\": []}\n```",
			expected: `{"fixes": []}`,
		},
		{
			name:     "JSON with plain fence and opening brace",
			input:    "```\n{\"fixes\": []}\n```",
			expected: `{"fixes": []}`,
		},
		{
			name:  "multiline JSON with fence",
			input: "```json\n{\n  \"fixes\": [\n    {\n      \"file\": \"test.go\"\n    }\n  ]\n}\n```",
			expected: `{
  "fixes": [
    {
      "file": "test.go"
    }
  ]
}`,
		},
		{
			name:     "JSON with leading/trailing whitespace",
			input:    "  \n```json\n{\"fixes\": []}\n```\n  ",
			expected: `{"fixes": []}`,
		},
		{
			name:     "plain JSON with whitespace",
			input:    "  {\"fixes\": []}  ",
			expected: `{"fixes": []}`,
		},
		{
			name:     "text before JSON fence",
			input:    "Here is some explanation text.\n\n```json\n{\"fixes\": []}\n```",
			expected: `{"fixes": []}`,
		},
		{
			name:     "LLM response with explanation before fence",
			input:    "Looking at the issue, I need to fix the problem. Let me generate the fix:\n\n```json\n{\"fixes\": [{\"file\": \"test.go\"}]}\n```",
			expected: `{"fixes": [{"file": "test.go"}]}`,
		},
		{
			name:     "raw JSON embedded in text",
			input:    "Here is the fix: {\"fixes\": []} done.",
			expected: `{"fixes": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON() =\n%q\nwant\n%q", result, tt.expected)
			}
		})
	}
}

func TestParseErrorFiles(t *testing.T) {
	cfg := &AutoFixConfig{}
	cfg.SetDefaults()
	llmClient := llm.NewClient("openai", "fake-key", "https://fake.endpoint")
	af := NewAutoFixer(cfg, llmClient, "/tmp/test")

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "Go build errors with file:line:col format",
			input: `# pullreview/cmd/pullreview
# [pullreview/cmd/pullreview]
vet.exe: cmd\pullreview\main.go:180:6: undefined: llm.SetVerbose
cmd\pullreview\main.go:213:4: r.ParseLLMResponse undefined`,
			expected: []string{"cmd/pullreview/main.go"},
		},
		{
			name: "gofmt output with just file paths",
			input: `internal\bitbucket\client.go
internal\config\config.go
internal\llm\client.go`,
			expected: []string{
				"internal/bitbucket/client.go",
				"internal/config/config.go",
				"internal/llm/client.go",
			},
		},
		{
			name: "mixed errors and formatting",
			input: `❌ go vet failed:
# pullreview/cmd/pullreview
cmd\pullreview\main.go:180:6: undefined: llm.SetVerbose
❌ gofmt check failed:
internal\bitbucket\client.go
internal\review\review.go`,
			expected: []string{
				"cmd/pullreview/main.go",
				"internal/bitbucket/client.go",
				"internal/review/review.go",
			},
		},
		{
			name: "Python files",
			input: `test.py:45:10: undefined name 'foo'
src/utils.py:23:1: E302 expected 2 blank lines`,
			expected: []string{"test.py", "src/utils.py"},
		},
		{
			name: "JavaScript/TypeScript files",
			input: `src/app.ts:12:5: error TS2304: Cannot find name 'React'
index.js:8:1: Unexpected token`,
			expected: []string{"src/app.ts", "index.js"},
		},
		{
			name:     "no files in error",
			input:    `Some generic error message without file references`,
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := af.parseErrorFiles(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("parseErrorFiles() returned %d files, want %d\nGot: %v\nWant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Check each file is present (order doesn't matter due to map)
			resultMap := make(map[string]bool)
			for _, file := range result {
				resultMap[file] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("parseErrorFiles() missing expected file: %s\nGot: %v", expected, result)
				}
			}
		})
	}
}

func TestAutoFormatFiles(t *testing.T) {
	// This is more of an integration test - just ensure it doesn't crash
	cfg := &AutoFixConfig{}
	cfg.SetDefaults()
	llmClient := llm.NewClient("openai", "fake-key", "https://fake.endpoint")
	af := NewAutoFixer(cfg, llmClient, t.TempDir())

	// Test with empty list
	err := af.autoFormatFiles([]string{})
	if err != nil {
		t.Errorf("autoFormatFiles([]) failed: %v", err)
	}

	// Test with non-Go files (should skip)
	err = af.autoFormatFiles([]string{"test.txt", "readme.md"})
	if err != nil {
		t.Errorf("autoFormatFiles() with non-Go files failed: %v", err)
	}
}
