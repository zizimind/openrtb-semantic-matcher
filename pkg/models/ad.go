package models

// AdCandidate represents a matched ad candidate
type AdCandidate struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Category       string   `json:"category"`
	AdvertiserID   string   `json:"advertiser_id"`
	BidPrice       float64  `json:"bid_price"`
	SemanticScore  float64  `json:"semantic_score"`
	SparseScore    float64  `json:"sparse_score"`
	FusedScore     float64  `json:"fused_score"`
	QualityScore   float64  `json:"quality_score"`
	Tags           []string `json:"tags,omitempty"`
	ImageURL       string   `json:"image_url,omitempty"`
	ClickURL       string   `json:"click_url,omitempty"`
	CreativeWidth  int      `json:"creative_w,omitempty"`
	CreativeHeight int      `json:"creative_h,omitempty"`
}

// MatchResponse is the API response for a match request
type MatchResponse struct {
	ID         string                 `json:"id"`
	Candidates []AdCandidate          `json:"candidates"`
	Latency    int64                  `json:"latency_ms"`
	Debug      map[string]interface{} `json:"debug,omitempty"`
	Metadata   *Metadata              `json:"metadata,omitempty"`
}

// Metadata contains additional response info
type Metadata struct {
	SparseHits int    `json:"sparse_hits"`
	DenseHits  int    `json:"dense_hits"`
	FusionType string `json:"fusion_type"`
}

// Ad represents a full ad record (for indexing)
type Ad struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	AdvertiserID    string   `json:"advertiser_id"`
	BidPrice        float64  `json:"bid_price"`
	QualityScore    float64  `json:"quality_score"`
	Tags            []string `json:"tags"`
	SemanticSummary string   `json:"semantic_summary,omitempty"`
	Embedding       []float32 `json:"embedding,omitempty"`
	ImageURL        string   `json:"image_url,omitempty"`
	ClickURL        string   `json:"click_url,omitempty"`
	CreativeWidth   int      `json:"creative_w,omitempty"`
	CreativeHeight  int      `json:"creative_h,omitempty"`
	Active          bool     `json:"active"`
	CreatedAt       int64    `json:"created_at"`
	UpdatedAt       int64    `json:"updated_at"`
}
