package bitbucket

import (
	"strings"
	"testing"
)

func TestTemplatePRTitle(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected string
	}{
		{
			name:     "default template",
			template: "",
			data: map[string]string{
				"pr_id":          "123",
				"original_title": "Fix bugs",
				"issue_count":    "5",
			},
			expected: "🤖 Auto-fixes for PR #123: Fix bugs",
		},
		{
			name:     "custom template",
			template: "Auto-fix PR #{pr_id}",
			data: map[string]string{
				"pr_id": "456",
			},
			expected: "Auto-fix PR #456",
		},
		{
			name:     "all placeholders",
			template: "{pr_id} - {original_title} - {issue_count} issues",
			data: map[string]string{
				"pr_id":          "789",
				"original_title": "Add feature",
				"issue_count":    "3",
			},
			expected: "789 - Add feature - 3 issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TemplatePRTitle(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTemplatePRDescription(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected string
	}{
		{
			name:     "simple replacement",
			template: "PR: {original_pr_id}\nIssues: {issue_count}",
			data: map[string]string{
				"original_pr_id": "123",
				"issue_count":    "5",
			},
			expected: "PR: 123\nIssues: 5",
		},
		{
			name:     "with special chars in title (should escape)",
			template: "Original: {original_title}",
			data: map[string]string{
				"original_title": "Fix *bugs* [important]",
			},
			expected: "Original: Fix \\*bugs\\* \\[important\\]",
		},
		{
			name:     "ai explanation not escaped",
			template: "Summary: {ai_explanation}",
			data: map[string]string{
				"ai_explanation": "Fixed **critical** issues:\n- Item 1",
			},
			expected: "Summary: Fixed **critical** issues:\n- Item 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TemplatePRDescription(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatFileList(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "empty list",
			files:    []string{},
			expected: "No files changed",
		},
		{
			name:     "single file",
			files:    []string{"main.go"},
			expected: "- `main.go`",
		},
		{
			name:     "multiple files",
			files:    []string{"main.go", "util.go", "test.go"},
			expected: "- `main.go`\n- `util.go`\n- `test.go`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileList(tt.files)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "asterisks",
			input:    "*bold* text",
			expected: "\\*bold\\* text",
		},
		{
			name:     "brackets",
			input:    "[link](url)",
			expected: "\\[link\\]\\(url\\)",
		},
		{
			name:     "multiple special chars",
			input:    "Fix *bugs* [#123]",
			expected: "Fix \\*bugs\\* \\[\\#123\\]",
		},
		{
			name:     "backticks",
			input:    "`code` block",
			expected: "\\`code\\` block",
		},
		{
			name:     "underscores",
			input:    "_italic_ text",
			expected: "\\_italic\\_ text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTemplateWithMissingPlaceholders(t *testing.T) {
	template := "PR: {pr_id}, Title: {original_title}, Missing: {foo}"
	data := map[string]string{
		"pr_id":          "123",
		"original_title": "Test",
	}

	result := TemplatePRTitle(template, data)

	// Should replace known placeholders
	if !strings.Contains(result, "PR: 123") {
		t.Error("expected pr_id to be replaced")
	}
	if !strings.Contains(result, "Title: Test") {
		t.Error("expected original_title to be replaced")
	}

	// Missing placeholders should remain as-is
	if !strings.Contains(result, "{foo}") {
		t.Error("expected missing placeholder to remain")
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "passed",
			status:   "passed",
			expected: "✅ passed",
		},
		{
			name:     "failed",
			status:   "failed",
			expected: "❌ failed",
		},
		{
			name:     "skipped",
			status:   "skipped",
			expected: "⏭️ skipped",
		},
		{
			name:     "unknown",
			status:   "unknown",
			expected: "unknown",
		},
		{
			name:     "uppercase passed",
			status:   "PASSED",
			expected: "✅ passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatStatus(tt.status)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultTemplates(t *testing.T) {
	// Ensure defaults are valid
	if DefaultPRTitleTemplate == "" {
		t.Error("DefaultPRTitleTemplate should not be empty")
	}

	if DefaultPRDescriptionTemplate == "" {
		t.Error("DefaultPRDescriptionTemplate should not be empty")
	}

	// Check that default title template contains expected placeholders
	if !strings.Contains(DefaultPRTitleTemplate, "{pr_id}") {
		t.Error("DefaultPRTitleTemplate should contain {pr_id} placeholder")
	}

	// Check that default description template contains expected placeholders
	requiredPlaceholders := []string{
		"{original_pr_id}",
		"{original_pr_link}",
		"{issue_count}",
		"{iteration_count}",
		"{file_list}",
		"{build_status}",
		"{test_status}",
		"{lint_status}",
		"{ai_explanation}",
	}

	for _, placeholder := range requiredPlaceholders {
		if !strings.Contains(DefaultPRDescriptionTemplate, placeholder) {
			t.Errorf("DefaultPRDescriptionTemplate should contain %s placeholder", placeholder)
		}
	}
}
