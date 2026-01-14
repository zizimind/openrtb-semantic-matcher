# Future Improvements & Roadmap

## 1. Evaluation: LLM-as-a-Judge
Currently, we verify results manually. We should automate this:
- **Pipeline**:
    1. Save query + top 3 results to a log.
    2. Offline script sends these to GPT-4 or Llama-3.
    3. **Prompt**: "User searched for 'X'. System returned Ad 'Y'. On a scale of 0-1, how relevant is this?"
    4. Compute **NDCG** (Normalized Discounted Cumulative Gain) score to track quality over time.

## 2. Personalization
Currently, we rank generally. We should personalize using **User Profiles**:
- **Ingest**: Listen to click-stream events via Kafka.
- **Profile**: Build user vectors (e.g., "User likes Tech & Travel").
- **Serving**:
    - When User X searches "Best deals", we mix their User Vector into the Query Vector.
    - `FinalVector = 0.7 * QueryVector + 0.3 * UserVector`

## 3. Production Hardening
- **Bloom Filters**:
    - **Use Case**: Prevent "Cache Penetration" (checking DB for keys that definitely don't exist).
    - **Use Case**: Frequency Capping (checking if user X has seen ad Y).
- **Early Gating (pre-filtering)**:
    - **Concept**: discarding "impossible" candidates cheaply before searching.
    - **Example**: "If User is in France, don't even embed query for US-only ads".
- **Distributed BM25**: Currently, BM25 is in-memory on each instance. For millions of ads, we need a distributed index (like Elasticsearch or a sharded Go setup).
- **Circuit Breakers**: If Qdrant fails, the system currently logs an error. We should implement automatic circuit breaking to stop trying Qdrant and degrade gracefully to BM25-only.
- **Kubernetes**: Move from Docker Compose to K8s for auto-scaling.

## 4. Advanced Retrieval (SPLADE)
- **Problem**: BM25 is purely keyword-based and misses context (synonyms), while Dense vectors miss exact matches.
- **Solution**: Implement **SPLADE** (Sparse Lexical and Expansion).
- **Benefit**: Replaces in-memory BM25 with Qdrant's Sparse Vector support, allowing us to offload index management and get better retrieval quality.

## 5. Advanced Ranking (Project "Lambda")
- Replace the simple linear formula (`0.5*Sem + 0.3*Bid`) with a **Learning-to-Rank (LTR)** model (XGBoost/LightGBM) trained on actual click data.
