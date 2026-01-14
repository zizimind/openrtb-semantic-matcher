# Architecture Overview

## High-Level Diagram

```mermaid
graph TD
    User[User/DSP] -->|OpenRTB Request| Gateway[Go Gateway]
    
    subgraph "Online Fast Path (<70ms)"
        Gateway -->|Broadcast| NATS[NATS JetStream]
        Gateway -->|Async Log| Kafka[Kafka Logger]
        
        Gateway -->|Keywords| BM25[BM25 Index (In-Memory)]
        Gateway -->|Vector| Qdrant[Qdrant Vector DB]
        
        BM25 -->|Results| RRF[RRF Fusion]
        Qdrant -->|Results| RRF
        RRF -->|Ranked| Scorer[Multi-Factor Ranker]
    end
    
    subgraph "Offline Teacher (Enrichment)"
        Ads[Raw Ads] -->|Ingest| PyEnricher[Python Enricher]
        PyEnricher -->|DSPy Call| Ollama[Ollama (Llama-3)]
        Ollama -->|Semantic Tags| PyEnricher
        PyEnricher -->|Generate Emb| EmbedModel[Sentence Transformers]
        EmbedModel -->|Upsert| Qdrant
    end
```

## Core Components

### 1. Go Gateway (The "Student")
The entry point for all bid requests. It is designed for extreme speed (Low Latency).
- **Role**: Parses requests, coordinates search, ranks results, and responds.
- **Design**: 
    - No heavy ML inference in the hot path.
    - Parallel execution of BM25 and Qdrant queries.
    - "Fail-open" logic: If Qdrant is slow, return BM25 results instantly.

### 2. Python Enricher (The "Teacher")
The intelligence layer. It runs offline or asynchronously.
- **Role**: "Teaches" the system what an ad is really about.
- **Tools**: 
    - **DSPy**: Orchestrates calls to the LLM (Ollama) to extract "Golden metadata" (e.g., inferring "luxury status" or "user intent" from a simple ad title).
    - **Sentence-Transformers**: Converts text into 384-dimensional vectors.

### 3. Messaging Layer (The "Nervous System")
- **NATS JetStream**:
    - **Why?** Go-native, ultra-low latency.
    - **Use Case**: Decoupling. When a new ad is created, it's published to NATS. The Python Enricher subscribes, processes it, and updates Qdrant. The Go Gateway subscribes to update its local BM25 index.
- **Kafka**:
    - **Why?** High throughput, standard for data lakes.
    - **Use Case**: Logging raw bid requests/responses for analytics and later model training (fine-tuning).

### 4. Storage
- **Qdrant**: Stores dense vectors (semantic meaning).
- **Redis** (Planned): Stores cached results for popular queries (e.g., "iphone cases" doesn't need to be re-computed 1000 times/sec).
- **In-Memory (Go)**: Stores the Sparse (BM25) index for maximum speed (0 network latency).

### 6. All-Neural Stack (Future State)
The industry standard is moving towards replacing BM25 with **SPLADE** (Sparse Lexical and Expansion).

- **Dense (Sentence Transformers)**: Captures intent ("feline" matches "cat").
- **Sparse (SPLADE)**: Captures exact keywords + AI expansion ("Apple" matches "iPhone").
- **Fusion**: RRF merges the best of both worlds.

This eliminates the "Exact Match Blind Spot" of Dense vectors and the "Semantic Blind Spot" of BM25.

### 7. Cold Start Strategy (New Ads)
- **Multi-Factor Scoring**: Combines Meaning (50%), Money (30%), and Quality (20%).
- **Cold Start Strategy**:
    - New ads have no historical Quality Score (CTR).
    - **Solution**: We apply **Mean Imputation** (default score = 0.5/1.0).
    - **Result**: New ads get a fair chance to display. If they perform well, their real quality score rises; if poorly, it drops below 0.5.
