package codegraph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jake/gocode/internal/lsp"
	"github.com/jake/gocode/internal/parser"
)

// Graph represents a code graph with symbols, definitions, references, and calls
type Graph struct {
	rootPath string
	lspMgr   *lsp.Manager

	// Graph data
	symbols     map[string]*SymbolNode      // symbol ID -> node
	files       map[string]*FileNode        // file path -> file node
	edges       map[string][]*Edge          // source symbol ID -> edges
	mu          sync.RWMutex

	// Cache
	cacheValid  map[string]bool             // file path -> is cache valid
	cacheMu     sync.RWMutex
}

// SymbolNode represents a symbol in the code graph
type SymbolNode struct {
	ID          string           // Unique identifier
	Name        string           // Symbol name
	Kind        string           // function, class, variable, etc.
	FilePath    string           // File containing the symbol
	Line        int              // Line number
	Column      int              // Column number
	Signature   string           // Full signature/declaration
	DocString   string           // Documentation
	ParentID    string           // Parent symbol (for methods in classes, etc.)
}

// FileNode represents a file in the code graph
type FileNode struct {
	Path         string
	Language     string
	SymbolIDs    []string         // Symbols defined in this file
	Imports      []string         // Imported packages/modules
	LastModified int64            // Unix timestamp
}

// Edge represents a relationship between symbols
type Edge struct {
	From     string     // Source symbol ID
	To       string     // Target symbol ID
	Type     EdgeType   // Type of relationship
	FilePath string     // File where this edge occurs
	Line     int        // Line where this edge occurs
}

// EdgeType represents the type of relationship
type EdgeType string

const (
	EdgeDefinition   EdgeType = "definition"   // A defines B
	EdgeReference    EdgeType = "reference"    // A references B
	EdgeCall         EdgeType = "call"         // A calls B
	EdgeInherits     EdgeType = "inherits"     // A inherits from B
	EdgeImplements   EdgeType = "implements"   // A implements B
	EdgeImports      EdgeType = "imports"      // A imports B
)

// NewGraph creates a new code graph
func NewGraph(rootPath string, lspMgr *lsp.Manager) *Graph {
	return &Graph{
		rootPath:    rootPath,
		lspMgr:      lspMgr,
		symbols:     make(map[string]*SymbolNode),
		files:       make(map[string]*FileNode),
		edges:       make(map[string][]*Edge),
		cacheValid:  make(map[string]bool),
	}
}

// IndexFile indexes a file and builds its symbol graph
func (g *Graph) IndexFile(ctx context.Context, filePath string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Read file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if cache is still valid
	if g.isCacheValid(filePath, fileInfo.ModTime().Unix()) {
		return nil // Already up to date
	}

	// Create or get file node
	fileNode := g.getOrCreateFileNode(filePath)
	fileNode.LastModified = fileInfo.ModTime().Unix()

	// Clear old symbols for this file
	g.clearFileSymbols(filePath)

	// Try LSP first
	if g.lspMgr != nil {
		if err := g.indexWithLSP(ctx, filePath, fileNode); err == nil {
			g.markCacheValid(filePath)
			return nil
		}
		// LSP failed, fall back to simple parser
	}

	// Fall back to simple parser
	if err := g.indexWithParser(filePath, fileNode); err != nil {
		return err
	}

	g.markCacheValid(filePath)
	return nil
}

// indexWithLSP indexes a file using LSP
func (g *Graph) indexWithLSP(ctx context.Context, filePath string, fileNode *FileNode) error {
	docSymbols, symInfo, err := g.lspMgr.GetDocumentSymbols(ctx, filePath)
	if err != nil {
		return err
	}

	// Process hierarchical document symbols
	if len(docSymbols) > 0 {
		for _, docSym := range docSymbols {
			g.processDocumentSymbol(filePath, &docSym, "")
		}
	} else if len(symInfo) > 0 {
		// Process flat symbol information
		for _, sym := range symInfo {
			g.processSymbolInformation(filePath, &sym)
		}
	}

	return nil
}

// processDocumentSymbol processes a hierarchical document symbol
func (g *Graph) processDocumentSymbol(filePath string, docSym *lsp.DocumentSymbol, parentID string) {
	symbolID := fmt.Sprintf("%s:%d:%d:%s", filePath, docSym.Range.Start.Line, docSym.Range.Start.Character, docSym.Name)

	node := &SymbolNode{
		ID:        symbolID,
		Name:      docSym.Name,
		Kind:      symbolKindToString(docSym.Kind),
		FilePath:  filePath,
		Line:      docSym.Range.Start.Line,
		Column:    docSym.Range.Start.Character,
		Signature: docSym.Detail,
		ParentID:  parentID,
	}

	g.symbols[symbolID] = node

	// Add to file node
	if fileNode, ok := g.files[filePath]; ok {
		fileNode.SymbolIDs = append(fileNode.SymbolIDs, symbolID)
	}

	// Process children recursively
	for i := range docSym.Children {
		g.processDocumentSymbol(filePath, &docSym.Children[i], symbolID)
	}
}

// processSymbolInformation processes a flat symbol
func (g *Graph) processSymbolInformation(filePath string, sym *lsp.SymbolInformation) {
	symbolID := fmt.Sprintf("%s:%d:%d:%s", filePath, sym.Location.Range.Start.Line, sym.Location.Range.Start.Character, sym.Name)

	node := &SymbolNode{
		ID:       symbolID,
		Name:     sym.Name,
		Kind:     symbolKindToString(sym.Kind),
		FilePath: lsp.URIToPath(sym.Location.URI),
		Line:     sym.Location.Range.Start.Line,
		Column:   sym.Location.Range.Start.Character,
	}

	g.symbols[symbolID] = node

	if fileNode, ok := g.files[filePath]; ok {
		fileNode.SymbolIDs = append(fileNode.SymbolIDs, symbolID)
	}
}

// indexWithParser indexes a file using the simple parser
func (g *Graph) indexWithParser(filePath string, fileNode *FileNode) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine language from extension
	language := getLanguageFromPath(filePath)
	p := parser.NewSimpleParser(language)

	symbols := p.Parse(string(content))

	for _, sym := range symbols {
		symbolID := fmt.Sprintf("%s:%d:%d:%s", filePath, sym.Line, sym.Column, sym.Name)

		node := &SymbolNode{
			ID:        symbolID,
			Name:      sym.Name,
			Kind:      string(sym.Kind),
			FilePath:  filePath,
			Line:      sym.Line,
			Column:    sym.Column,
			Signature: sym.Signature,
			DocString: sym.DocString,
		}

		g.symbols[symbolID] = node
		fileNode.SymbolIDs = append(fileNode.SymbolIDs, symbolID)
	}

	return nil
}

// FindDefinitions finds definitions of a symbol at a location
func (g *Graph) FindDefinitions(ctx context.Context, filePath string, line, column int) ([]*SymbolNode, error) {
	if g.lspMgr != nil {
		locations, err := g.lspMgr.FindDefinition(ctx, filePath, line, column)
		if err == nil && len(locations) > 0 {
			return g.locationsToSymbols(locations), nil
		}
	}

	// Fallback: search in graph
	return g.findDefinitionsInGraph(filePath, line, column), nil
}

// FindReferences finds all references to a symbol
func (g *Graph) FindReferences(ctx context.Context, filePath string, line, column int) ([]*SymbolNode, error) {
	if g.lspMgr != nil {
		locations, err := g.lspMgr.FindReferences(ctx, filePath, line, column, true)
		if err == nil && len(locations) > 0 {
			return g.locationsToSymbols(locations), nil
		}
	}

	// Fallback: search in graph edges
	return g.findReferencesInGraph(filePath, line, column), nil
}

// GetSymbolsByFile returns all symbols in a file
func (g *Graph) GetSymbolsByFile(filePath string) []*SymbolNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fileNode, ok := g.files[filePath]
	if !ok {
		return []*SymbolNode{}
	}

	symbols := make([]*SymbolNode, 0, len(fileNode.SymbolIDs))
	for _, symbolID := range fileNode.SymbolIDs {
		if sym, ok := g.symbols[symbolID]; ok {
			symbols = append(symbols, sym)
		}
	}

	return symbols
}

// TraverseFrom performs graph traversal from a symbol
func (g *Graph) TraverseFrom(symbolID string, edgeType EdgeType, maxDepth int) []*SymbolNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	result := []*SymbolNode{}

	g.traverse(symbolID, edgeType, maxDepth, 0, visited, &result)

	return result
}

// traverse is the recursive traversal function
func (g *Graph) traverse(symbolID string, edgeType EdgeType, maxDepth, currentDepth int, visited map[string]bool, result *[]*SymbolNode) {
	if currentDepth >= maxDepth || visited[symbolID] {
		return
	}

	visited[symbolID] = true

	// Add current symbol
	if sym, ok := g.symbols[symbolID]; ok {
		*result = append(*result, sym)
	}

	// Traverse edges
	if edges, ok := g.edges[symbolID]; ok {
		for _, edge := range edges {
			if edgeType == "" || edge.Type == edgeType {
				g.traverse(edge.To, edgeType, maxDepth, currentDepth+1, visited, result)
			}
		}
	}
}

// Helper functions

func (g *Graph) getOrCreateFileNode(filePath string) *FileNode {
	if fileNode, ok := g.files[filePath]; ok {
		return fileNode
	}

	fileNode := &FileNode{
		Path:      filePath,
		Language:  getLanguageFromPath(filePath),
		SymbolIDs: []string{},
	}

	g.files[filePath] = fileNode
	return fileNode
}

func (g *Graph) clearFileSymbols(filePath string) {
	if fileNode, ok := g.files[filePath]; ok {
		// Remove symbols
		for _, symbolID := range fileNode.SymbolIDs {
			delete(g.symbols, symbolID)
			delete(g.edges, symbolID)
		}
		fileNode.SymbolIDs = []string{}
	}
}

func (g *Graph) isCacheValid(filePath string, modTime int64) bool {
	g.cacheMu.RLock()
	defer g.cacheMu.RUnlock()

	if !g.cacheValid[filePath] {
		return false
	}

	if fileNode, ok := g.files[filePath]; ok {
		return fileNode.LastModified >= modTime
	}

	return false
}

func (g *Graph) markCacheValid(filePath string) {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	g.cacheValid[filePath] = true
}

func (g *Graph) locationsToSymbols(locations []lsp.Location) []*SymbolNode {
	symbols := make([]*SymbolNode, 0, len(locations))

	for _, loc := range locations {
		filePath := lsp.URIToPath(loc.URI)
		line := loc.Range.Start.Line
		column := loc.Range.Start.Character

		// Find symbol at this location
		symbolID := g.findSymbolIDAtLocation(filePath, line, column)
		if symbolID != "" {
			if sym, ok := g.symbols[symbolID]; ok {
				symbols = append(symbols, sym)
			}
		}
	}

	return symbols
}

func (g *Graph) findSymbolIDAtLocation(filePath string, line, column int) string {
	if fileNode, ok := g.files[filePath]; ok {
		for _, symbolID := range fileNode.SymbolIDs {
			if sym, ok := g.symbols[symbolID]; ok {
				if sym.Line == line {
					return symbolID
				}
			}
		}
	}
	return ""
}

func (g *Graph) findDefinitionsInGraph(filePath string, line, column int) []*SymbolNode {
	// Simple implementation - find symbol at location
	symbolID := g.findSymbolIDAtLocation(filePath, line, column)
	if symbolID == "" {
		return []*SymbolNode{}
	}

	if sym, ok := g.symbols[symbolID]; ok {
		return []*SymbolNode{sym}
	}

	return []*SymbolNode{}
}

func (g *Graph) findReferencesInGraph(filePath string, line, column int) []*SymbolNode {
	// Simple implementation - return empty for now
	// Could be enhanced to track references in the graph
	return []*SymbolNode{}
}

func getLanguageFromPath(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".ts", ".jsx", ".tsx":
		return "javascript"
	default:
		return "unknown"
	}
}

func symbolKindToString(kind lsp.SymbolKind) string {
	switch kind {
	case lsp.Function:
		return "function"
	case lsp.Method:
		return "method"
	case lsp.Class:
		return "class"
	case lsp.Interface:
		return "interface"
	case lsp.Struct:
		return "struct"
	case lsp.Variable:
		return "variable"
	case lsp.Constant:
		return "constant"
	default:
		return "unknown"
	}
}
