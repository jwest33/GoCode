package retrieval

import (
	"context"
	"fmt"

	"github.com/jake/gocode/internal/embeddings"
)

// FusionWeights configures how different retrievers are weighted
type FusionWeights struct {
	BM25     float32 // Weight for keyword search
	Semantic float32 // Weight for semantic search
	Trigram  float32 // Weight for fuzzy matching
}

// DefaultFusionWeights returns balanced weights
func DefaultFusionWeights() FusionWeights {
	return FusionWeights{
		BM25:     0.4,
		Semantic: 0.5,
		Trigram:  0.1,
	}
}

// HybridRetriever combines multiple retrieval methods
type HybridRetriever struct {
	bm25Index      *BM25Index
	trigramIndex   *TrigramIndex
	embeddingsMgr  *embeddings.Manager
	weights        FusionWeights
	useSemanticSearch bool // Only if embeddings are available
}

// NewHybridRetriever creates a new hybrid retriever
func NewHybridRetriever(weights FusionWeights, embeddingsMgr *embeddings.Manager) *HybridRetriever {
	return &HybridRetriever{
		bm25Index:      NewBM25Index(DefaultBM25Params()),
		trigramIndex:   NewTrigramIndex(),
		embeddingsMgr:  embeddingsMgr,
		weights:        weights,
		useSemanticSearch: embeddingsMgr != nil,
	}
}

// AddDocument indexes a document across all retrievers
func (hr *HybridRetriever) AddDocument(ctx context.Context, doc Document) error {
	// Add to BM25
	hr.bm25Index.AddDocument(doc)

	// Add to trigram
	hr.trigramIndex.AddDocument(doc)

	// Add to semantic index if available
	if hr.useSemanticSearch && hr.embeddingsMgr != nil {
		// For semantic search, we need file path - assume doc.ID is file path
		if doc.FilePath != "" {
			if err := hr.embeddingsMgr.IndexFile(ctx, doc.FilePath); err != nil {
				return fmt.Errorf("failed to index file semantically: %w", err)
			}
		}
	}

	return nil
}

// RemoveDocument removes a document from all retrievers
func (hr *HybridRetriever) RemoveDocument(docID string) {
	hr.bm25Index.RemoveDocument(docID)
	hr.trigramIndex.RemoveDocument(docID)
	// Note: Semantic index removal would need to be implemented in embeddings.Manager
}

// Search performs hybrid search combining all methods
func (hr *HybridRetriever) Search(ctx context.Context, query string, topK int) ([]FusedResult, error) {
	// Retrieve from each method (get more than topK for better fusion)
	retrievalK := topK * 3

	// BM25 search
	bm25Results := hr.bm25Index.Search(query, retrievalK)

	// Trigram search
	trigramResults := hr.trigramIndex.Search(query, retrievalK)

	// Semantic search (if available)
	var semanticResults []embeddings.SearchResult
	if hr.useSemanticSearch && hr.embeddingsMgr != nil {
		var err error
		semanticResults, err = hr.embeddingsMgr.Search(ctx, query, retrievalK)
		if err != nil {
			// Log but don't fail - continue with other methods
			fmt.Printf("Warning: semantic search failed: %v\n", err)
		}
	}

	// Normalize scores to [0, 1] range
	bm25Scores := normalizeScores(bm25Results)
	trigramScores := normalizeScores(trigramResults)
	semanticScores := normalizeSemanticScores(semanticResults)

	// Fuse results
	fusedScores := make(map[string]*FusedResult)

	// Add BM25 results
	for docID, score := range bm25Scores {
		if fusedScores[docID] == nil {
			fusedScores[docID] = &FusedResult{
				Document: getDocumentByID(bm25Results, docID),
			}
		}
		fusedScores[docID].BM25Score = score
		fusedScores[docID].FinalScore += score * hr.weights.BM25
	}

	// Add trigram results
	for docID, score := range trigramScores {
		if fusedScores[docID] == nil {
			fusedScores[docID] = &FusedResult{
				Document: getDocumentByID(trigramResults, docID),
			}
		}
		fusedScores[docID].TrigramScore = score
		fusedScores[docID].FinalScore += score * hr.weights.Trigram
	}

	// Add semantic results
	for docID, score := range semanticScores {
		if fusedScores[docID] == nil {
			// Create document from semantic result
			fusedScores[docID] = &FusedResult{
				Document: getDocumentFromSemanticResult(semanticResults, docID),
			}
		}
		fusedScores[docID].SemanticScore = score
		fusedScores[docID].FinalScore += score * hr.weights.Semantic
	}

	// Convert to slice and sort
	results := make([]FusedResult, 0, len(fusedScores))
	for _, result := range fusedScores {
		results = append(results, *result)
	}

	sortFusedResults(results)

	// Return top K
	if topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

// normalizeScores normalizes scores to [0, 1] range
func normalizeScores(results []ScoredDocument) map[string]float32 {
	if len(results) == 0 {
		return make(map[string]float32)
	}

	// Find max score
	maxScore := results[0].Score
	if maxScore == 0 {
		maxScore = 1.0 // Avoid division by zero
	}

	normalized := make(map[string]float32)
	for _, result := range results {
		normalized[result.Document.ID] = result.Score / maxScore
	}

	return normalized
}

// normalizeSemanticScores normalizes semantic search scores
func normalizeSemanticScores(results []embeddings.SearchResult) map[string]float32 {
	if len(results) == 0 {
		return make(map[string]float32)
	}

	// Semantic scores are already cosine similarity in [0, 1]
	// But we normalize to the max in this batch for consistency
	maxScore := results[0].Score
	if maxScore == 0 {
		maxScore = 1.0
	}

	normalized := make(map[string]float32)
	for _, result := range results {
		// Use FilePath as ID for semantic results
		docID := fmt.Sprintf("%s:%d-%d", result.FilePath, result.StartLine, result.EndLine)
		normalized[docID] = result.Score / maxScore
	}

	return normalized
}

// getDocumentByID retrieves a document from scored results
func getDocumentByID(results []ScoredDocument, docID string) Document {
	for _, result := range results {
		if result.Document.ID == docID {
			return result.Document
		}
	}
	return Document{ID: docID} // Return empty if not found
}

// getDocumentFromSemanticResult converts a semantic result to a Document
func getDocumentFromSemanticResult(results []embeddings.SearchResult, docID string) Document {
	for _, result := range results {
		resultID := fmt.Sprintf("%s:%d-%d", result.FilePath, result.StartLine, result.EndLine)
		if resultID == docID {
			return Document{
				ID:       docID,
				Content:  result.Chunk.Text,
				FilePath: result.FilePath,
				Metadata: result.Chunk.Metadata,
			}
		}
	}
	return Document{ID: docID}
}

// Count returns the number of indexed documents
func (hr *HybridRetriever) Count() int {
	return hr.bm25Index.Count()
}
