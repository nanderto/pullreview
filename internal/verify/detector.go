package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// languageInfo tracks detection data for a language.
type languageInfo struct {
	name          string
	fileCount     int
	hasConfigFile bool
	hasDirMarker  bool
}

// DetectLanguages scans the repository and returns a list of detected languages.
// Returns languages in priority order (most prevalent first).
func DetectLanguages(repoPath string) ([]string, error) {
	// Validate repo path
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}

	info, err := os.Stat(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access repository path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repository path is not a directory: %s", repoPath)
	}

	// Initialize language tracking
	languages := map[string]*languageInfo{
		"go":         {name: "go"},
		"python":     {name: "python"},
		"javascript": {name: "javascript"},
		"typescript": {name: "typescript"},
		"java":       {name: "java"},
		"rust":       {name: "rust"},
		"ruby":       {name: "ruby"},
		"php":        {name: "php"},
		"csharp":     {name: "csharp"},
	}

	// Walk the repository
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		// Get relative path for ignore logic
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			relPath = path
		}

		// Check if we should ignore this path
		if shouldIgnore(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for config files
		if !info.IsDir() {
			checkConfigFile(filepath.Base(path), path, languages)

			// Count file extensions
			ext := strings.ToLower(filepath.Ext(path))
			countFileExtension(ext, languages)
		} else {
			// Check for directory markers
			checkDirMarker(filepath.Base(path), languages)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk repository: %w", err)
	}

	// Filter and sort languages
	result := filterAndSortLanguages(languages)

	if len(result) == 0 {
		return nil, fmt.Errorf("no recognized programming languages detected in repository")
	}

	return result, nil
}

// shouldIgnore checks if a path should be ignored during scanning.
func shouldIgnore(relPath string, info os.FileInfo) bool {
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Directories to ignore
	ignoreDirs := []string{
		"vendor/",
		"node_modules/",
		".git/",
		"dist/",
		"build/",
		"__pycache__/",
		".venv/",
		"venv/",
		"target/",
	}

	if info.IsDir() {
		dirName := filepath.Base(relPath) + "/"
		for _, ignore := range ignoreDirs {
			if strings.HasSuffix(ignore, "/") && dirName == ignore {
				return true
			}
		}
	}

	// Check if path contains any ignore directory
	for _, ignore := range ignoreDirs {
		if strings.Contains(relPath, ignore) {
			return true
		}
	}

	return false
}

// checkConfigFile checks if a file is a language configuration file.
func checkConfigFile(filename string, fullPath string, languages map[string]*languageInfo) {
	switch filename {
	case "go.mod", "go.sum":
		languages["go"].hasConfigFile = true
	case "pyproject.toml", "setup.py", "requirements.txt", "Pipfile":
		languages["python"].hasConfigFile = true
	case "package.json":
		languages["javascript"].hasConfigFile = true
		// Check if package.json contains TypeScript
		if hasTypeScriptDependency(fullPath) {
			languages["typescript"].hasConfigFile = true
		}
	case "tsconfig.json":
		languages["typescript"].hasConfigFile = true
	case "pom.xml", "build.gradle":
		languages["java"].hasConfigFile = true
	case "Cargo.toml":
		languages["rust"].hasConfigFile = true
	case "Gemfile":
		languages["ruby"].hasConfigFile = true
	case "composer.json":
		languages["php"].hasConfigFile = true
	}
	
	// Check for C# project files (case-insensitive)
	if strings.HasSuffix(strings.ToLower(filename), ".csproj") ||
		strings.HasSuffix(strings.ToLower(filename), ".sln") {
		languages["csharp"].hasConfigFile = true
	}
}

// hasTypeScriptDependency checks if package.json contains TypeScript.
func hasTypeScriptDependency(packageJsonPath string) bool {
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return false
	}

	contentStr := string(content)
	return strings.Contains(contentStr, "\"typescript\"")
}

// countFileExtension counts occurrences of file extensions.
func countFileExtension(ext string, languages map[string]*languageInfo) {
	switch ext {
	case ".go":
		languages["go"].fileCount++
	case ".py":
		languages["python"].fileCount++
	case ".js", ".jsx":
		languages["javascript"].fileCount++
	case ".ts", ".tsx":
		languages["typescript"].fileCount++
	case ".java":
		languages["java"].fileCount++
	case ".rs":
		languages["rust"].fileCount++
	case ".rb":
		languages["ruby"].fileCount++
	case ".php":
		languages["php"].fileCount++
	case ".cs":
		languages["csharp"].fileCount++
	}
}

// checkDirMarker checks for directory markers that indicate a language.
func checkDirMarker(dirName string, languages map[string]*languageInfo) {
	switch dirName {
	case "node_modules":
		languages["javascript"].hasDirMarker = true
		languages["typescript"].hasDirMarker = true
	case "venv", "__pycache__", ".venv":
		languages["python"].hasDirMarker = true
	case "target":
		languages["java"].hasDirMarker = true
	}
}

// filterAndSortLanguages filters languages based on threshold and sorts by prevalence.
func filterAndSortLanguages(languages map[string]*languageInfo) []string {
	const minFileThreshold = 5

	var detected []*languageInfo

	for _, lang := range languages {
		// Include if config file found OR meets file count threshold
		if lang.hasConfigFile || lang.fileCount >= minFileThreshold {
			detected = append(detected, lang)
		}
	}

	// Sort by priority:
	// 1. Config file presence (highest priority)
	// 2. File count (most files first)
	sort.Slice(detected, func(i, j int) bool {
		// If one has config and other doesn't, prefer the one with config
		if detected[i].hasConfigFile != detected[j].hasConfigFile {
			return detected[i].hasConfigFile
		}
		// Otherwise sort by file count
		return detected[i].fileCount > detected[j].fileCount
	})

	// Extract language names
	result := make([]string, len(detected))
	for i, lang := range detected {
		result[i] = lang.name
	}

	return result
}
