package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type ReadTool struct{}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return "Reads a file from the filesystem. Returns file contents with line numbers."
}

func (t *ReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "number",
				"description": "The line number to start reading from (optional)",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "The number of lines to read (optional)",
			},
		},
		"required": []string{"file_path"},
	}
}

type ReadArgs struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func (t *ReadTool) Execute(ctx context.Context, args string) (string, error) {
	var readArgs ReadArgs
	if err := UnmarshalArgs(args, &readArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	file, err := os.Open(readArgs.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var result strings.Builder
	lineNum := 1

	// Default limit to 2000 lines if not specified
	limit := readArgs.Limit
	if limit == 0 {
		limit = 2000
	}

	offset := readArgs.Offset
	if offset == 0 {
		offset = 1
	}

	for scanner.Scan() {
		if lineNum >= offset && lineNum < offset+limit {
			line := scanner.Text()
			// Truncate lines longer than 2000 characters
			if len(line) > 2000 {
				line = line[:2000] + "..."
			}
			result.WriteString(fmt.Sprintf("%6d→%s\n", lineNum, line))
		}
		lineNum++
		if lineNum >= offset+limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return result.String(), nil
}
