package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/codegraph"
)

// FindDefinitionTool finds symbol definitions
type FindDefinitionTool struct {
	graph *codegraph.Graph
}

// NewFindDefinitionTool creates a new find definition tool
func NewFindDefinitionTool(graph *codegraph.Graph) *FindDefinitionTool {
	return &FindDefinitionTool{graph: graph}
}

func (t *FindDefinitionTool) Name() string {
	return "find_definition"
}

func (t *FindDefinitionTool) Description() string {
	return "Find the definition of a symbol at a specific location in a file. Use this to jump to where a function, class, or variable is defined."
}

func (t *FindDefinitionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file containing the symbol",
			},
			"line": map[string]interface{}{
				"type":        "number",
				"description": "Line number (0-indexed) where the symbol appears",
			},
			"column": map[string]interface{}{
				"type":        "number",
				"description": "Column number (0-indexed) where the symbol appears",
			},
		},
		"required": []string{"file_path", "line", "column"},
	}
}

type FindDefinitionArgs struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (t *FindDefinitionTool) Execute(ctx context.Context, args string) (string, error) {
	var fdArgs FindDefinitionArgs
	if err := UnmarshalArgs(args, &fdArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Index the file first
	if err := t.graph.IndexFile(ctx, fdArgs.FilePath); err != nil {
		return "", fmt.Errorf("failed to index file: %w", err)
	}

	// Find definitions
	symbols, err := t.graph.FindDefinitions(ctx, fdArgs.FilePath, fdArgs.Line, fdArgs.Column)
	if err != nil {
		return "", fmt.Errorf("failed to find definitions: %w", err)
	}

	if len(symbols) == 0 {
		return "No definitions found at this location", nil
	}

	// Format results
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d definition(s):\n\n", len(symbols)))

	for i, sym := range symbols {
		result.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, sym.Name, sym.Kind))
		result.WriteString(fmt.Sprintf("   Location: %s:%d:%d\n", sym.FilePath, sym.Line, sym.Column))
		if sym.Signature != "" {
			result.WriteString(fmt.Sprintf("   Signature: %s\n", sym.Signature))
		}
		if sym.DocString != "" {
			result.WriteString(fmt.Sprintf("   Doc: %s\n", sym.DocString))
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}
