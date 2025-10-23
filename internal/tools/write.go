package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type WriteTool struct{}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return "Writes content to a file. Creates new file or overwrites existing file."
}

func (t *WriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

type WriteArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (t *WriteTool) Execute(ctx context.Context, args string) (string, error) {
	var writeArgs WriteArgs
	if err := UnmarshalArgs(args, &writeArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(writeArgs.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(writeArgs.FilePath, []byte(writeArgs.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File created successfully at: %s", writeArgs.FilePath), nil
}
