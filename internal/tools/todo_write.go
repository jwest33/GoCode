package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type TodoWriteTool struct {
	todos    []TodoItem
	todoFile string
	mu       sync.RWMutex
}

type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

func NewTodoWriteTool(todoFile string) *TodoWriteTool {
	t := &TodoWriteTool{
		todoFile: todoFile,
		todos:    []TodoItem{},
	}
	t.Load()
	return t
}

func (t *TodoWriteTool) Name() string {
	return "todo_write"
}

func (t *TodoWriteTool) Description() string {
	return "Creates and manages a structured task list. Tracks progress with pending/in_progress/completed states."
}

func (t *TodoWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The task description (imperative form, e.g., 'Run tests')",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Task status",
							"enum":        []string{"pending", "in_progress", "completed"},
						},
						"activeForm": map[string]interface{}{
							"type":        "string",
							"description": "Present continuous form (e.g., 'Running tests')",
						},
					},
					"required": []string{"content", "status", "activeForm"},
				},
			},
		},
		"required": []string{"todos"},
	}
}

type TodoWriteArgs struct {
	Todos []TodoItem `json:"todos"`
}

func (t *TodoWriteTool) Execute(ctx context.Context, args string) (string, error) {
	var todoArgs TodoWriteArgs
	if err := UnmarshalArgs(args, &todoArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	t.mu.Lock()
	t.todos = todoArgs.Todos
	t.mu.Unlock()

	if err := t.Save(); err != nil {
		return "", fmt.Errorf("failed to save todos: %w", err)
	}

	return t.FormatTodos(), nil
}

func (t *TodoWriteTool) FormatTodos() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result strings.Builder
	result.WriteString("Updated todo list:\n\n")

	for i, todo := range t.todos {
		status := ""
		switch todo.Status {
		case "pending":
			status = "[ ]"
		case "in_progress":
			status = "[→]"
		case "completed":
			status = "[✓]"
		}
		result.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, todo.Content))
	}

	return result.String()
}

func (t *TodoWriteTool) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var content strings.Builder
	content.WriteString("# TODO\n\n")
	content.WriteString("This file tracks pending tasks between interactions.\n\n")

	for i, todo := range t.todos {
		status := ""
		switch todo.Status {
		case "pending":
			status = "[ ]"
		case "in_progress":
			status = "[→]"
		case "completed":
			status = "[✓]"
		}
		content.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, todo.Content))
	}

	// Also save JSON for easy loading
	content.WriteString("\n---\n\n```json\n")
	jsonData, err := json.MarshalIndent(t.todos, "", "  ")
	if err != nil {
		return err
	}
	content.WriteString(string(jsonData))
	content.WriteString("\n```\n")

	return os.WriteFile(t.todoFile, []byte(content.String()), 0644)
}

func (t *TodoWriteTool) Load() error {
	data, err := os.ReadFile(t.todoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return err
	}

	// Try to extract JSON from markdown code block
	content := string(data)
	start := strings.Index(content, "```json")
	if start == -1 {
		return nil
	}

	start += 7 // Skip ```json\n
	end := strings.Index(content[start:], "```")
	if end == -1 {
		return nil
	}

	jsonContent := content[start : start+end]
	jsonContent = strings.TrimSpace(jsonContent)

	t.mu.Lock()
	defer t.mu.Unlock()

	return json.Unmarshal([]byte(jsonContent), &t.todos)
}

func (t *TodoWriteTool) GetTodos() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]TodoItem{}, t.todos...)
}
