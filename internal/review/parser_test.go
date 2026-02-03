package review

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type Expectation struct {
	Type    string // "inline", "file", "summary"
	File    string
	Line    int
	Comment string
}

func parseExpectations(section string) ([]Expectation, error) {
	var exps []Expectation
	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// inline: file=internal/bitbucket/client.go line=23 comment=This is great
		// file: file=internal/bitbucket/client.go comment=This is a file comment
		// summary: THis is the summary
		if strings.HasPrefix(line, "inline:") {
			exp := Expectation{Type: "inline"}
			parts := strings.Fields(line[len("inline:"):])
			for _, part := range parts {
				if strings.HasPrefix(part, "file=") {
					exp.File = strings.TrimPrefix(part, "file=")
				} else if strings.HasPrefix(part, "line=") {
					fmtSscanf(part, "line=%d", &exp.Line)
				} else if strings.HasPrefix(part, "comment=") {
					exp.Comment = strings.TrimPrefix(part, "comment=")
					// If comment contains spaces, join the rest
					idx := strings.Index(line, "comment=")
					if idx != -1 {
						exp.Comment = strings.TrimSpace(line[idx+len("comment="):])
						break
					}
				}
			}
			exps = append(exps, exp)
		} else if strings.HasPrefix(line, "file:") {
			exp := Expectation{Type: "file"}
			parts := strings.Fields(line[len("file:"):])
			for _, part := range parts {
				if strings.HasPrefix(part, "file=") {
					exp.File = strings.TrimPrefix(part, "file=")
				} else if strings.HasPrefix(part, "comment=") {
					exp.Comment = strings.TrimPrefix(part, "comment=")
					idx := strings.Index(line, "comment=")
					if idx != -1 {
						exp.Comment = strings.TrimSpace(line[idx+len("comment="):])
						break
					}
				}
			}
			exps = append(exps, exp)
		} else if strings.HasPrefix(line, "summary:") {
			exp := Expectation{Type: "summary"}
			exp.Comment = strings.TrimSpace(line[len("summary:"):])
			exps = append(exps, exp)
		}
	}
	return exps, scanner.Err()
}

// fmtSscanf is a helper for parsing ints without importing fmt just for Sscanf.
func fmtSscanf(s string, format string, dest *int) {
	// expects format "line=%d"
	if strings.HasPrefix(format, "line=%d") && strings.HasPrefix(s, "line=") {
		val := s[len("line="):]
		*dest = 0
		for i := 0; i < len(val) && val[i] >= '0' && val[i] <= '9'; i++ {
			*dest = *dest*10 + int(val[i]-'0')
		}
	}
}

func normalizeContentWordsOnly(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	// Remove markdown headers and formatting
	md := regexp.MustCompile(`[*_#>` + "`" + `]+|^\s*[-+*]\s+`)
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		// Remove markdown formatting from each line
		clean := md.ReplaceAllString(line, "")
		out = append(out, clean)
	}
	joined := strings.Join(out, " ")
	// Collapse all whitespace to a single space
	spaceCollapse := regexp.MustCompile(`\s+`)
	return spaceCollapse.ReplaceAllString(strings.TrimSpace(joined), " ")
}

func checkExpectations(t *testing.T, exps []Expectation, comments []Comment, summary string) {
	t.Helper()
	// Inline and file-level comments
	var gotInline []Expectation
	var gotFile []Expectation
	for _, c := range comments {
		if c.IsFileLevel {
			gotFile = append(gotFile, Expectation{
				Type:    "file",
				File:    c.FilePath,
				Comment: strings.TrimSpace(c.Text),
			})
		} else {
			gotInline = append(gotInline, Expectation{
				Type:    "inline",
				File:    c.FilePath,
				Line:    c.Line,
				Comment: strings.TrimSpace(c.Text),
			})
		}
	}
	// Compare inline
	var wantInline, wantFile []Expectation
	var wantSummary string
	for _, e := range exps {
		switch e.Type {
		case "inline":
			wantInline = append(wantInline, e)
		case "file":
			wantFile = append(wantFile, e)
		case "summary":
			wantSummary = e.Comment
		}
	}
	// Inline comments
	if len(gotInline) != len(wantInline) {
		t.Errorf("expected %d inline comments, got %d", len(wantInline), len(gotInline))
	}
	for _, want := range wantInline {
		found := false
		for _, got := range gotInline {
			if got.File == want.File && got.Line == want.Line && got.Comment == want.Comment {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected inline comment: file=%s line=%d comment=%q", want.File, want.Line, want.Comment)
		}
	}
	// File-level comments (normalize words only)
	if len(gotFile) != len(wantFile) {
		t.Errorf("expected %d file-level comments, got %d", len(wantFile), len(gotFile))
	}
	for _, want := range wantFile {
		found := false
		wantNorm := normalizeContentWordsOnly(want.Comment)
		for _, got := range gotFile {
			gotNorm := normalizeContentWordsOnly(got.Comment)
			if got.File == want.File && gotNorm == wantNorm {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected file-level comment: file=%s comment=%q", want.File, want.Comment)
		}
	}
	// Summary (normalize words only)
	wantSummaryNorm := normalizeContentWordsOnly(wantSummary)
	gotSummaryNorm := normalizeContentWordsOnly(summary)
	if wantSummaryNorm != "" && wantSummaryNorm != gotSummaryNorm {
		t.Errorf("expected summary %q, got %q", wantSummaryNorm, gotSummaryNorm)
		t.Logf("DEBUG: wantSummaryNorm: %q", wantSummaryNorm)
		t.Logf("DEBUG: gotSummaryNorm: %q", gotSummaryNorm)
	}
}

func TestLLMResponseParsingFromTestFiles(t *testing.T) {
	files, err := filepath.Glob("testdata/llm_output_*.txt")
	if err != nil {
		t.Fatalf("failed to glob testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no testdata files found")
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", file, err)
			}
			parts := strings.SplitN(string(data), "***Raw*Seperator***", 2)
			if len(parts) != 2 {
				t.Fatalf("file %s missing ***Raw*Seperator***", file)
			}
			exps, err := parseExpectations(parts[0])
			if err != nil {
				t.Fatalf("failed to parse expectations: %v", err)
			}
			raw := strings.TrimSpace(parts[1])
			comments, summary := ParseLLMResponse(raw)
			// DEBUG: Print all extracted inline comments for README.md
			for _, c := range comments {
				if c.FilePath == "README.md" && c.Line > 0 {
					t.Logf("[DEBUG] Extracted inline comment for README.md line %d: %q", c.Line, c.Text)
				}
			}
			checkExpectations(t, exps, comments, summary)
		})
	}
}
