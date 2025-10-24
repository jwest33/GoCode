package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/codegraph"
)

// FindReferencesTool finds symbol references
type FindReferencesTool struct {
	graph *codegraph.Graph
}

// NewFindReferencesTool creates a new find references tool
func NewFindReferencesTool(graph *codegraph.Graph) *FindReferencesTool {
	return &FindReferencesTool{graph: graph}
}

func (t *FindReferencesTool) Name() string {
	return "find_references"
}

func (t *FindReferencesTool) Description() string {
	return "Find all references to a symbol in the codebase. Use this to see where a function, class, or variable is used."
}

func (t *FindReferencesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file containing the symbol",
			},
			"line": map[string]interface{}{
				"type":        "number",
				"description": "Line number (0-indexed) where the symbol is defined or appears",
			},
			"column": map[string]interface{}{
				"type":        "number",
				"description": "Column number (0-indexed) where the symbol appears",
			},
		},
		"required": []string{"file_path", "line", "column"},
	}
}

type FindReferencesArgs struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (t *FindReferencesTool) Execute(ctx context.Context, args string) (string, error) {
	var frArgs FindReferencesArgs
	if err := UnmarshalArgs(args, &frArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Index the file first
	if err := t.graph.IndexFile(ctx, frArgs.FilePath); err != nil {
		return "", fmt.Errorf("failed to index file: %w", err)
	}

	// Find references
	symbols, err := t.graph.FindReferences(ctx, frArgs.FilePath, frArgs.Line, frArgs.Column)
	if err != nil {
		return "", fmt.Errorf("failed to find references: %w", err)
	}

	if len(symbols) == 0 {
		return "No references found for this symbol", nil
	}

	// Format results
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d reference(s):\n\n", len(symbols)))

	for i, sym := range symbols {
		result.WriteString(fmt.Sprintf("%d. %s:%d:%d\n", i+1, sym.FilePath, sym.Line, sym.Column))
		if sym.Signature != "" {
			result.WriteString(fmt.Sprintf("   Context: %s\n", sym.Signature))
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}
