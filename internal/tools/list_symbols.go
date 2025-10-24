package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/codegraph"
)

// ListSymbolsTool lists all symbols in a file
type ListSymbolsTool struct {
	graph *codegraph.Graph
}

// NewListSymbolsTool creates a new list symbols tool
func NewListSymbolsTool(graph *codegraph.Graph) *ListSymbolsTool {
	return &ListSymbolsTool{graph: graph}
}

func (t *ListSymbolsTool) Name() string {
	return "list_symbols"
}

func (t *ListSymbolsTool) Description() string {
	return "List all symbols (functions, classes, variables, etc.) defined in a file. Useful for understanding file structure."
}

func (t *ListSymbolsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to analyze",
			},
			"kind_filter": map[string]interface{}{
				"type":        "string",
				"description": "Optional filter by symbol kind (function, class, method, variable, etc.)",
			},
		},
		"required": []string{"file_path"},
	}
}

type ListSymbolsArgs struct {
	FilePath   string `json:"file_path"`
	KindFilter string `json:"kind_filter,omitempty"`
}

func (t *ListSymbolsTool) Execute(ctx context.Context, args string) (string, error) {
	var lsArgs ListSymbolsArgs
	if err := UnmarshalArgs(args, &lsArgs); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Index the file first
	if err := t.graph.IndexFile(ctx, lsArgs.FilePath); err != nil {
		return "", fmt.Errorf("failed to index file: %w", err)
	}

	// Get symbols
	symbols := t.graph.GetSymbolsByFile(lsArgs.FilePath)

	if len(symbols) == 0 {
		return "No symbols found in this file", nil
	}

	// Filter by kind if specified
	if lsArgs.KindFilter != "" {
		filtered := []*codegraph.SymbolNode{}
		for _, sym := range symbols {
			if strings.EqualFold(sym.Kind, lsArgs.KindFilter) {
				filtered = append(filtered, sym)
			}
		}
		symbols = filtered
	}

	if len(symbols) == 0 {
		return fmt.Sprintf("No symbols of kind '%s' found in this file", lsArgs.KindFilter), nil
	}

	// Format results
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d symbol(s) in %s:\n\n", len(symbols), lsArgs.FilePath))

	// Group by kind
	byKind := make(map[string][]*codegraph.SymbolNode)
	for _, sym := range symbols {
		byKind[sym.Kind] = append(byKind[sym.Kind], sym)
	}

	for kind, syms := range byKind {
		result.WriteString(fmt.Sprintf("## %s (%d)\n", strings.Title(kind), len(syms)))
		for _, sym := range syms {
			result.WriteString(fmt.Sprintf("  - %s (line %d)\n", sym.Name, sym.Line))
			if sym.Signature != "" {
				result.WriteString(fmt.Sprintf("    %s\n", sym.Signature))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}
