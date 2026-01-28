package review

import (
	"strings"
	"testing"
)

const sampleDiff = `diff --git a/foo.go b/foo.go
index 1234567..89abcde 100644
--- a/foo.go
+++ b/foo.go
@@ -1,6 +1,7 @@
 package main

-func hello() {
-    println("Hello, world!")
+func hello(name string) {
+    println("Hello,", name)
 }
+
@@ -10,7 +11,8 @@
 func bye() {
-    println("Bye!")
+    println("Goodbye!")
+    println("See you soon!")
 }
`

func TestParseUnifiedDiff_Simple(t *testing.T) {
	files, err := ParseUnifiedDiff(sampleDiff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	file := files[0]
	if file.NewPath != "foo.go" {
		t.Errorf("expected file NewPath 'foo.go', got '%s'", file.NewPath)
	}
	if len(file.Hunks) != 2 {
		t.Errorf("expected 2 hunks, got %d", len(file.Hunks))
	}
	// Check first hunk header
	h0 := file.Hunks[0]
	if !strings.HasPrefix(h0.Header, "@@ -1,6 +1,7 @@") {
		t.Errorf("unexpected hunk header: %s", h0.Header)
	}
	// Check line mapping in first hunk
	adds, dels := 0, 0
	for _, hl := range h0.LineMapping {
		switch hl.Type {
		case AdditionLine:
			adds++
		case DeletionLine:
			dels++
		}
	}
	if adds == 0 || dels == 0 {
		t.Errorf("expected at least one addition and one deletion in first hunk")
	}
}

func TestReview_ParseDiffAndFormatForLLM(t *testing.T) {
	r := NewReview("123", sampleDiff)
	if err := r.ParseDiff(); err != nil {
		t.Fatalf("ParseDiff failed: %v", err)
	}
	out := r.FormatDiffForLLM()
	if !strings.Contains(out, "File: foo.go") {
		t.Errorf("FormatDiffForLLM missing file header")
	}
	if !strings.Contains(out, "+ func hello(name string) {") {
		t.Errorf("FormatDiffForLLM missing addition line")
	}
	if !strings.Contains(out, "- func hello() {") {
		t.Errorf("FormatDiffForLLM missing deletion line")
	}
}

func TestParseUnifiedDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
index 1..2 100644
--- a/a.go
+++ b/a.go
@@ -1 +1,2 @@
-func A() {}
+func A() {}
+func B() {}
diff --git a/b.go b/b.go
index 3..4 100644
--- a/b.go
+++ b/b.go
@@ -1 +1,2 @@
-func X() {}
+func X() {}
+func Y() {}
`
	files, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].NewPath != "a.go" || files[1].NewPath != "b.go" {
		t.Errorf("unexpected file paths: %s, %s", files[0].NewPath, files[1].NewPath)
	}
}

func TestParseUnifiedDiff_Empty(t *testing.T) {
	files, err := ParseUnifiedDiff("")
	if err != nil {
		t.Fatalf("ParseUnifiedDiff failed on empty diff: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(files))
	}
}

func TestParseLLMResponse_InlineAndSummary(t *testing.T) {
	llmResp := "Overall, this PR looks good. See inline comments for details.\n\n" +
		"```inline foo.go:10\nConsider renaming this variable for clarity.\n```\n\n" +
		"```inline bar.go:25\nPossible off-by-one error here.\n```\n"
	r := &Review{}
	r.ParseLLMResponse(llmResp)

	if len(r.Comments) != 2 {
		t.Fatalf("expected 2 inline comments, got %d", len(r.Comments))
	}
	if r.Comments[0].FilePath != "foo.go" || r.Comments[0].Line != 10 {
		t.Errorf("unexpected first inline comment location: %s:%d", r.Comments[0].FilePath, r.Comments[0].Line)
	}
	if !strings.Contains(r.Comments[0].Text, "renaming") {
		t.Errorf("unexpected first inline comment text: %s", r.Comments[0].Text)
	}
	if r.Comments[1].FilePath != "bar.go" || r.Comments[1].Line != 25 {
		t.Errorf("unexpected second inline comment location: %s:%d", r.Comments[1].FilePath, r.Comments[1].Line)
	}
	if !strings.Contains(r.Comments[1].Text, "off-by-one") {
		t.Errorf("unexpected second inline comment text: %s", r.Comments[1].Text)
	}
	if !strings.HasPrefix(r.Summary, "Overall, this PR looks good") {
		t.Errorf("unexpected summary: %s", r.Summary)
	}
}

func TestParseLLMResponse_SummaryOnly(t *testing.T) {
	llmResp := "This PR is well-structured and requires no changes."
	r := &Review{}
	r.ParseLLMResponse(llmResp)
	if len(r.Comments) != 0 {
		t.Errorf("expected 0 inline comments, got %d", len(r.Comments))
	}
	if !strings.Contains(r.Summary, "well-structured") {
		t.Errorf("unexpected summary: %s", r.Summary)
	}
}
func TestParseLLMResponse_InlineOnly(t *testing.T) {
	llmResp := "```inline foo.go:5\nFix the bug here.\n```"
	r := &Review{}
	r.ParseLLMResponse(llmResp)
	if len(r.Comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(r.Comments))
	}
	if r.Comments[0].FilePath != "foo.go" || r.Comments[0].Line != 5 {
		t.Errorf("unexpected inline comment location: %s:%d", r.Comments[0].FilePath, r.Comments[0].Line)
	}
	if !strings.Contains(r.Comments[0].Text, "bug") {
		t.Errorf("unexpected inline comment text: %s", r.Comments[0].Text)
	}
	if r.Summary != "" {
		t.Errorf("expected empty summary, got: %s", r.Summary)
	}
}

func TestParseLLMResponse_NaturalLanguageInlineSingleLine(t *testing.T) {
	llmResp := "internal/bitbucket/client.go Line 42: This line needs better error handling."
	r := &Review{}
	r.ParseLLMResponse(llmResp)
	if len(r.Comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(r.Comments))
	}
	c := r.Comments[0]
	if c.FilePath != "internal/bitbucket/client.go" || c.Line != 42 {
		t.Errorf("unexpected inline comment location: %s:%d", c.FilePath, c.Line)
	}
	if !strings.Contains(c.Text, "better error handling") {
		t.Errorf("unexpected inline comment text: %s", c.Text)
	}
}

func TestParseLLMResponse_NaturalLanguageInlineMultiLine(t *testing.T) {
	llmResp := "internal/llm/client.go Lines 10-12: Consider refactoring this block for clarity."
	r := &Review{}
	r.ParseLLMResponse(llmResp)
	if len(r.Comments) != 3 {
		t.Fatalf("expected 3 inline comments, got %d", len(r.Comments))
	}
	for i, c := range r.Comments {
		if c.FilePath != "internal/llm/client.go" || c.Line != 10+i {
			t.Errorf("unexpected inline comment location: %s:%d", c.FilePath, c.Line)
		}
		if !strings.Contains(c.Text, "refactoring this block") {
			t.Errorf("unexpected inline comment text: %s", c.Text)
		}
	}
}

func TestParseLLMResponse_NaturalLanguageAndSummary(t *testing.T) {
	llmResp := "General feedback: Good work overall.\n\ninternal/utils/utils.go Line 7: Use a more descriptive variable name.\n\nThank you!"
	r := &Review{}
	r.ParseLLMResponse(llmResp)
	if len(r.Comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(r.Comments))
	}
	c := r.Comments[0]
	if c.FilePath != "internal/utils/utils.go" || c.Line != 7 {
		t.Errorf("unexpected inline comment location: %s:%d", c.FilePath, c.Line)
	}
	if !strings.Contains(c.Text, "descriptive variable name") {
		t.Errorf("unexpected inline comment text: %s", c.Text)
	}
	if !strings.Contains(r.Summary, "General feedback") || !strings.Contains(r.Summary, "Thank you!") {
		t.Errorf("unexpected summary: %s", r.Summary)
	}
}
