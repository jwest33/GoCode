package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type EditTool struct{}

func (t *EditTool) Name() string {
	return "edit"
}

func (t *EditTool) Description() string {
	return "Performs exact string replacement in files. The old_string must be unique in the file unless replace_all is true."
}

func (t *EditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace it with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default false)",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

type EditArgs struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (t *EditTool) Execute(ctx context.Context, args string) (string, error) {
	var editArgs EditArgs
	if err := UnmarshalArgs(args, &editArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	content, err := os.ReadFile(editArgs.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)

	if editArgs.ReplaceAll {
		newContent := strings.ReplaceAll(contentStr, editArgs.OldString, editArgs.NewString)
		if err := os.WriteFile(editArgs.FilePath, []byte(newContent), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		count := strings.Count(contentStr, editArgs.OldString)
		return fmt.Sprintf("Replaced %d occurrence(s) in %s", count, editArgs.FilePath), nil
	}

	// Check if old_string is unique
	count := strings.Count(contentStr, editArgs.OldString)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}
	if count > 1 {
		return "", fmt.Errorf("old_string appears %d times in file, must be unique or use replace_all", count)
	}

	newContent := strings.Replace(contentStr, editArgs.OldString, editArgs.NewString, 1)
	if err := os.WriteFile(editArgs.FilePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Edit successful in %s", editArgs.FilePath), nil
}
