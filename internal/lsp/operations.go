package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// GetDefinition gets the definition of a symbol at a position
func (c *Client) GetDefinition(ctx context.Context, uri string, line, character int) ([]Location, error) {
	params := DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: line, Character: character},
		},
	}

	var result json.RawMessage
	if err := c.Call(ctx, "textDocument/definition", params, &result); err != nil {
		return nil, err
	}

	// Result can be Location, []Location, or null
	if len(result) == 0 || string(result) == "null" {
		return []Location{}, nil
	}

	// Try single location
	var singleLoc Location
	if err := json.Unmarshal(result, &singleLoc); err == nil {
		return []Location{singleLoc}, nil
	}

	// Try array of locations
	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal definition result: %w", err)
	}

	return locations, nil
}

// GetReferences gets all references to a symbol at a position
func (c *Client) GetReferences(ctx context.Context, uri string, line, character int, includeDeclaration bool) ([]Location, error) {
	params := ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: line, Character: character},
		},
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	var locations []Location
	if err := c.Call(ctx, "textDocument/references", params, &locations); err != nil {
		return nil, err
	}

	return locations, nil
}

// GetDocumentSymbols gets all symbols in a document
func (c *Client) GetDocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, []SymbolInformation, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	var result json.RawMessage
	if err := c.Call(ctx, "textDocument/documentSymbol", params, &result); err != nil {
		return nil, nil, err
	}

	if len(result) == 0 || string(result) == "null" {
		return []DocumentSymbol{}, []SymbolInformation{}, nil
	}

	// Try hierarchical document symbols first
	var docSymbols []DocumentSymbol
	if err := json.Unmarshal(result, &docSymbols); err == nil && len(docSymbols) > 0 {
		// Check if it's really document symbols (has range)
		if docSymbols[0].Range.Start.Line >= 0 {
			return docSymbols, nil, nil
		}
	}

	// Fall back to symbol information
	var symInfo []SymbolInformation
	if err := json.Unmarshal(result, &symInfo); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal document symbols: %w", err)
	}

	return nil, symInfo, nil
}

// GetWorkspaceSymbols searches for symbols in the workspace
func (c *Client) GetWorkspaceSymbols(ctx context.Context, query string) ([]SymbolInformation, error) {
	params := WorkspaceSymbolParams{
		Query: query,
	}

	var symbols []SymbolInformation
	if err := c.Call(ctx, "workspace/symbol", params, &symbols); err != nil {
		return nil, err
	}

	return symbols, nil
}

// GetHover gets hover information at a position
func (c *Client) GetHover(ctx context.Context, uri string, line, character int) (*Hover, error) {
	params := HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: line, Character: character},
		},
	}

	var result json.RawMessage
	if err := c.Call(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 || string(result) == "null" {
		return nil, nil
	}

	var hover Hover
	if err := json.Unmarshal(result, &hover); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hover result: %w", err)
	}

	return &hover, nil
}

// DidOpenTextDocument notifies the server that a document was opened
func (c *Client) DidOpenTextDocument(uri, languageID, text string) error {
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       text,
		},
	}

	return c.Notify("textDocument/didOpen", params)
}

// DidCloseTextDocument notifies the server that a document was closed
func (c *Client) DidCloseTextDocument(uri string) error {
	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	return c.Notify("textDocument/didClose", params)
}

// OpenFile opens a file and sends didOpen notification
func (c *Client) OpenFile(filePath string, languageID string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	uri := pathToURI(filePath)
	return c.DidOpenTextDocument(uri, languageID, string(content))
}

// CloseFile sends didClose notification for a file
func (c *Client) CloseFile(filePath string) error {
	uri := pathToURI(filePath)
	return c.DidCloseTextDocument(uri)
}

// pathToURI converts a file path to a URI
func pathToURI(path string) string {
	// Simple conversion - may need to handle Windows paths specially
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	return "file://" + path
}

// URIToPath converts a URI to a file path
func URIToPath(uri string) string {
	// Remove file:// prefix
	if len(uri) > 7 && uri[:7] == "file://" {
		path := uri[7:]
		// On Windows, paths start with /C:/ which we need to convert to C:/
		if len(path) > 2 && path[0] == '/' && path[2] == ':' {
			return path[1:]
		}
		return path
	}
	return uri
}

// DefaultClientCapabilities returns default client capabilities
func DefaultClientCapabilities() ClientCapabilities {
	return ClientCapabilities{
		TextDocument: &TextDocumentClientCapabilities{
			Definition: &DefinitionCapability{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			References: &ReferencesCapability{
				DynamicRegistration: false,
			},
			DocumentSymbol: &DocumentSymbolCapability{
				DynamicRegistration:               false,
				HierarchicalDocumentSymbolSupport: true,
			},
			Hover: &HoverCapability{
				DynamicRegistration: false,
				ContentFormat:       []string{"markdown", "plaintext"},
			},
			Implementation: &ImplementationCapability{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			TypeDefinition: &TypeDefinitionCapability{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
		},
		Workspace: &WorkspaceClientCapabilities{
			Symbol: &WorkspaceSymbolCapability{
				DynamicRegistration: false,
			},
		},
	}
}
