package dense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openrtb-semantic-matcher/pkg/models"
)

// QdrantClient handles vector search operations via HTTP
type QdrantClient struct {
	baseURL        string
	collectionName string
	httpClient     *http.Client
	embeddingURL   string // Python embedding service
	mockMode       bool
}

// NewQdrantClient creates a new Qdrant HTTP client
func NewQdrantClient(host string, port int, collection string, embeddingURL string, mock bool) (*QdrantClient, error) {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)

	client := &QdrantClient{
		baseURL:        baseURL,
		collectionName: collection,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		embeddingURL: embeddingURL,
		mockMode:     mock,
	}

	// Verify connection
	resp, err := client.httpClient.Get(baseURL + "/collections/" + collection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("collection %s not found", collection)
	}

	return client, nil
}

// SearchRequest is the Qdrant search request format
type SearchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

// SearchResponse is the Qdrant search response format
type SearchResponse struct {
	Result []SearchResult `json:"result"`
}

// SearchResult is a single search result from Qdrant
type SearchResult struct {
	ID      interface{}            `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// Search performs vector search using a pre-computed query vector
func (c *QdrantClient) SearchWithVector(ctx context.Context, queryVector []float32, topK int) ([]models.AdCandidate, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collectionName)

	reqBody := SearchRequest{
		Vector:      queryVector,
		Limit:       topK,
		WithPayload: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search failed: %s", string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	// Convert to AdCandidate
	candidates := make([]models.AdCandidate, 0, len(searchResp.Result))
	for _, r := range searchResp.Result {
		candidate := models.AdCandidate{
			ID:            getString(r.Payload, "id"),
			Title:         getString(r.Payload, "title"),
			Description:   getString(r.Payload, "description"),
			Category:      getString(r.Payload, "category"),
			AdvertiserID:  getString(r.Payload, "advertiser_id"),
			BidPrice:      getFloat(r.Payload, "bid_price"),
			SemanticScore: r.Score,
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// Search performs dense semantic search (embedding query text first)
// For now, we use a simple bag-of-words approximation
// In production, this would call the Python embedding service
func (c *QdrantClient) Search(ctx context.Context, query string, topK int) ([]models.AdCandidate, error) {
	// Try to get embedding from Python service
	embedding, err := c.getEmbedding(ctx, query)
	if err != nil {
		// Fallback: return empty results if embedding service unavailable
		return nil, fmt.Errorf("qdrant search failed: embedding service unavailable: %w", err)
	}

	return c.SearchWithVector(ctx, embedding, topK)
}

// getEmbedding calls the Python service to get query embedding
func (c *QdrantClient) getEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.mockMode {
		// Return random 384-dimensional vector
		vec := make([]float32, 384)
		for i := range vec {
			vec[i] = 0.05 // Static value to ensure deterministic testing or use rand
		}
		return vec, nil
	}

	reqBody := map[string]string{"text": text}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.embeddingURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embedding service returned %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedding, nil
}

// Close closes the client
func (c *QdrantClient) Close() error {
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
