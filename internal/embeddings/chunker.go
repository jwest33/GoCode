package embeddings

import (
	"strings"
	"unicode"
)

// Chunk represents a piece of text with metadata
type Chunk struct {
	Text     string            // The actual text content
	FilePath string            // Source file path
	StartLine int              // Starting line number
	EndLine   int              // Ending line number
	Metadata  map[string]string // Additional metadata
}

// ChunkerConfig holds configuration for text chunking
type ChunkerConfig struct {
	MaxChunkSize    int  // Maximum characters per chunk
	OverlapSize     int  // Characters to overlap between chunks
	RespectCodeBlocks bool // Don't split inside code blocks
}

// DefaultChunkerConfig returns sensible defaults for code chunking
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		MaxChunkSize:    512,   // ~128 tokens for typical code
		OverlapSize:     64,    // ~16 tokens overlap
		RespectCodeBlocks: true,
	}
}

// Chunker splits text into chunks suitable for embedding
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new chunker with the given config
func NewChunker(config ChunkerConfig) *Chunker {
	return &Chunker{config: config}
}

// ChunkFile splits a file's content into chunks
func (c *Chunker) ChunkFile(filePath string, content string) []Chunk {
	lines := strings.Split(content, "\n")
	chunks := []Chunk{}

	currentChunk := strings.Builder{}
	currentStartLine := 0
	currentLine := 0

	for i, line := range lines {
		lineLen := len(line) + 1 // +1 for newline

		// Check if adding this line would exceed max chunk size
		if currentChunk.Len()+lineLen > c.config.MaxChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, Chunk{
				Text:      strings.TrimSpace(currentChunk.String()),
				FilePath:  filePath,
				StartLine: currentStartLine,
				EndLine:   currentLine,
				Metadata:  c.extractMetadata(currentChunk.String()),
			})

			// Start new chunk with overlap
			currentChunk.Reset()
			overlapStart := max(0, i-c.calculateOverlapLines(lines, i))
			for j := overlapStart; j < i; j++ {
				currentChunk.WriteString(lines[j])
				currentChunk.WriteString("\n")
			}
			currentStartLine = overlapStart
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentLine = i
	}

	// Add final chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Text:      strings.TrimSpace(currentChunk.String()),
			FilePath:  filePath,
			StartLine: currentStartLine,
			EndLine:   currentLine,
			Metadata:  c.extractMetadata(currentChunk.String()),
		})
	}

	return chunks
}

// calculateOverlapLines determines how many lines to include for overlap
func (c *Chunker) calculateOverlapLines(lines []string, currentIdx int) int {
	overlapChars := 0
	overlapLines := 0

	for i := currentIdx - 1; i >= 0 && overlapChars < c.config.OverlapSize; i-- {
		overlapChars += len(lines[i]) + 1
		overlapLines++
	}

	return overlapLines
}

// extractMetadata extracts useful metadata from chunk text
func (c *Chunker) extractMetadata(text string) map[string]string {
	metadata := make(map[string]string)

	// Detect language (simple heuristic)
	if strings.Contains(text, "func ") && strings.Contains(text, "package ") {
		metadata["language"] = "go"
	} else if strings.Contains(text, "def ") && strings.Contains(text, "import ") {
		metadata["language"] = "python"
	} else if strings.Contains(text, "function ") || strings.Contains(text, "const ") {
		metadata["language"] = "javascript"
	}

	// Detect if it's a function definition
	if strings.Contains(text, "func ") || strings.Contains(text, "def ") || strings.Contains(text, "function ") {
		metadata["type"] = "function"
	} else if strings.Contains(text, "class ") || strings.Contains(text, "struct ") || strings.Contains(text, "interface ") {
		metadata["type"] = "type_definition"
	} else if strings.Contains(text, "import ") || strings.Contains(text, "package ") {
		metadata["type"] = "imports"
	}

	// Extract function/class names (very simple pattern)
	if name := c.extractIdentifier(text, "func "); name != "" {
		metadata["name"] = name
	} else if name := c.extractIdentifier(text, "def "); name != "" {
		metadata["name"] = name
	} else if name := c.extractIdentifier(text, "class "); name != "" {
		metadata["name"] = name
	}

	return metadata
}

// extractIdentifier extracts an identifier after a keyword
func (c *Chunker) extractIdentifier(text string, keyword string) string {
	idx := strings.Index(text, keyword)
	if idx == -1 {
		return ""
	}

	// Skip the keyword
	rest := text[idx+len(keyword):]

	// Skip whitespace
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)

	// Extract identifier (alphanumeric + underscore)
	identifier := strings.Builder{}
	for _, ch := range rest {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			identifier.WriteRune(ch)
		} else {
			break
		}
	}

	return identifier.String()
}

// ChunkText is a simpler method for chunking arbitrary text
func (c *Chunker) ChunkText(text string) []string {
	chunks := []string{}
	currentChunk := strings.Builder{}

	words := strings.Fields(text)
	for _, word := range words {
		if currentChunk.Len()+len(word)+1 > c.config.MaxChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
