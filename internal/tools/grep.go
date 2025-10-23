package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GrepTool struct{}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Powerful search tool for finding patterns in files. Supports regex, file filtering, and multiple output modes."
}

func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regular expression pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory to search in (defaults to current directory)",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g. '*.js', '*.{ts,tsx}')",
			},
			"output_mode": map[string]interface{}{
				"type":        "string",
				"description": "Output mode: 'content' (show lines), 'files_with_matches' (show files), 'count' (show counts)",
				"enum":        []string{"content", "files_with_matches", "count"},
			},
			"case_insensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Case insensitive search",
			},
			"line_numbers": map[string]interface{}{
				"type":        "boolean",
				"description": "Show line numbers (only for content mode)",
			},
			"head_limit": map[string]interface{}{
				"type":        "number",
				"description": "Limit output to first N results",
			},
		},
		"required": []string{"pattern"},
	}
}

type GrepArgs struct {
	Pattern         string `json:"pattern"`
	Path            string `json:"path,omitempty"`
	Glob            string `json:"glob,omitempty"`
	OutputMode      string `json:"output_mode,omitempty"`
	CaseInsensitive bool   `json:"case_insensitive,omitempty"`
	LineNumbers     bool   `json:"line_numbers,omitempty"`
	HeadLimit       int    `json:"head_limit,omitempty"`
}

func (t *GrepTool) Execute(ctx context.Context, args string) (string, error) {
	var grepArgs GrepArgs
	if err := UnmarshalArgs(args, &grepArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Default output mode
	if grepArgs.OutputMode == "" {
		grepArgs.OutputMode = "files_with_matches"
	}

	searchPath := grepArgs.Path
	if searchPath == "" {
		var err error
		searchPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Compile regex
	flags := ""
	if grepArgs.CaseInsensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + grepArgs.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var result strings.Builder
	count := 0
	limitReached := false

	// Search files
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Apply glob filter if specified
		if grepArgs.Glob != "" {
			matched, _ := filepath.Match(grepArgs.Glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		fileHasMatch := false
		matchCount := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			if re.MatchString(line) {
				matchCount++
				fileHasMatch = true

				if grepArgs.OutputMode == "content" {
					if grepArgs.LineNumbers {
						result.WriteString(fmt.Sprintf("%s:%d:%s\n", path, lineNum, line))
					} else {
						result.WriteString(fmt.Sprintf("%s:%s\n", path, line))
					}
					count++
					if grepArgs.HeadLimit > 0 && count >= grepArgs.HeadLimit {
						limitReached = true
						return filepath.SkipDir
					}
				}
			}
		}

		if fileHasMatch {
			if grepArgs.OutputMode == "files_with_matches" {
				result.WriteString(path + "\n")
				count++
				if grepArgs.HeadLimit > 0 && count >= grepArgs.HeadLimit {
					limitReached = true
					return filepath.SkipDir
				}
			} else if grepArgs.OutputMode == "count" {
				result.WriteString(fmt.Sprintf("%s:%d\n", path, matchCount))
				count++
				if grepArgs.HeadLimit > 0 && count >= grepArgs.HeadLimit {
					limitReached = true
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil && !limitReached {
		return "", fmt.Errorf("search failed: %w", err)
	}

	output := strings.TrimSpace(result.String())
	if output == "" {
		return "No matches found", nil
	}

	return output, nil
}
