package tools

import (
	"context"
	"fmt"
	"strings"
)

type BashOutputTool struct {
	bashTool *BashTool
}

func NewBashOutputTool(bashTool *BashTool) *BashOutputTool {
	return &BashOutputTool{bashTool: bashTool}
}

func (t *BashOutputTool) Name() string {
	return "bash_output"
}

func (t *BashOutputTool) Description() string {
	return "Retrieves output from a running or completed background bash shell."
}

func (t *BashOutputTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"bash_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the background shell to retrieve output from",
			},
		},
		"required": []string{"bash_id"},
	}
}

type BashOutputArgs struct {
	BashID string `json:"bash_id"`
}

func (t *BashOutputTool) Execute(ctx context.Context, args string) (string, error) {
	var outputArgs BashOutputArgs
	if err := UnmarshalArgs(args, &outputArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	proc, ok := t.bashTool.GetProcess(outputArgs.BashID)
	if !ok {
		return "", fmt.Errorf("process not found: %s", outputArgs.BashID)
	}

	var result strings.Builder

	// Get new output since last check
	stdout := proc.Stdout.String()
	stderr := proc.Stderr.String()

	newOutput := stdout[proc.lastPos:]
	proc.lastPos = len(stdout)

	if newOutput != "" {
		result.WriteString("STDOUT:\n")
		result.WriteString(newOutput)
	}

	if stderr != "" {
		result.WriteString("\nSTDERR:\n")
		result.WriteString(stderr)
	}

	// Check if process is done
	select {
	case err := <-proc.Done:
		if err != nil {
			result.WriteString(fmt.Sprintf("\n\nProcess completed with error: %v", err))
		} else {
			result.WriteString("\n\nProcess completed successfully")
		}
	default:
		result.WriteString("\n\nProcess still running")
	}

	output := result.String()
	if output == "" {
		return "No new output", nil
	}

	return output, nil
}
