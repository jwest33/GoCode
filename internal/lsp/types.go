package lsp

// ClientCapabilities represents LSP client capabilities
type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// TextDocumentClientCapabilities represents text document capabilities
type TextDocumentClientCapabilities struct {
	Definition         *DefinitionCapability         `json:"definition,omitempty"`
	References         *ReferencesCapability         `json:"references,omitempty"`
	DocumentSymbol     *DocumentSymbolCapability     `json:"documentSymbol,omitempty"`
	Hover              *HoverCapability              `json:"hover,omitempty"`
	Implementation     *ImplementationCapability     `json:"implementation,omitempty"`
	TypeDefinition     *TypeDefinitionCapability     `json:"typeDefinition,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities
type WorkspaceClientCapabilities struct {
	Symbol *WorkspaceSymbolCapability `json:"symbol,omitempty"`
}

// DefinitionCapability represents definition capability
type DefinitionCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// ReferencesCapability represents references capability
type ReferencesCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolCapability represents document symbol capability
type DocumentSymbolCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport,omitempty"`
}

// HoverCapability represents hover capability
type HoverCapability struct {
	DynamicRegistration bool     `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

// ImplementationCapability represents implementation capability
type ImplementationCapability struct{
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// TypeDefinitionCapability represents type definition capability
type TypeDefinitionCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// WorkspaceSymbolCapability represents workspace symbol capability
type WorkspaceSymbolCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// InitializeParams represents initialization parameters
type InitializeParams struct {
	ProcessID    interface{}        `json:"processId"`
	RootURI      string             `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// InitializeResult represents initialization result
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	TextDocumentSync   interface{}                  `json:"textDocumentSync,omitempty"`
	DefinitionProvider bool                         `json:"definitionProvider,omitempty"`
	ReferencesProvider bool                         `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider bool                     `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider bool                    `json:"workspaceSymbolProvider,omitempty"`
	HoverProvider      bool                         `json:"hoverProvider,omitempty"`
	ImplementationProvider bool                     `json:"implementationProvider,omitempty"`
	TypeDefinitionProvider bool                     `json:"typeDefinitionProvider,omitempty"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentPositionParams represents text document position parameters
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// DefinitionParams represents parameters for textDocument/definition
type DefinitionParams struct {
	TextDocumentPositionParams
}

// ReferenceContext represents context for finding references
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceParams represents parameters for textDocument/references
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// DocumentSymbolParams represents parameters for textDocument/documentSymbol
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SymbolKind represents the kind of a symbol
type SymbolKind int

const (
	File          SymbolKind = 1
	Module        SymbolKind = 2
	Namespace     SymbolKind = 3
	Package       SymbolKind = 4
	Class         SymbolKind = 5
	Method        SymbolKind = 6
	Property      SymbolKind = 7
	Field         SymbolKind = 8
	Constructor   SymbolKind = 9
	Enum          SymbolKind = 10
	Interface     SymbolKind = 11
	Function      SymbolKind = 12
	Variable      SymbolKind = 13
	Constant      SymbolKind = 14
	String        SymbolKind = 15
	Number        SymbolKind = 16
	Boolean       SymbolKind = 17
	Array         SymbolKind = 18
	Object        SymbolKind = 19
	Key           SymbolKind = 20
	Null          SymbolKind = 21
	EnumMember    SymbolKind = 22
	Struct        SymbolKind = 23
	Event         SymbolKind = 24
	Operator      SymbolKind = 25
	TypeParameter SymbolKind = 26
)

// DocumentSymbol represents a symbol in a document
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// WorkspaceSymbolParams represents parameters for workspace/symbol
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// HoverParams represents parameters for textDocument/hover
type HoverParams struct {
	TextDocumentPositionParams
}

// MarkedString can be a string or a code block
type MarkedString struct {
	Language string `json:"language,omitempty"`
	Value    string `json:"value"`
}

// MarkupContent represents marked up content
type MarkupContent struct {
	Kind  string `json:"kind"` // "plaintext" or "markdown"
	Value string `json:"value"`
}

// Hover represents hover information
type Hover struct {
	Contents interface{} `json:"contents"` // Can be MarkedString, []MarkedString, or MarkupContent
	Range    *Range      `json:"range,omitempty"`
}

// DidOpenTextDocumentParams represents parameters for textDocument/didOpen
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentItem represents a text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidChangeTextDocumentParams represents parameters for textDocument/didChange
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentContentChangeEvent represents a change to a text document
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams represents parameters for textDocument/didClose
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
