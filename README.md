# OpenRTB Semantic Matcher

A high-performance, hybrid search advertising engine capable of **250+ QPS** (single node). It combines Go-native sparse retrieval (BM25) with dense semantic search (Qdrant), enriched by offline Large Language Models (DSPy/Ollama).

## 🚀 Key Features

- **Hybrid Search**: Fuses keyword matching (BM25) and semantic understanding (Vector Search) using Reciprocal Rank Fusion (RRF).
- **Latency Optimized**: "Fast Path" architecture in Go using **Fiber** (Zero Allocation Router).
- **LLM-Enhanced**: Offline "Teacher" (Llama-3 via DSPy) enriches ads with semantic metadata and intent tags.
- **Microsecond Caching**: Redis look-aside cache with TTL strategy for hot queries (< 1ms hits).
- **Real-Time Updates**: NATS JetStream event propagation for instant budget/status updates.
- **Async Logging**: Non-blocking ZSTD-compressed Kafka logging.

## 🛠️ Tech Stack

- **Gateway**: Go 1.21+, `prebid/openrtb`, `gofiber/fiber` (v2).
- **Search**: 
  - **Sparse**: In-memory BM25 (Go-native).
  - **Dense**: Qdrant Vector Database (HTTP).
- **Caching**: Redis (v7).
- **Enrichment**: Python, DSPy, Ollama, Sentence-Transformers.
- **Messaging**: NATS JetStream (Events), Kafka (Logging).
- **Infra**: Docker Compose.

## ⚡️ Quick Start

### 1. Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Python 3.10+

### 2. Start Infrastructure
```bash
make docker-up
# Starts: Qdrant, NATS, Kafka, Redis, Zookeeper
```

### 3. Generate Mock Data & Build
```bash
make mock-data  # Generates 10k realistic ads in data/mock_ads.json
make build      # Builds the Go Gateway
```

### 4. Run Services
**Terminal 1 (Python Enricher):**
```bash
cd python/enricher
pip install -r requirements.txt
python main.py
```

**Terminal 2 (Go Gateway):**
```bash
./bin/gateway
```

### 5. Index Data (One-time)
Call the python indexer to generate embeddings and push to Qdrant:
```bash
curl -X POST http://localhost:8001/index-from-file
```

### 6. Test a Query
```bash
curl -s -X POST http://localhost:8080/v1/match \
  -H "Content-Type: application/json" \
  -d '{"id": "test", "site": {"page": "luxury travel vacation"}}' | python3 -m json.tool
```

### 7. Real-Time Budget Update (NATS Demo)
Update an advertiser's budget instantly across all instances:
```bash
curl -X POST http://localhost:8080/v1/admin/budget \
  -H "Content-Type: application/json" \
  -d '{"advertiser_id": "nike", "new_budget": 5000.0}'
```

### 8. Debugging Logs
Consume and decompress binary Kafka logs in real-time:
```bash
# Listen to all logs
go run scripts/read_logs.go

# Search for specific request ID or keyword
go run scripts/read_logs.go bid.requests "nike"
```

## 🧠 Ranking & Personnelization

The engine doesn't just find ads; it ranks them to maximize revenue and user experience using a 3-stage pipeline:

### 1. Multi-Factor Scoring
Every candidate is scored using a weighted formula:
```go
Params: SemanticWeight=0.5, BidWeight=0.3, QualityWeight=0.2

FinalScore = (0.5 * SemanticSimilarity) + 
             (0.3 * NormalizedBidPrice) + 
             (0.2 * QualityScore)
```
*   **Semantic**: How well the ad text matches the query (Vector Search).
*   **Bid**: Normalized against a $20.00 max cap.
*   **Quality**: Historical CTR or advertiser reputation.

### 2. Personalization (Re-Ranking)
If a `UserProfile` is known, we boost ads **before** the final sort:
*   **Category Affinity**: +20% Semantic Score boost if user likes this category (e.g., "Shoes").
*   **Price Awareness**: +0.1 Quality Score if bid falls within user's typical purchase range.

### 3. Recommendation (Diversity)
To prevent one advertiser from spamming the results, we apply a **Diversity Filter**:
*   **Rule**: Max **2** ads per `AdvertiserID` in the top results.
*   **Effect**: Ensures users see a variety of brands.

## 📊 Understanding the Scores

The API returns detailed scoring information for debugging and transparency:

```json
{
  "semantic_score": 0.6911,  // Cosine similarity (0-1) from Vector Search (Meaning)
  "sparse_score": 0,         // BM25 score. 0 means keywords didn't effectively match.
  "bid_price": 1.40,         // Assessing the commercial value ($1.40 CPM)
  "fused_score": 0.602       // FINAL Ranking Score (0-1)
}
```

**The Final Formula:**
The `fused_score` is a weighted combination of multiple factors used to sort the results:
```go
FinalScore = (0.5 * Semantic) + (0.3 * BidPrice) + (0.2 * Quality)
```
*Note: Bid Price is normalized against a $20.00 max cap.*

## 📂 Documentation

- [Architecture Overview](docs/architecture.md) - High-level diagrams and component breakdown.
- [Enrichment & Real-time Updates](docs/enrichment_and_updates.md) - How ads are "trained" and updated.
- [Future Roadmap](docs/future_roadmap.md) - Improvements and next steps.
