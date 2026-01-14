package ranking

import (
	"sort"

	"github.com/openrtb-semantic-matcher/pkg/models"
)

// Config holds ranking configuration
type Config struct {
	// Weight factors for scoring
	SemanticWeight float64
	BidPriceWeight float64
	QualityWeight  float64

	// Diversity settings
	MaxAdsPerAdvertiser int

	// Normalization bounds
	MaxBidPrice float64
}

// DefaultConfig returns default ranking configuration
func DefaultConfig() Config {
	return Config{
		SemanticWeight:      0.5,
		BidPriceWeight:      0.3,
		QualityWeight:       0.2,
		MaxAdsPerAdvertiser: 2,
		MaxBidPrice:         20.0, // For normalization
	}
}

// Ranker implements multi-factor scoring
type Ranker struct {
	config Config
}

// NewRanker creates a new ranker with config
func NewRanker(config Config) *Ranker {
	return &Ranker{config: config}
}

// Rank applies multi-factor scoring and diversity filters
func (r *Ranker) Rank(candidates []models.AdCandidate, topK int) []models.AdCandidate {
	if len(candidates) == 0 {
		return candidates
	}

	// Step 1: Calculate composite scores
	scored := r.scoreAll(candidates)

	// Step 2: Sort by composite score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].CompositeScore > scored[j].CompositeScore
	})

	// Step 3: Apply diversity filter
	diverse := r.applyDiversity(scored)

	// Step 4: Limit to topK
	if len(diverse) > topK {
		diverse = diverse[:topK]
	}

	// Convert back to AdCandidate
	result := make([]models.AdCandidate, len(diverse))
	for i, s := range diverse {
		result[i] = s.Candidate
		// Store the composite score in FusedScore for API response
		result[i].FusedScore = s.CompositeScore
	}

	return result
}

// ScoredCandidate holds candidate with computed scores
type ScoredCandidate struct {
	Candidate      models.AdCandidate
	NormalizedSemantic float64
	NormalizedBid      float64
	NormalizedQuality  float64
	CompositeScore     float64
}

// scoreAll calculates composite scores for all candidates
func (r *Ranker) scoreAll(candidates []models.AdCandidate) []ScoredCandidate {
	scored := make([]ScoredCandidate, len(candidates))

	// Find max values for normalization
	maxSemantic := 0.0
	maxBid := r.config.MaxBidPrice

	for _, c := range candidates {
		if c.SemanticScore > maxSemantic {
			maxSemantic = c.SemanticScore
		}
	}

	// Avoid division by zero
	if maxSemantic == 0 {
		maxSemantic = 1
	}

	for i, c := range candidates {
		// Normalize scores to 0-1 range
		normSemantic := c.SemanticScore / maxSemantic
		if c.SparseScore > 0 {
			// Boost if also has sparse match
			normSemantic = (normSemantic + 0.2) // Bonus for keyword match
			if normSemantic > 1 {
				normSemantic = 1
			}
		}

		normBid := c.BidPrice / maxBid
		if normBid > 1 {
			normBid = 1
		}

		normQuality := c.QualityScore
		if normQuality == 0 {
			normQuality = 0.5 // Default quality
		}

		// Calculate weighted composite score
		composite := (r.config.SemanticWeight * normSemantic) +
			(r.config.BidPriceWeight * normBid) +
			(r.config.QualityWeight * normQuality)

		scored[i] = ScoredCandidate{
			Candidate:          c,
			NormalizedSemantic: normSemantic,
			NormalizedBid:      normBid,
			NormalizedQuality:  normQuality,
			CompositeScore:     composite,
		}
	}

	return scored
}

// applyDiversity ensures max N ads per advertiser
func (r *Ranker) applyDiversity(scored []ScoredCandidate) []ScoredCandidate {
	if r.config.MaxAdsPerAdvertiser <= 0 {
		return scored
	}

	advertiserCount := make(map[string]int)
	result := make([]ScoredCandidate, 0, len(scored))

	for _, s := range scored {
		advID := s.Candidate.AdvertiserID
		if advertiserCount[advID] < r.config.MaxAdsPerAdvertiser {
			result = append(result, s)
			advertiserCount[advID]++
		}
	}

	return result
}

// RankWithPersonalization applies user-based adjustments
func (r *Ranker) RankWithPersonalization(candidates []models.AdCandidate, profile *UserProfile, topK int) []models.AdCandidate {
	if profile == nil {
		return r.Rank(candidates, topK)
	}

	// Apply personalization boosts before scoring
	boosted := make([]models.AdCandidate, len(candidates))
	copy(boosted, candidates)

	for i := range boosted {
		// Boost preferred categories
		if contains(profile.PreferredCategories, boosted[i].Category) {
			boosted[i].SemanticScore *= 1.2 // 20% boost
		}

		// Boost preferred price range
		if boosted[i].BidPrice >= profile.MinBidPrice && boosted[i].BidPrice <= profile.MaxBidPrice {
			boosted[i].QualityScore += 0.1
		}
	}

	return r.Rank(boosted, topK)
}

// UserProfile represents user preferences for personalization
type UserProfile struct {
	UserID              string
	PreferredCategories []string
	MinBidPrice         float64
	MaxBidPrice         float64
	Location            string
	DeviceType          string
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
