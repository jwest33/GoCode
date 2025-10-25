package tools

import (
	"context"
	"fmt"

	"github.com/jake/gocode/internal/memory"
)

type StoreMemoryTool struct {
	ltm *memory.LongTermMemory
}

func NewStoreMemoryTool(ltm *memory.LongTermMemory) *StoreMemoryTool {
	return &StoreMemoryTool{ltm: ltm}
}

func (t *StoreMemoryTool) Name() string {
	return "store_memory"
}

func (t *StoreMemoryTool) Description() string {
	return `Store important information in long-term memory for future sessions. Use this to remember:
- Project goals and high-level plans
- Important design decisions and their rationale
- Architectural patterns discovered in the codebase
- Solutions to errors encountered
- Key facts about the project structure`
}

func (t *StoreMemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Type of memory",
				"enum":        []string{"fact", "decision", "pattern", "error"},
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "Short summary (1-2 sentences) of what this memory is about",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Detailed content of the memory",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tags for categorizing and searching (e.g., 'api', 'database', 'authentication')",
			},
			"importance": map[string]interface{}{
				"type":        "number",
				"description": "Importance score 0.0-1.0 (0.5=normal, 0.8+=high importance)",
				"minimum":     0.0,
				"maximum":     1.0,
			},
		},
		"required": []string{"type", "summary", "content", "tags"},
	}
}

type StoreMemoryArgs struct {
	Type       string   `json:"type"`
	Summary    string   `json:"summary"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Importance float32  `json:"importance"`
}

func (t *StoreMemoryTool) Execute(ctx context.Context, args string) (string, error) {
	var memArgs StoreMemoryArgs
	if err := UnmarshalArgs(args, &memArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set default importance if not provided
	if memArgs.Importance == 0 {
		memArgs.Importance = 0.5
	}

	// Create memory object
	mem := &memory.Memory{
		Type:       memory.MemoryType(memArgs.Type),
		Summary:    memArgs.Summary,
		Content:    memArgs.Content,
		Tags:       memArgs.Tags,
		Importance: memArgs.Importance,
	}

	// Store in long-term memory
	if err := t.ltm.Store(mem); err != nil {
		return "", fmt.Errorf("failed to store memory: %w", err)
	}

	return fmt.Sprintf("Memory stored successfully (ID: %s, Type: %s, Tags: %v)",
		mem.ID, mem.Type, mem.Tags), nil
}
