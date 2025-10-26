package lsp

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// LanguageServerConfig holds configuration for a language server
type LanguageServerConfig struct {
	Command  string   // Command to run the language server
	Args     []string // Arguments for the language server
	FileExts []string // File extensions this server handles
}

// DefaultLanguageServers returns default language server configurations
func DefaultLanguageServers() map[string]LanguageServerConfig {
	return map[string]LanguageServerConfig{
		"go": {
			Command:  "gopls",
			Args:     []string{},
			FileExts: []string{".go"},
		},
		"python": {
			Command:  "pylsp", // python-lsp-server
			Args:     []string{},
			FileExts: []string{".py"},
		},
		"typescript": {
			Command:  "typescript-language-server",
			Args:     []string{"--stdio"},
			FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		},
		"rust": {
			Command:  "rust-analyzer",
			Args:     []string{},
			FileExts: []string{".rs"},
		},
	}
}

// Manager manages multiple LSP clients for different languages
type Manager struct {
	clients     map[string]*Client // language -> client
	configs     map[string]LanguageServerConfig
	rootURI     string
	mu          sync.RWMutex
	initialized map[string]bool
}

// NewManager creates a new LSP manager
func NewManager(rootPath string, configs map[string]LanguageServerConfig) *Manager {
	return &Manager{
		clients:     make(map[string]*Client),
		configs:     configs,
		rootURI:     pathToURI(rootPath),
		initialized: make(map[string]bool),
	}
}

// GetClientForFile returns an LSP client for the given file
func (m *Manager) GetClientForFile(ctx context.Context, filePath string) (*Client, string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Find language for this extension
	var language string
	var config LanguageServerConfig
	for lang, cfg := range m.configs {
		for _, fileExt := range cfg.FileExts {
			if fileExt == ext {
				language = lang
				config = cfg
				break
			}
		}
		if language != "" {
			break
		}
	}

	if language == "" {
		return nil, "", fmt.Errorf("no language server configured for file extension: %s", ext)
	}

	// Get or create client for this language
	client, err := m.getOrCreateClient(ctx, language, config)
	if err != nil {
		return nil, "", err
	}

	return client, language, nil
}

// getOrCreateClient gets an existing client or creates a new one
func (m *Manager) getOrCreateClient(ctx context.Context, language string, config LanguageServerConfig) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return existing client if available
	if client, exists := m.clients[language]; exists {
		return client, nil
	}

	// Create new client
	client, err := NewClient(config.Command, config.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LSP client for %s: %w", language, err)
	}

	// Initialize the client
	capabilities := DefaultClientCapabilities()
	_, err = client.Initialize(ctx, m.rootURI, capabilities)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize LSP client for %s: %w", language, err)
	}

	m.clients[language] = client
	m.initialized[language] = true

	return client, nil
}

// FindDefinition finds the definition of a symbol
func (m *Manager) FindDefinition(ctx context.Context, filePath string, line, character int) ([]Location, error) {
	client, _, err := m.GetClientForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Open the file if not already open
	if err := client.OpenFile(filePath, m.getLanguageID(filePath)); err != nil {
		return nil, err
	}

	uri := pathToURI(filePath)
	return client.GetDefinition(ctx, uri, line, character)
}

// FindReferences finds all references to a symbol
func (m *Manager) FindReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]Location, error) {
	client, _, err := m.GetClientForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Open the file if not already open
	if err := client.OpenFile(filePath, m.getLanguageID(filePath)); err != nil {
		return nil, err
	}

	uri := pathToURI(filePath)
	return client.GetReferences(ctx, uri, line, character, includeDeclaration)
}

// GetDocumentSymbols gets all symbols in a document
func (m *Manager) GetDocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, []SymbolInformation, error) {
	client, _, err := m.GetClientForFile(ctx, filePath)
	if err != nil {
		return nil, nil, err
	}

	// Open the file if not already open
	if err := client.OpenFile(filePath, m.getLanguageID(filePath)); err != nil {
		return nil, nil, err
	}

	uri := pathToURI(filePath)
	return client.GetDocumentSymbols(ctx, uri)
}

// SearchWorkspaceSymbols searches for symbols in the workspace
func (m *Manager) SearchWorkspaceSymbols(ctx context.Context, query string, language string) ([]SymbolInformation, error) {
	m.mu.RLock()
	client, exists := m.clients[language]
	m.mu.RUnlock()

	if !exists {
		// Try to initialize client for this language
		config, ok := m.configs[language]
		if !ok {
			return nil, fmt.Errorf("no configuration for language: %s", language)
		}

		var err error
		client, err = m.getOrCreateClient(ctx, language, config)
		if err != nil {
			return nil, err
		}
	}

	return client.GetWorkspaceSymbols(ctx, query)
}

// GetHover gets hover information for a symbol
func (m *Manager) GetHover(ctx context.Context, filePath string, line, character int) (*Hover, error) {
	client, _, err := m.GetClientForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Open the file if not already open
	if err := client.OpenFile(filePath, m.getLanguageID(filePath)); err != nil {
		return nil, err
	}

	uri := pathToURI(filePath)
	return client.GetHover(ctx, uri, line, character)
}

// getLanguageID returns the language ID for a file
func (m *Manager) getLanguageID(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".ts":   "typescript",
		".tsx":  "typescriptreact",
		".js":   "javascript",
		".jsx":  "javascriptreact",
		".rs":   "rust",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".h":    "c",
		".hpp":  "cpp",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "plaintext"
}

// ValidateServers checks which configured LSP servers are available
// Returns a map of language -> available (bool)
func (m *Manager) ValidateServers() map[string]bool {
	results := make(map[string]bool)

	for lang, config := range m.configs {
		_, err := exec.LookPath(config.Command)
		results[lang] = (err == nil)
	}

	return results
}

// Shutdown shuts down all LSP clients
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for lang, client := range m.clients {
		if err := client.Shutdown(ctx); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to shutdown %s client: %w", lang, err)
			}
		}
	}

	return firstErr
}

// Close closes all LSP clients
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for lang, client := range m.clients {
		if err := client.Close(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to close %s client: %w", lang, err)
			}
		}
	}

	return firstErr
}
