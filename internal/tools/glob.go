package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GlobTool struct{}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Fast file pattern matching tool. Supports glob patterns like '**/*.js' or 'src/**/*.ts'. Returns matching file paths sorted by modification time."
}

func (t *GlobTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in (defaults to current working directory)",
			},
		},
		"required": []string{"pattern"},
	}
}

type GlobArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

type fileWithTime struct {
	path    string
	modTime time.Time
}

func (t *GlobTool) Execute(ctx context.Context, args string) (string, error) {
	var globArgs GlobArgs
	if err := UnmarshalArgs(args, &globArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	searchPath := globArgs.Path
	if searchPath == "" {
		var err error
		searchPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Handle ** patterns by walking the directory tree
	var matches []fileWithTime
	if strings.Contains(globArgs.Pattern, "**") {
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(searchPath, path)
			matched, _ := filepath.Match(strings.ReplaceAll(globArgs.Pattern, "**", "*"), relPath)
			if !matched {
				// Try more sophisticated matching
				matched = matchPattern(globArgs.Pattern, relPath)
			}

			if matched {
				matches = append(matches, fileWithTime{
					path:    path,
					modTime: info.ModTime(),
				})
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		// Simple glob pattern
		fullPattern := filepath.Join(searchPath, globArgs.Pattern)
		paths, err := filepath.Glob(fullPattern)
		if err != nil {
			return "", fmt.Errorf("glob failed: %w", err)
		}

		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if !info.IsDir() {
				matches = append(matches, fileWithTime{
					path:    path,
					modTime: info.ModTime(),
				})
			}
		}
	}

	if len(matches) == 0 {
		return "No files found", nil
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].modTime.After(matches[i].modTime) {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	var result strings.Builder
	for _, match := range matches {
		result.WriteString(match.path)
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String()), nil
}

// Simple pattern matching for ** wildcards
func matchPattern(pattern, path string) bool {
	pattern = strings.ReplaceAll(pattern, "**", ".*")
	pattern = strings.ReplaceAll(pattern, "*", "[^/\\\\]*")
	pattern = "^" + pattern + "$"

	// Simple regex-like matching
	return strings.Contains(path, strings.ReplaceAll(pattern, ".*", ""))
}
