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

func TestMatchCommentsToDiff(t *testing.T) {
	diff := `diff --git a/foo.go b/foo.go
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
	files, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff failed: %v", err)
	}

	comments := []Comment{
		// Valid inline comment (line 3 is an addition in foo.go)
		{FilePath: "foo.go", Line: 3, Text: "Valid inline", IsFileLevel: false},
		// Invalid inline comment (line not present)
		{FilePath: "foo.go", Line: 99, Text: "Invalid line", IsFileLevel: false},
		// Valid file-level comment
		{FilePath: "foo.go", Line: 0, Text: "File-level", IsFileLevel: true},
		// Invalid file-level comment (file does not exist)
		{FilePath: "notfound.go", Line: 0, Text: "No such file", IsFileLevel: true},
		// Invalid inline comment (file does not exist)
		{FilePath: "notfound.go", Line: 1, Text: "No such file inline", IsFileLevel: false},
	}

	matched, unmatched := MatchCommentsToDiff(comments, files)

	// Check matched
	if len(matched) != 2 {
		t.Errorf("expected 2 matched comments, got %d", len(matched))
	}
	// Check unmatched
	if len(unmatched) != 3 {
		t.Errorf("expected 3 unmatched comments, got %d", len(unmatched))
	}

	// Check that valid inline and file-level are matched
	foundInline := false
	foundFile := false
	for _, c := range matched {
		if c.FilePath == "foo.go" && c.Line == 3 && !c.IsFileLevel {
			foundInline = true
		}
		if c.FilePath == "foo.go" && c.IsFileLevel {
			foundFile = true
		}
	}
	if !foundInline {
		t.Errorf("expected valid inline comment to be matched")
	}
	if !foundFile {
		t.Errorf("expected valid file-level comment to be matched")
	}

	// Check that unmatched contains the right comments
	for _, c := range unmatched {
		if c.FilePath == "foo.go" && c.Line == 99 {
			// ok
		} else if c.FilePath == "notfound.go" {
			// ok
		} else {
			t.Errorf("unexpected unmatched comment: %+v", c)
		}
	}
}
