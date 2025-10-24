package embeddings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager coordinates embedding operations
type Manager struct {
	client      *Client
	vectorStore *VectorStore
	chunker     *Chunker
}

// Config holds configuration for the embeddings manager
type Config struct {
	EmbeddingEndpoint string // Local embedding server endpoint
	EmbeddingDim      int    // Embedding dimension
	VectorDBPath      string // Path to SQLite vector database
	ChunkerConfig     ChunkerConfig
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		EmbeddingEndpoint: "http://localhost:8081", // Separate from LLM server
		EmbeddingDim:      384,                     // Common for nomic-embed-text
		VectorDBPath:      "embeddings.db",
		ChunkerConfig:     DefaultChunkerConfig(),
	}
}

// NewManager creates a new embeddings manager
func NewManager(config Config) (*Manager, error) {
	client := NewClient(config.EmbeddingEndpoint, config.EmbeddingDim)

	// Check if embedding server is available
	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		return nil, fmt.Errorf("embedding server not available at %s: %w", config.EmbeddingEndpoint, err)
	}

	vectorStore, err := NewVectorStore(config.VectorDBPath, config.EmbeddingDim)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	chunker := NewChunker(config.ChunkerConfig)

	return &Manager{
		client:      client,
		vectorStore: vectorStore,
		chunker:     chunker,
	}, nil
}

// IndexFile processes and indexes a file
func (m *Manager) IndexFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Delete existing chunks for this file
	if err := m.vectorStore.DeleteByFilePath(filePath); err != nil {
		return fmt.Errorf("failed to delete existing chunks: %w", err)
	}

	// Chunk the file
	chunks := m.chunker.ChunkFile(filePath, string(content))
	if len(chunks) == 0 {
		return nil // Empty file or no chunkable content
	}

	// Extract text from chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}

	// Generate embeddings
	embeddings, err := m.client.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Store in vector store
	if err := m.vectorStore.AddBatch(chunks, embeddings); err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	return nil
}

// IndexDirectory recursively indexes all files in a directory
func (m *Manager) IndexDirectory(ctx context.Context, dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-code files
		if info.IsDir() {
			// Skip common directories
			dirName := filepath.Base(path)
			if dirName == ".git" || dirName == "node_modules" || dirName == "vendor" ||
			   dirName == ".gocode" || dirName == "logs" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only index code files
		if !isCodeFile(path) {
			return nil
		}

		// Skip large files (> 1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		fmt.Printf("Indexing: %s\n", path)
		return m.IndexFile(ctx, path)
	})
}

// Search searches for semantically similar code
func (m *Manager) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// Generate embedding for query
	queryEmbedding, err := m.client.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search vector store
	results, err := m.vectorStore.Search(queryEmbedding, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
}

// Stats returns statistics about the indexed content
func (m *Manager) Stats() map[string]interface{} {
	return map[string]interface{}{
		"total_chunks": m.vectorStore.Count(),
		"dimension":    m.client.Dimension(),
	}
}

// Close closes the manager and releases resources
func (m *Manager) Close() error {
	return m.vectorStore.Close()
}

// isCodeFile checks if a file is a code file based on extension
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExtensions := map[string]bool{
		".go":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".tsx":  true,
		".jsx":  true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
		".rs":   true,
		".rb":   true,
		".php":  true,
		".cs":   true,
		".swift": true,
		".kt":   true,
		".scala": true,
		".sql":  true,
		".sh":   true,
		".bash": true,
		".yaml": true,
		".yml":  true,
		".json": true,
		".xml":  true,
		".md":   true,
	}
	return codeExtensions[ext]
}
