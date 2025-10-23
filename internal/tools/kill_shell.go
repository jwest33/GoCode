package tools

import (
	"context"
	"fmt"
)

type KillShellTool struct {
	bashTool *BashTool
}

func NewKillShellTool(bashTool *BashTool) *KillShellTool {
	return &KillShellTool{bashTool: bashTool}
}

func (t *KillShellTool) Name() string {
	return "kill_shell"
}

func (t *KillShellTool) Description() string {
	return "Kills a running background bash shell by its ID."
}

func (t *KillShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"shell_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the background shell to kill",
			},
		},
		"required": []string{"shell_id"},
	}
}

type KillShellArgs struct {
	ShellID string `json:"shell_id"`
}

func (t *KillShellTool) Execute(ctx context.Context, args string) (string, error) {
	var killArgs KillShellArgs
	if err := UnmarshalArgs(args, &killArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if err := t.bashTool.KillProcess(killArgs.ShellID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Process %s killed successfully", killArgs.ShellID), nil
}
