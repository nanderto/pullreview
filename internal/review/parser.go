package review

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

func ParseLLMResponse(llmResp string) ([]Comment, string) {

	var comments []Comment

	var summary string

	sections := splitSectionsNewFormat(llmResp)

	// Parse inline comments
	if inline, ok := sections["INLINE COMMENTS"]; ok {

		comments = append(comments, parseExplicitInlineComments(inline)...)

	}
	// Parse file-level comments
	if filelevel, ok := sections["FILE-LEVEL COMMENTS"]; ok {

		comments = append(comments, parseExplicitFileLevelComments(filelevel)...)

	}
	// Parse summary
	if summ, ok := sections["SUMMARY"]; ok {

		summary = parseExplicitSummary(summ)

	}

	return comments, summary
}

func splitSectionsNewFormat(llmResp string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(llmResp, "\n")
	var currentSection string
	var currentContent []string

	// Relaxed regex: match any number of asterisks, 'SECTION:', capture anything up to next asterisk, then any number of asterisks
	sectionHeaderRe := regexp.MustCompile(`^\*+\s*SECTION:\s*([^*]+?)\s*\*+$`)
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if m := sectionHeaderRe.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != "" {
				sections[strings.ToUpper(strings.TrimSpace(currentSection))] = strings.TrimSpace(strings.Join(currentContent, "\n"))
			}
			currentSection = strings.TrimSpace(m[1])
			currentContent = []string{}
		} else if currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}
	if currentSection != "" {
		sections[strings.ToUpper(strings.TrimSpace(currentSection))] = strings.TrimSpace(strings.Join(currentContent, "\n"))
	}
	return sections
}

func parseExplicitInlineComments(content string) []Comment {
	var comments []Comment
	scanner := bufio.NewScanner(strings.NewReader(content))
	var file string
	var line int
	var comment string
	for scanner.Scan() {
		txt := strings.TrimSpace(scanner.Text())
		if txt == "" {
			if file != "" && line > 0 && comment != "" {
				comments = append(comments, Comment{
					FilePath: file,
					Line:     line,
					Text:     comment,
				})
			}
			file, line, comment = "", 0, ""
			continue
		}
		if strings.HasPrefix(txt, "FILE:") {
			file = strings.TrimSpace(txt[len("FILE:"):])
		} else if strings.HasPrefix(txt, "LINE:") {
			lineStr := strings.TrimSpace(txt[len("LINE:"):])
			line, _ = strconv.Atoi(lineStr)
		} else if strings.HasPrefix(txt, "COMMENT:") {
			comment = strings.TrimSpace(txt[len("COMMENT:"):])
		}
	}
	// Handle last block if not followed by blank line
	if file != "" && line > 0 && comment != "" {
		comments = append(comments, Comment{
			FilePath: file,
			Line:     line,
			Text:     comment,
		})
	}
	return comments
}

func parseExplicitFileLevelComments(content string) []Comment {
	var comments []Comment
	scanner := bufio.NewScanner(strings.NewReader(content))
	var file string
	var comment string
	for scanner.Scan() {
		txt := strings.TrimSpace(scanner.Text())
		if txt == "" {
			if file != "" && comment != "" {
				comments = append(comments, Comment{
					FilePath:    file,
					Line:        0,
					Text:        comment,
					IsFileLevel: true,
				})
			}
			file, comment = "", ""
			continue
		}
		if strings.HasPrefix(txt, "FILE:") {
			file = strings.TrimSpace(txt[len("FILE:"):])
		} else if strings.HasPrefix(txt, "COMMENT:") {
			comment = strings.TrimSpace(txt[len("COMMENT:"):])
		}
	}
	// Handle last block if not followed by blank line
	if file != "" && comment != "" {
		comments = append(comments, Comment{
			FilePath:    file,
			Line:        0,
			Text:        comment,
			IsFileLevel: true,
		})
	}
	return comments
}

func parseExplicitSummary(content string) string {
	// The summary section is just the text content, possibly with blank lines.
	// We'll trim leading/trailing whitespace and collapse multiple blank lines to a single space.
	// Ignore any section header lines (e.g., END marker).
	lines := strings.Split(content, "\n")
	var filtered []string
	sectionHeaderRe := regexp.MustCompile(`^\*+\s*[A-Z: ]+\s*\*+$`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if sectionHeaderRe.MatchString(trimmed) {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return strings.Join(filtered, " ")
}
