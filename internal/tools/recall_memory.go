package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/memory"
)

type RecallMemoryTool struct {
	ltm *memory.LongTermMemory
}

func NewRecallMemoryTool(ltm *memory.LongTermMemory) *RecallMemoryTool {
	return &RecallMemoryTool{ltm: ltm}
}

func (t *RecallMemoryTool) Name() string {
	return "recall_memory"
}

func (t *RecallMemoryTool) Description() string {
	return `Search and retrieve information from long-term memory. Use this to:
- Recall project goals and plans from previous sessions
- Look up design decisions and their rationale
- Find solutions to previously encountered errors
- Retrieve architectural patterns discovered earlier`
}

func (t *RecallMemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query (searches content, summary, and tags)",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Filter by memory type (optional)",
				"enum":        []string{"", "fact", "decision", "pattern", "error"},
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Filter by tags (optional, must match all tags)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of memories to return (default: 5)",
				"minimum":     1,
				"maximum":     20,
			},
		},
		"required": []string{},
	}
}

type RecallMemoryArgs struct {
	Query string   `json:"query"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
	Limit int      `json:"limit"`
}

func (t *RecallMemoryTool) Execute(ctx context.Context, args string) (string, error) {
	var recallArgs RecallMemoryArgs
	if err := UnmarshalArgs(args, &recallArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set default limit
	if recallArgs.Limit == 0 {
		recallArgs.Limit = 5
	}

	var memories []*memory.Memory
	var err error

	// Search by different criteria
	if len(recallArgs.Tags) > 0 {
		// Search by tags
		memories, err = t.ltm.GetByTags(recallArgs.Tags, recallArgs.Limit)
	} else if recallArgs.Type != "" {
		// Search by type
		memories, err = t.ltm.GetByType(memory.MemoryType(recallArgs.Type), recallArgs.Limit)
	} else if recallArgs.Query != "" {
		// Full-text search
		memories, err = t.ltm.Search(recallArgs.Query, recallArgs.Limit)
	} else {
		// Get most important memories
		memories, err = t.ltm.GetMostImportant(recallArgs.Limit)
	}

	if err != nil {
		return "", fmt.Errorf("failed to recall memories: %w", err)
	}

	if len(memories) == 0 {
		return "No memories found matching your criteria.", nil
	}

	// Format memories for display
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d relevant memor%s:\n\n",
		len(memories), pluralize(len(memories), "y", "ies")))

	for i, mem := range memories {
		result.WriteString(fmt.Sprintf("## Memory %d (ID: %s)\n", i+1, mem.ID))
		result.WriteString(fmt.Sprintf("**Type:** %s | **Importance:** %.1f | **Tags:** %s\n",
			mem.Type, mem.Importance, strings.Join(mem.Tags, ", ")))
		result.WriteString(fmt.Sprintf("**Summary:** %s\n\n", mem.Summary))
		result.WriteString(fmt.Sprintf("**Content:**\n%s\n\n", mem.Content))
		result.WriteString(fmt.Sprintf("*Accessed %d times, last: %s*\n\n",
			mem.AccessCount, formatTimeAgo(mem.LastAccess)))
		result.WriteString("---\n\n")
	}

	return result.String(), nil
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func formatTimeAgo(t interface{}) string {
	// Simple time ago formatter - can be enhanced
	return "recently"
}
