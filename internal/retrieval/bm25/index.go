package bm25

import (
	"encoding/gob"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/openrtb-semantic-matcher/pkg/models"
)

// Index holds the BM25 search index
type Index struct {
	Documents  []Document
	AvgDocLen  float64
	DocCount   int
	TermFreqs  map[string]map[int]int // term -> docID -> frequency
	DocFreqs   map[string]int         // term -> document frequency
	DocLengths map[int]int            // docID -> document length
	K1         float64                // BM25 parameter
	B          float64                // BM25 parameter
	mu         sync.RWMutex
}

// Document represents an indexed document
type Document struct {
	ID          string
	Title       string
	Content     string
	Category    string
	BidPrice    float64
	AdvertiserID string
}

// SearchResult holds a search result with score
type SearchResult struct {
	DocID int
	Score float64
}

// NewIndex creates a new BM25 index from file
func NewIndex(path string) (*Index, error) {
	idx := NewEmptyIndex()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(idx); err != nil {
		return nil, err
	}

	return idx, nil
}

// NewEmptyIndex creates an empty BM25 index
func NewEmptyIndex() *Index {
	return &Index{
		Documents:  make([]Document, 0),
		TermFreqs:  make(map[string]map[int]int),
		DocFreqs:   make(map[string]int),
		DocLengths: make(map[int]int),
		K1:         1.5,
		B:          0.75,
	}
}

// AddDocument adds a document to the index
func (idx *Index) AddDocument(doc Document) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docID := len(idx.Documents)
	idx.Documents = append(idx.Documents, doc)

	// Tokenize and index
	tokens := tokenize(doc.Title + " " + doc.Content)
	idx.DocLengths[docID] = len(tokens)

	termCounts := make(map[string]int)
	for _, token := range tokens {
		termCounts[token]++
	}

	for term, count := range termCounts {
		if idx.TermFreqs[term] == nil {
			idx.TermFreqs[term] = make(map[int]int)
		}
		idx.TermFreqs[term][docID] = count
		idx.DocFreqs[term]++
	}

	// Update statistics
	idx.DocCount = len(idx.Documents)
	idx.updateAvgDocLen()
}

// Search performs BM25 search and returns top-k results
func (idx *Index) Search(query string, topK int) []models.AdCandidate {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.DocCount == 0 {
		return nil
	}

	tokens := tokenize(query)
	scores := make(map[int]float64)

	for _, term := range tokens {
		if docFreqs, exists := idx.TermFreqs[term]; exists {
			idf := idx.calcIDF(term)

			for docID, tf := range docFreqs {
				docLen := float64(idx.DocLengths[docID])
				tfScore := (float64(tf) * (idx.K1 + 1)) /
					(float64(tf) + idx.K1*(1-idx.B+idx.B*(docLen/idx.AvgDocLen)))

				scores[docID] += idf * tfScore
			}
		}
	}

	// Sort by score
	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, SearchResult{DocID: docID, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}

	// Convert to AdCandidate
	candidates := make([]models.AdCandidate, 0, len(results))
	for _, r := range results {
		if r.DocID < len(idx.Documents) {
			doc := idx.Documents[r.DocID]
			candidates = append(candidates, models.AdCandidate{
				ID:           doc.ID,
				Title:        doc.Title,
				Description:  doc.Content,
				Category:     doc.Category,
				BidPrice:     doc.BidPrice,
				AdvertiserID: doc.AdvertiserID,
				SparseScore:  r.Score,
			})
		}
	}

	return candidates
}

// calcIDF calculates Inverse Document Frequency
func (idx *Index) calcIDF(term string) float64 {
	df := float64(idx.DocFreqs[term])
	if df == 0 {
		return 0
	}
	n := float64(idx.DocCount)
	return (n - df + 0.5) / (df + 0.5)
}

// updateAvgDocLen updates average document length
func (idx *Index) updateAvgDocLen() {
	if idx.DocCount == 0 {
		idx.AvgDocLen = 0
		return
	}

	total := 0
	for _, length := range idx.DocLengths {
		total += length
	}
	idx.AvgDocLen = float64(total) / float64(idx.DocCount)
}

// Save persists the index to disk
func (idx *Index) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(idx)
}

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	text = strings.ToLower(text)

	// Simple tokenization - split on non-alphanumeric
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// GetDocCount returns number of documents in index
func (idx *Index) GetDocCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.DocCount
}
