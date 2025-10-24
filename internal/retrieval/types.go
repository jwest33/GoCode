package retrieval

import "sort"

// ScoredDocument represents a document with a relevance score
type ScoredDocument struct {
	Document Document
	Score    float32
}

// sortScoredDocuments sorts documents by score in descending order
func sortScoredDocuments(docs []ScoredDocument) {
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score > docs[j].Score
	})
}

// FusedResult represents a result from multiple retrievers
type FusedResult struct {
	Document      Document
	FinalScore    float32
	BM25Score     float32
	SemanticScore float32
	TrigramScore  float32
}

// sortFusedResults sorts fused results by final score descending
func sortFusedResults(results []FusedResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})
}
