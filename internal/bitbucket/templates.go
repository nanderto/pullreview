package bitbucket

import (
	"fmt"
	"strings"
)

// Default templates for PR title and description.
const (
	DefaultPRTitleTemplate = "🤖 Auto-fixes for PR #{pr_id}: {original_title}"

	DefaultPRDescriptionTemplate = `## Auto-generated fixes for PR #{original_pr_id}

**Original PR:** {original_pr_link}  
**Issues Fixed:** {issue_count}  
**Iterations Required:** {iteration_count}

### Changes Made:
{file_list}

### Build Verification:
- Build: {build_status}
- Tests: {test_status}
- Lint: {lint_status}

### AI Summary:
{ai_explanation}

---

*This PR was automatically created by pullreview. Please review the changes before merging.*`
)

// TemplatePRTitle generates a PR title from a template.
// Supports placeholders:
// - {pr_id} - Original PR ID
// - {original_title} - Original PR title
// - {issue_count} - Number of issues fixed
func TemplatePRTitle(template string, data map[string]string) string {
	if template == "" {
		template = DefaultPRTitleTemplate
	}

	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// TemplatePRDescription generates a PR description from a template.
// Supports placeholders:
// - {original_pr_id} - Original PR ID
// - {original_pr_link} - Link to original PR
// - {issue_count} - Number of issues fixed
// - {iteration_count} - Number of iterations required
// - {file_list} - Markdown list of changed files
// - {build_status} - passed/failed
// - {test_status} - passed/failed
// - {lint_status} - passed/failed
// - {ai_explanation} - LLM's summary of fixes
func TemplatePRDescription(template string, data map[string]string) string {
	if template == "" {
		template = DefaultPRDescriptionTemplate
	}

	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{%s}", key)
		// Escape markdown in user-provided values for certain fields
		if shouldEscapeField(key) {
			value = EscapeMarkdown(value)
		}
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// shouldEscapeField determines if a field should have markdown escaped.
// We don't escape fields like ai_explanation that are meant to contain markdown.
func shouldEscapeField(fieldName string) bool {
	noEscapeFields := map[string]bool{
		"ai_explanation": true,
		"file_list":      true,
		"build_status":   true,
		"test_status":    true,
		"lint_status":    true,
	}
	return !noEscapeFields[fieldName]
}

// FormatFileList creates a markdown list of changed files.
func FormatFileList(files []string) string {
	if len(files) == 0 {
		return "No files changed"
	}

	var items []string
	for _, file := range files {
		items = append(items, fmt.Sprintf("- `%s`", file))
	}

	return strings.Join(items, "\n")
}

// EscapeMarkdown escapes special markdown characters in user input.
// This prevents markdown injection while preserving readability.
func EscapeMarkdown(text string) string {
	// Characters that have special meaning in markdown
	replacements := map[string]string{
		"\\": "\\\\", // Backslash first
		"*":  "\\*",
		"_":  "\\_",
		"[":  "\\[",
		"]":  "\\]",
		"(":  "\\(",
		")":  "\\)",
		"#":  "\\#",
		"+":  "\\+",
		"-":  "\\-",
		".":  "\\.",
		"!":  "\\!",
		"`":  "\\`",
		"|":  "\\|",
	}

	result := text
	// Apply backslash escaping first
	result = strings.ReplaceAll(result, "\\", "\\\\")

	// Then escape other special characters
	for char, escaped := range replacements {
		if char != "\\" { // Skip backslash as we already handled it
			result = strings.ReplaceAll(result, char, escaped)
		}
	}

	return result
}

// FormatStatus formats a verification status with emoji.
func FormatStatus(status string) string {
	switch strings.ToLower(status) {
	case "passed":
		return "✅ passed"
	case "failed":
		return "❌ failed"
	case "skipped":
		return "⏭️ skipped"
	default:
		return status
	}
}
