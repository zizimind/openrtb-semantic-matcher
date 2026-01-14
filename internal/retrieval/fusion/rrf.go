package fusion

import (
	"sort"

	"github.com/openrtb-semantic-matcher/pkg/models"
)

// RRFFusion implements Reciprocal Rank Fusion
type RRFFusion struct {
	k int // RRF constant (typically 60)
}

// NewRRFFusion creates a new RRF fusion instance
func NewRRFFusion(k int) *RRFFusion {
	if k <= 0 {
		k = 60 // Standard default
	}
	return &RRFFusion{k: k}
}

// Merge combines results from sparse and dense retrieval using RRF
// Formula: score(d) = Σ 1/(k + rank(d)) for each ranking
func (f *RRFFusion) Merge(sparse, dense []models.AdCandidate) []models.AdCandidate {
	scores := make(map[string]*fusionEntry)

	// Process sparse results
	for rank, candidate := range sparse {
		entry := getOrCreate(scores, candidate)
		entry.sparseRank = rank + 1
		entry.sparseScore = candidate.SparseScore
		entry.rrfScore += 1.0 / float64(f.k+rank+1)
	}

	// Process dense results
	for rank, candidate := range dense {
		entry := getOrCreate(scores, candidate)
		entry.denseRank = rank + 1
		entry.denseScore = candidate.SemanticScore
		entry.rrfScore += 1.0 / float64(f.k+rank+1)
	}

	// Convert to sorted slice
	results := make([]models.AdCandidate, 0, len(scores))
	for _, entry := range scores {
		entry.candidate.FusedScore = entry.rrfScore
		entry.candidate.SparseScore = entry.sparseScore
		entry.candidate.SemanticScore = entry.denseScore
		results = append(results, entry.candidate)
	}

	// Sort by fused score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	return results
}

// fusionEntry holds intermediate fusion state
type fusionEntry struct {
	candidate   models.AdCandidate
	sparseRank  int
	denseRank   int
	sparseScore float64
	denseScore  float64
	rrfScore    float64
}

// getOrCreate retrieves or creates a fusion entry for a candidate
func getOrCreate(scores map[string]*fusionEntry, candidate models.AdCandidate) *fusionEntry {
	if entry, exists := scores[candidate.ID]; exists {
		return entry
	}

	entry := &fusionEntry{
		candidate: candidate,
	}
	scores[candidate.ID] = entry
	return entry
}

// WeightedRRFFusion allows custom weights for each retrieval source
type WeightedRRFFusion struct {
	k            int
	sparseWeight float64
	denseWeight  float64
}

// NewWeightedRRFFusion creates a weighted RRF fusion
func NewWeightedRRFFusion(k int, sparseWeight, denseWeight float64) *WeightedRRFFusion {
	return &WeightedRRFFusion{
		k:            k,
		sparseWeight: sparseWeight,
		denseWeight:  denseWeight,
	}
}

// Merge combines results with weighted RRF
func (f *WeightedRRFFusion) Merge(sparse, dense []models.AdCandidate) []models.AdCandidate {
	scores := make(map[string]*fusionEntry)

	// Process sparse with weight
	for rank, candidate := range sparse {
		entry := getOrCreate(scores, candidate)
		entry.sparseRank = rank + 1
		entry.sparseScore = candidate.SparseScore
		entry.rrfScore += f.sparseWeight * (1.0 / float64(f.k+rank+1))
	}

	// Process dense with weight
	for rank, candidate := range dense {
		entry := getOrCreate(scores, candidate)
		entry.denseRank = rank + 1
		entry.denseScore = candidate.SemanticScore
		entry.rrfScore += f.denseWeight * (1.0 / float64(f.k+rank+1))
	}

	// Convert and sort
	results := make([]models.AdCandidate, 0, len(scores))
	for _, entry := range scores {
		entry.candidate.FusedScore = entry.rrfScore
		results = append(results, entry.candidate)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FusedScore > results[j].FusedScore
	})

	return results
}
