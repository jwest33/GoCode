package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type BashTool struct {
	processes map[string]*BackgroundProcess
	mu        sync.RWMutex
	nextID    int
}

type BackgroundProcess struct {
	ID      string
	Cmd     *exec.Cmd
	Stdout  *bytes.Buffer
	Stderr  *bytes.Buffer
	Done    chan error
	lastPos int
}

func NewBashTool() *BashTool {
	return &BashTool{
		processes: make(map[string]*BackgroundProcess),
	}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Executes bash commands with optional timeout and background execution support. Use this to run tests (python tests.py, npm test, go test), builds, and other shell commands."
}

func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Clear description of what this command does (5-10 words)",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Optional timeout in milliseconds (default 120000, max 600000)",
			},
			"run_in_background": map[string]interface{}{
				"type":        "boolean",
				"description": "Run command in background",
			},
		},
		"required": []string{"command"},
	}
}

type BashArgs struct {
	Command         string `json:"command"`
	Description     string `json:"description,omitempty"`
	Timeout         int    `json:"timeout,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
}

func (t *BashTool) Execute(ctx context.Context, args string) (string, error) {
	var bashArgs BashArgs
	if err := UnmarshalArgs(args, &bashArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if bashArgs.RunInBackground {
		return t.executeBackground(bashArgs)
	}

	return t.executeForeground(ctx, bashArgs)
}

func (t *BashTool) executeForeground(ctx context.Context, args BashArgs) (string, error) {
	timeout := args.Timeout
	if timeout == 0 {
		timeout = 120000 // 2 minutes default
	}
	if timeout > 600000 {
		timeout = 600000 // 10 minutes max
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Use cmd.exe for Windows
	cmd := exec.CommandContext(execCtx, "cmd", "/C", args.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	// Truncate if over 30000 characters
	if len(output) > 30000 {
		output = output[:30000] + "\n... (output truncated)"
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("command timed out after %dms", timeout)
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

func (t *BashTool) executeBackground(args BashArgs) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.nextID++
	id := fmt.Sprintf("bash_%d", t.nextID)

	cmd := exec.Command("cmd", "/C", args.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	proc := &BackgroundProcess{
		ID:     id,
		Cmd:    cmd,
		Stdout: &stdout,
		Stderr: &stderr,
		Done:   make(chan error, 1),
	}

	t.processes[id] = proc

	if err := cmd.Start(); err != nil {
		delete(t.processes, id)
		return "", fmt.Errorf("failed to start background process: %w", err)
	}

	// Monitor process completion
	go func() {
		proc.Done <- cmd.Wait()
	}()

	return fmt.Sprintf("Background process started with ID: %s\nUse bash_output tool to read output.", id), nil
}

func (t *BashTool) GetProcess(id string) (*BackgroundProcess, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	proc, ok := t.processes[id]
	return proc, ok
}

func (t *BashTool) KillProcess(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	proc, ok := t.processes[id]
	if !ok {
		return fmt.Errorf("process not found: %s", id)
	}

	if proc.Cmd.Process != nil {
		if err := proc.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	delete(t.processes, id)
	return nil
}
