package embeddings

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// VectorStore manages storage and retrieval of embeddings
type VectorStore struct {
	db         *sql.DB
	mu         sync.RWMutex
	inMemIndex map[string]*IndexedChunk // For fast similarity search
	dimension  int
}

// IndexedChunk represents a chunk with its embedding in memory
type IndexedChunk struct {
	ID        string
	Chunk     Chunk
	Embedding []float32
}

// SearchResult represents a search result with score
type SearchResult struct {
	Chunk      Chunk
	Score      float32
	FilePath   string
	StartLine  int
	EndLine    int
}

// NewVectorStore creates a new vector store backed by SQLite
func NewVectorStore(dbPath string, dimension int) (*VectorStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS chunks (
		id TEXT PRIMARY KEY,
		file_path TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		text TEXT NOT NULL,
		metadata TEXT,
		embedding BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_file_path ON chunks(file_path);
	CREATE INDEX IF NOT EXISTS idx_created_at ON chunks(created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	vs := &VectorStore{
		db:         db,
		inMemIndex: make(map[string]*IndexedChunk),
		dimension:  dimension,
	}

	// Load existing chunks into memory
	if err := vs.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return vs, nil
}

// Add stores a chunk with its embedding
func (vs *VectorStore) Add(id string, chunk Chunk, embedding []float32) error {
	if len(embedding) != vs.dimension {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(embedding), vs.dimension)
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Serialize metadata
	metadataJSON, err := json.Marshal(chunk.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Serialize embedding
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	// Insert into database
	_, err = vs.db.Exec(`
		INSERT OR REPLACE INTO chunks (id, file_path, start_line, end_line, text, metadata, embedding)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, chunk.FilePath, chunk.StartLine, chunk.EndLine, chunk.Text, string(metadataJSON), embeddingJSON)

	if err != nil {
		return fmt.Errorf("failed to insert chunk: %w", err)
	}

	// Update in-memory index
	vs.inMemIndex[id] = &IndexedChunk{
		ID:        id,
		Chunk:     chunk,
		Embedding: embedding,
	}

	return nil
}

// AddBatch stores multiple chunks with their embeddings
func (vs *VectorStore) AddBatch(chunks []Chunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch")
	}

	for i, chunk := range chunks {
		id := fmt.Sprintf("%s:%d-%d", chunk.FilePath, chunk.StartLine, chunk.EndLine)
		if err := vs.Add(id, chunk, embeddings[i]); err != nil {
			return fmt.Errorf("failed to add chunk %d: %w", i, err)
		}
	}

	return nil
}

// Search finds the most similar chunks to the query embedding
func (vs *VectorStore) Search(queryEmbedding []float32, topK int) ([]SearchResult, error) {
	if len(queryEmbedding) != vs.dimension {
		return nil, fmt.Errorf("query embedding dimension mismatch")
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Calculate similarity for all chunks
	scores := make([]struct {
		chunk *IndexedChunk
		score float32
	}, 0, len(vs.inMemIndex))

	for _, indexedChunk := range vs.inMemIndex {
		similarity := cosineSimilarity(queryEmbedding, indexedChunk.Embedding)
		scores = append(scores, struct {
			chunk *IndexedChunk
			score float32
		}{chunk: indexedChunk, score: similarity})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return top K results
	k := topK
	if k > len(scores) {
		k = len(scores)
	}

	results := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = SearchResult{
			Chunk:     scores[i].chunk.Chunk,
			Score:     scores[i].score,
			FilePath:  scores[i].chunk.Chunk.FilePath,
			StartLine: scores[i].chunk.Chunk.StartLine,
			EndLine:   scores[i].chunk.Chunk.EndLine,
		}
	}

	return results, nil
}

// DeleteByFilePath removes all chunks for a given file
func (vs *VectorStore) DeleteByFilePath(filePath string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Delete from database
	_, err := vs.db.Exec("DELETE FROM chunks WHERE file_path = ?", filePath)
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	// Remove from in-memory index
	for id, chunk := range vs.inMemIndex {
		if chunk.Chunk.FilePath == filePath {
			delete(vs.inMemIndex, id)
		}
	}

	return nil
}

// Count returns the total number of chunks
func (vs *VectorStore) Count() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.inMemIndex)
}

// loadIndex loads all chunks from database into memory
func (vs *VectorStore) loadIndex() error {
	rows, err := vs.db.Query(`
		SELECT id, file_path, start_line, end_line, text, metadata, embedding
		FROM chunks
	`)
	if err != nil {
		return fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, filePath, text, metadataJSON string
		var startLine, endLine int
		var embeddingJSON []byte

		if err := rows.Scan(&id, &filePath, &startLine, &endLine, &text, &metadataJSON, &embeddingJSON); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Deserialize metadata
		var metadata map[string]string
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Deserialize embedding
		var embedding []float32
		if err := json.Unmarshal(embeddingJSON, &embedding); err != nil {
			return fmt.Errorf("failed to unmarshal embedding: %w", err)
		}

		chunk := Chunk{
			Text:      text,
			FilePath:  filePath,
			StartLine: startLine,
			EndLine:   endLine,
			Metadata:  metadata,
		}

		vs.inMemIndex[id] = &IndexedChunk{
			ID:        id,
			Chunk:     chunk,
			Embedding: embedding,
		}
	}

	return rows.Err()
}

// Close closes the database connection
func (vs *VectorStore) Close() error {
	return vs.db.Close()
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
