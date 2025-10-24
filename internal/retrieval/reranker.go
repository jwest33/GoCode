package retrieval

import (
	"strings"
)

// Reranker reorders retrieved results based on heuristics
type Reranker struct {
	// Can be extended to use a cross-encoder model in the future
}

// NewReranker creates a new reranker
func NewReranker() *Reranker {
	return &Reranker{}
}

// RerankingFeatures extracts features for reranking
type RerankingFeatures struct {
	ExactMatch       bool    // Query exactly matches content
	FileRelevance    float32 // Importance based on file type/location
	Recency          float32 // How recently the file was modified
	SymbolImportance float32 // If it contains important symbols (func, class, etc.)
	QueryTermDensity float32 // Density of query terms in the chunk
	ChunkPosition    float32 // Position in file (beginning is more important)
}

// Rerank reorders results using heuristic scoring
func (r *Reranker) Rerank(results []FusedResult, query string, maxResults int) []FusedResult {
	queryLower := strings.ToLower(query)
	queryTerms := tokenize(query)

	// Calculate reranking scores
	for i := range results {
		features := r.extractFeatures(results[i], queryLower, queryTerms)
		boostScore := r.calculateBoost(features)

		// Apply boost to final score
		results[i].FinalScore *= (1.0 + boostScore)
	}

	// Re-sort after applying boosts
	sortFusedResults(results)

	// Return top results
	if maxResults < len(results) {
		results = results[:maxResults]
	}

	return results
}

// extractFeatures extracts reranking features from a result
func (r *Reranker) extractFeatures(result FusedResult, queryLower string, queryTerms []string) RerankingFeatures {
	contentLower := strings.ToLower(result.Document.Content)

	features := RerankingFeatures{}

	// Exact match
	features.ExactMatch = strings.Contains(contentLower, queryLower)

	// File relevance based on path
	features.FileRelevance = r.calculateFileRelevance(result.Document.FilePath)

	// Symbol importance
	features.SymbolImportance = r.calculateSymbolImportance(result.Document.Content, result.Document.Metadata)

	// Query term density
	features.QueryTermDensity = r.calculateTermDensity(contentLower, queryTerms)

	// Chunk position (from metadata if available)
	features.ChunkPosition = r.calculateChunkPosition(result.Document.Metadata)

	return features
}

// calculateBoost computes a boost factor from features
func (r *Reranker) calculateBoost(features RerankingFeatures) float32 {
	boost := float32(0.0)

	if features.ExactMatch {
		boost += 0.3 // 30% boost for exact match
	}

	boost += features.FileRelevance * 0.2        // Up to 20% from file relevance
	boost += features.SymbolImportance * 0.25    // Up to 25% from symbol importance
	boost += features.QueryTermDensity * 0.15    // Up to 15% from term density
	boost += features.ChunkPosition * 0.1        // Up to 10% from position

	return boost
}

// calculateFileRelevance scores file based on path and type
func (r *Reranker) calculateFileRelevance(filePath string) float32 {
	score := float32(0.5) // Base score

	// Boost for main source files
	if strings.Contains(filePath, "/src/") || strings.Contains(filePath, "\\src\\") {
		score += 0.2
	}

	// Boost for main package files (not test/vendor)
	if !strings.Contains(filePath, "_test") &&
	   !strings.Contains(filePath, "/vendor/") &&
	   !strings.Contains(filePath, "/node_modules/") {
		score += 0.2
	}

	// Penalize deeply nested files
	depth := strings.Count(filePath, "/") + strings.Count(filePath, "\\")
	if depth > 5 {
		score -= float32(depth-5) * 0.05
	}

	// Boost for certain important files
	if strings.Contains(filePath, "main.") || strings.Contains(filePath, "index.") {
		score += 0.1
	}

	// Clamp to [0, 1]
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// calculateSymbolImportance scores based on important code symbols
func (r *Reranker) calculateSymbolImportance(content string, metadata map[string]string) float32 {
	score := float32(0.3) // Base score

	// Check metadata for type
	if metadata != nil {
		typeInfo := metadata["type"]
		switch typeInfo {
		case "function":
			score += 0.4
		case "type_definition":
			score += 0.5
		case "imports":
			score += 0.2
		}

		// If metadata has a name, it's likely important
		if metadata["name"] != "" {
			score += 0.2
		}
	}

	// Heuristic checks if metadata not available
	if strings.Contains(content, "func ") || strings.Contains(content, "def ") || strings.Contains(content, "function ") {
		score += 0.2
	}

	if strings.Contains(content, "class ") || strings.Contains(content, "interface ") || strings.Contains(content, "struct ") {
		score += 0.2
	}

	// Clamp to [0, 1]
	if score > 1 {
		score = 1
	}

	return score
}

// calculateTermDensity measures how dense query terms are in content
func (r *Reranker) calculateTermDensity(contentLower string, queryTerms []string) float32 {
	if len(queryTerms) == 0 {
		return 0
	}

	contentTokens := tokenize(contentLower)
	if len(contentTokens) == 0 {
		return 0
	}

	matchCount := 0
	for _, token := range contentTokens {
		for _, queryTerm := range queryTerms {
			if token == queryTerm {
				matchCount++
				break
			}
		}
	}

	density := float32(matchCount) / float32(len(contentTokens))

	// Normalize to reasonable range (density rarely exceeds 0.1)
	if density > 0.1 {
		density = 0.1
	}

	return density * 10 // Scale to [0, 1]
}

// calculateChunkPosition scores based on position in file
func (r *Reranker) calculateChunkPosition(metadata map[string]string) float32 {
	// If metadata indicates this is near the beginning, boost it
	// For now, use simple heuristic
	if metadata != nil {
		typeInfo := metadata["type"]
		if typeInfo == "imports" {
			return 0.3 // Imports are usually at top but less important for content
		}
	}

	// Default: assume middle of file
	return 0.5
}

// OrderContext orders retrieved chunks optimally for prompting
// Critical chunks at top and bottom to avoid "lost in the middle" effect
func OrderContext(chunks []string, maxChunks int) []string {
	if len(chunks) <= maxChunks {
		return chunks
	}

	// Take top half and bottom half
	halfMax := maxChunks / 2
	topChunks := chunks[:halfMax]
	bottomChunks := chunks[len(chunks)-halfMax:]

	// Interleave or just concatenate
	ordered := make([]string, 0, maxChunks)
	ordered = append(ordered, topChunks...)
	ordered = append(ordered, bottomChunks...)

	return ordered
}
