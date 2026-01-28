package review

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Review encapsulates the logic for preparing and posting code review comments.
type Review struct {
	PRID     string
	Diff     string
	Comments []Comment
	Summary  string

	Files []*DiffFile // Parsed diff files
}

// Comment represents an inline or file-level comment to be posted on a PR.
type Comment struct {
	FilePath string
	Line     int
	Text     string
}

// DiffFile represents a file changed in the diff, with its hunks.
type DiffFile struct {
	OldPath string
	NewPath string
	Hunks   []*DiffHunk
}

// DiffHunk represents a hunk in the diff (a contiguous block of changes).
type DiffHunk struct {
	Header      string // The @@ ... @@ header line
	OldStart    int
	OldLines    int
	NewStart    int
	NewLines    int
	Lines       []string   // All lines in the hunk, including context, additions, deletions
	LineMapping []HunkLine // Mapping of diff lines to new file line numbers
}

// HunkLine maps a line in the diff to its type and line number in the new file.
type HunkLine struct {
	Type    LineType
	Content string
	OldLine int // 0 if not present
	NewLine int // 0 if not present
}

// LineType indicates if a line is context, addition, or deletion.
type LineType int

const (
	ContextLine LineType = iota
	AdditionLine
	DeletionLine
)

// NewReview creates a new Review instance.
func NewReview(prID, diff string) *Review {
	return &Review{
		PRID: prID,
		Diff: diff,
	}
}

// ParseDiff parses the unified diff and populates the Files field.
func (r *Review) ParseDiff() error {
	files, err := ParseUnifiedDiff(r.Diff)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}
	r.Files = files
	return nil
}

// ParseUnifiedDiff parses a unified diff string into a slice of DiffFile.
func ParseUnifiedDiff(diff string) ([]*DiffFile, error) {
	var files []*DiffFile
	var currentFile *DiffFile
	var currentHunk *DiffHunk

	lines := strings.Split(diff, "\n")
	fileHeaderRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if matches := fileHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Start of a new file diff
			if currentFile != nil {
				// Save previous file
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, currentHunk)
					currentHunk = nil
				}
				files = append(files, currentFile)
			}
			currentFile = &DiffFile{
				OldPath: matches[1],
				NewPath: matches[2],
			}
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			// Start of a new hunk
			if currentHunk != nil && currentFile != nil {
				currentFile.Hunks = append(currentFile.Hunks, currentHunk)
			}
			if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
				oldStart, _ := strconv.Atoi(matches[1])
				oldLines := 1
				if matches[2] != "" {
					oldLines, _ = strconv.Atoi(matches[2])
				}
				newStart, _ := strconv.Atoi(matches[3])
				newLines := 1
				if matches[4] != "" {
					newLines, _ = strconv.Atoi(matches[4])
				}
				currentHunk = &DiffHunk{
					Header:      line,
					OldStart:    oldStart,
					OldLines:    oldLines,
					NewStart:    newStart,
					NewLines:    newLines,
					Lines:       []string{},
					LineMapping: []HunkLine{},
				}
				// Parse hunk lines
				oldLineNum := oldStart
				newLineNum := newStart
				for j := i + 1; j < len(lines); j++ {
					hunkLine := lines[j]
					if strings.HasPrefix(hunkLine, "diff --git ") || strings.HasPrefix(hunkLine, "@@ ") {
						// End of hunk
						i = j - 1
						break
					}
					currentHunk.Lines = append(currentHunk.Lines, hunkLine)
					switch {
					case strings.HasPrefix(hunkLine, "+"):
						currentHunk.LineMapping = append(currentHunk.LineMapping, HunkLine{
							Type:    AdditionLine,
							Content: hunkLine,
							OldLine: 0,
							NewLine: newLineNum,
						})
						newLineNum++
					case strings.HasPrefix(hunkLine, "-"):
						currentHunk.LineMapping = append(currentHunk.LineMapping, HunkLine{
							Type:    DeletionLine,
							Content: hunkLine,
							OldLine: oldLineNum,
							NewLine: 0,
						})
						oldLineNum++
					default:
						currentHunk.LineMapping = append(currentHunk.LineMapping, HunkLine{
							Type:    ContextLine,
							Content: hunkLine,
							OldLine: oldLineNum,
							NewLine: newLineNum,
						})
						oldLineNum++
						newLineNum++
					}
				}
			}
			continue
		}
	}
	// Add last file/hunk if present
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, currentHunk)
		}
		files = append(files, currentFile)
	}
	return files, nil
}

// FormatDiffForLLM returns a string representation of the parsed diff with clear file and hunk context for LLM input.
func (r *Review) FormatDiffForLLM() string {
	if len(r.Files) == 0 {
		return r.Diff
	}
	var sb strings.Builder
	for _, f := range r.Files {
		sb.WriteString(fmt.Sprintf("File: %s\n", f.NewPath))
		for _, h := range f.Hunks {
			sb.WriteString(fmt.Sprintf("  %s\n", h.Header))
			for _, hl := range h.LineMapping {
				switch hl.Type {
				case AdditionLine:
					sb.WriteString(fmt.Sprintf("    + %s\n", strings.TrimPrefix(hl.Content, "+")))
				case DeletionLine:
					sb.WriteString(fmt.Sprintf("    - %s\n", strings.TrimPrefix(hl.Content, "-")))
				default:
					sb.WriteString(fmt.Sprintf("      %s\n", hl.Content))
				}
			}
		}
	}
	return sb.String()
}
