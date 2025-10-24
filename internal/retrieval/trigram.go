package retrieval

import (
	"strings"
)

// TrigramIndex provides fuzzy string matching using trigrams
type TrigramIndex struct {
	trigrams map[string]map[string]bool // trigram -> set of docIDs
	documents map[string]*Document       // docID -> document
}

// NewTrigramIndex creates a new trigram index
func NewTrigramIndex() *TrigramIndex {
	return &TrigramIndex{
		trigrams:  make(map[string]map[string]bool),
		documents: make(map[string]*Document),
	}
}

// AddDocument adds a document to the trigram index
func (idx *TrigramIndex) AddDocument(doc Document) {
	idx.documents[doc.ID] = &doc

	// Extract trigrams from content
	trigrams := extractTrigrams(doc.Content)

	for trigram := range trigrams {
		if idx.trigrams[trigram] == nil {
			idx.trigrams[trigram] = make(map[string]bool)
		}
		idx.trigrams[trigram][doc.ID] = true
	}
}

// RemoveDocument removes a document from the index
func (idx *TrigramIndex) RemoveDocument(docID string) {
	doc, exists := idx.documents[docID]
	if !exists {
		return
	}

	// Remove from trigram index
	trigrams := extractTrigrams(doc.Content)
	for trigram := range trigrams {
		if idx.trigrams[trigram] != nil {
			delete(idx.trigrams[trigram], docID)
			if len(idx.trigrams[trigram]) == 0 {
				delete(idx.trigrams, trigram)
			}
		}
	}

	delete(idx.documents, docID)
}

// Search performs fuzzy search using trigram matching
func (idx *TrigramIndex) Search(query string, topK int) []ScoredDocument {
	queryTrigrams := extractTrigrams(query)
	if len(queryTrigrams) == 0 {
		return []ScoredDocument{}
	}

	// Count trigram matches per document
	docMatches := make(map[string]int)

	for trigram := range queryTrigrams {
		if docIDs, exists := idx.trigrams[trigram]; exists {
			for docID := range docIDs {
				docMatches[docID]++
			}
		}
	}

	// Calculate scores based on Jaccard similarity
	results := make([]ScoredDocument, 0, len(docMatches))

	for docID, matches := range docMatches {
		doc := idx.documents[docID]
		docTrigrams := extractTrigrams(doc.Content)

		// Jaccard similarity = intersection / union
		intersection := float32(matches)
		union := float32(len(queryTrigrams) + len(docTrigrams) - matches)

		score := intersection / union

		results = append(results, ScoredDocument{
			Document: *doc,
			Score:    score,
		})
	}

	// Sort by score
	sortScoredDocuments(results)

	// Return top K
	if topK < len(results) {
		results = results[:topK]
	}

	return results
}

// Count returns the number of indexed documents
func (idx *TrigramIndex) Count() int {
	return len(idx.documents)
}

// extractTrigrams extracts all trigrams from text
func extractTrigrams(text string) map[string]bool {
	text = strings.ToLower(text)
	trigrams := make(map[string]bool)

	// Pad text with spaces for boundary trigrams
	text = "  " + text + "  "

	// Extract all trigrams
	runes := []rune(text)
	for i := 0; i <= len(runes)-3; i++ {
		trigram := string(runes[i : i+3])
		// Only add if it contains at least one alphanumeric character
		if containsAlphanumeric(trigram) {
			trigrams[trigram] = true
		}
	}

	return trigrams
}

// containsAlphanumeric checks if a string contains at least one alphanumeric character
func containsAlphanumeric(s string) bool {
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			return true
		}
	}
	return false
}
