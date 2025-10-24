package retrieval

import (
	"math"
	"strings"
	"unicode"
)

// BM25Params holds tuning parameters for BM25
type BM25Params struct {
	K1 float64 // Term frequency saturation (typical: 1.2-2.0)
	B  float64 // Length normalization (typical: 0.75)
}

// DefaultBM25Params returns standard BM25 parameters
func DefaultBM25Params() BM25Params {
	return BM25Params{
		K1: 1.5,
		B:  0.75,
	}
}

// Document represents a document in the BM25 index
type Document struct {
	ID       string
	Content  string
	FilePath string
	Metadata map[string]string
}

// BM25Index implements the BM25 ranking algorithm
type BM25Index struct {
	params      BM25Params
	documents   map[string]*Document
	termFreqs   map[string]map[string]int     // term -> docID -> frequency
	docFreqs    map[string]int                 // term -> number of docs containing term
	docLengths  map[string]int                 // docID -> document length in terms
	avgDocLen   float64
	totalDocs   int
}

// NewBM25Index creates a new BM25 index
func NewBM25Index(params BM25Params) *BM25Index {
	return &BM25Index{
		params:     params,
		documents:  make(map[string]*Document),
		termFreqs:  make(map[string]map[string]int),
		docFreqs:   make(map[string]int),
		docLengths: make(map[string]int),
	}
}

// AddDocument indexes a document
func (idx *BM25Index) AddDocument(doc Document) {
	// Tokenize document
	tokens := tokenize(doc.Content)

	// Calculate term frequencies for this document
	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// Update global structures
	idx.documents[doc.ID] = &doc
	idx.docLengths[doc.ID] = len(tokens)

	for term, freq := range termFreq {
		if idx.termFreqs[term] == nil {
			idx.termFreqs[term] = make(map[string]int)
		}
		idx.termFreqs[term][doc.ID] = freq

		// Track document frequency (how many docs contain this term)
		if freq > 0 {
			idx.docFreqs[term]++
		}
	}

	// Update statistics
	idx.totalDocs++
	idx.updateAvgDocLen()
}

// RemoveDocument removes a document from the index
func (idx *BM25Index) RemoveDocument(docID string) {
	doc, exists := idx.documents[docID]
	if !exists {
		return
	}

	// Tokenize to get terms
	tokens := tokenize(doc.Content)
	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// Update term frequencies
	for term := range termFreq {
		if idx.termFreqs[term] != nil {
			delete(idx.termFreqs[term], docID)
			if len(idx.termFreqs[term]) == 0 {
				delete(idx.termFreqs, term)
			}
			idx.docFreqs[term]--
			if idx.docFreqs[term] <= 0 {
				delete(idx.docFreqs, term)
			}
		}
	}

	delete(idx.documents, docID)
	delete(idx.docLengths, docID)
	idx.totalDocs--
	idx.updateAvgDocLen()
}

// Search performs BM25 search
func (idx *BM25Index) Search(query string, topK int) []ScoredDocument {
	queryTerms := tokenize(query)
	scores := make(map[string]float64)

	// Calculate BM25 score for each document
	for _, term := range queryTerms {
		idf := idx.calculateIDF(term)

		// For each document containing this term
		if docTermFreqs, exists := idx.termFreqs[term]; exists {
			for docID, termFreq := range docTermFreqs {
				docLen := idx.docLengths[docID]

				// BM25 formula
				numerator := float64(termFreq) * (idx.params.K1 + 1)
				denominator := float64(termFreq) + idx.params.K1*(1-idx.params.B+idx.params.B*(float64(docLen)/idx.avgDocLen))

				scores[docID] += idf * (numerator / denominator)
			}
		}
	}

	// Convert to sorted results
	results := make([]ScoredDocument, 0, len(scores))
	for docID, score := range scores {
		results = append(results, ScoredDocument{
			Document: *idx.documents[docID],
			Score:    float32(score),
		})
	}

	// Sort by score descending
	sortScoredDocuments(results)

	// Return top K
	if topK < len(results) {
		results = results[:topK]
	}

	return results
}

// calculateIDF computes Inverse Document Frequency
func (idx *BM25Index) calculateIDF(term string) float64 {
	if idx.totalDocs == 0 {
		return 0
	}

	df := idx.docFreqs[term]
	if df == 0 {
		return 0
	}

	// IDF = log((N - df + 0.5) / (df + 0.5) + 1)
	// Using smoothed IDF to avoid negative values
	return math.Log((float64(idx.totalDocs)-float64(df)+0.5)/(float64(df)+0.5) + 1.0)
}

// updateAvgDocLen recalculates average document length
func (idx *BM25Index) updateAvgDocLen() {
	if idx.totalDocs == 0 {
		idx.avgDocLen = 0
		return
	}

	totalLen := 0
	for _, len := range idx.docLengths {
		totalLen += len
	}
	idx.avgDocLen = float64(totalLen) / float64(idx.totalDocs)
}

// Count returns the number of indexed documents
func (idx *BM25Index) Count() int {
	return idx.totalDocs
}

// tokenize splits text into tokens (words)
func tokenize(text string) []string {
	// Convert to lowercase and split on non-alphanumeric
	text = strings.ToLower(text)

	tokens := []string{}
	currentToken := strings.Builder{}

	for _, ch := range text {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			currentToken.WriteRune(ch)
		} else {
			if currentToken.Len() > 0 {
				token := currentToken.String()
				// Filter out very short tokens and common stop words
				if len(token) > 1 && !isStopWord(token) {
					tokens = append(tokens, token)
				}
				currentToken.Reset()
			}
		}
	}

	// Add last token
	if currentToken.Len() > 0 {
		token := currentToken.String()
		if len(token) > 1 && !isStopWord(token) {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// isStopWord checks if a word is a common stop word
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "as": true, "by": true, "is": true,
		"was": true, "are": true, "be": true, "this": true, "that": true,
		"it": true, "with": true, "from": true, "have": true, "has": true,
	}
	return stopWords[word]
}
