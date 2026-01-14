package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/openrtb-semantic-matcher/internal/retrieval/bm25"
)

func main() {
	// Load mock ads
	data, err := os.ReadFile("data/mock_ads.json")
	if err != nil {
		log.Fatalf("Failed to read mock ads: %v", err)
	}

	var ads []map[string]interface{}
	if err := json.Unmarshal(data, &ads); err != nil {
		log.Fatalf("Failed to parse mock ads: %v", err)
	}

	// Create BM25 index
	idx := bm25.NewEmptyIndex()

	for i, ad := range ads {
		bidPrice := 0.0
		if bp, ok := ad["bid_price"].(float64); ok {
			bidPrice = bp
		}
		
		doc := bm25.Document{
			ID:           ad["id"].(string),
			Title:        ad["title"].(string),
			Content:      ad["description"].(string),
			Category:     ad["category"].(string),
			BidPrice:     bidPrice,
			AdvertiserID: ad["advertiser_id"].(string),
		}
		idx.AddDocument(doc)

		if (i+1)%1000 == 0 {
			fmt.Printf("Indexed %d ads...\n", i+1)
		}
	}

	// Save index
	if err := idx.Save("data/bm25_index.gob"); err != nil {
		log.Fatalf("Failed to save index: %v", err)
	}

	fmt.Printf("✅ Indexed %d ads to data/bm25_index.gob\n", len(ads))
}
