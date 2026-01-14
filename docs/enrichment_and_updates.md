# Enrichment, Training, and Real-Time Systems

## How Ads are "Trained" for Semantic Matching

In this system, we don't "train" a model from scratch every time. Instead, we use a **Teacher-Student** approach to "Enrich" data.

### 1. The Raw Input
An advertiser submits a basic ad:
- **Title**: "Beach Hotel stay"
- **Desc**: "Good rooms, nice view."

### 2. The "Teacher" (LLM/DSPy)
We pass this raw text to a Large Language Model (Llama-3 running on Ollama) via DSPy. The LLM acts as an expert marketer.
- **Question**: "What is the intent of this ad? specific tags? quality score?"
- **Enriched Output**:
  - **Intent**: "Luxury Vacation", "Relaxation", "Oceanfront"
  - **Implied Category**: `Travel > Luxury`
  - **Expanded Text**: "Beach Hotel stay luxury vacation oceanfront view relaxation"

### 3. The "Student" (Embedding)
We run the **Expanded Text** through our Embedding Model (`all-MiniLM-L6-v2`).
- The vector produced now contains the mathematical concept of "Luxury" and "Relaxation", even though the original ad only said "Nice view".
- **Result**: Users searching for "Premium holiday" will matches this ad, even without keyword overlap.

---

## Real-Time Updates & NATS

Updating ads in real-time without downtime requires an Event-Driven Architecture.

### The Flow
1. **Ad Created/Updated**: API receives new ad.
2. **Publish Event**: API publishes event `ads.new` to **NATS**.
3. **Subscribers React**:
   - **Subscriber A (Python Enricher)**: Picks up the ad, runs LLM enrichment, generates vector, upserts to **Qdrant**.
   - **Subscriber B (Go Gateway)**: Picks up the ad, adds it to the **local In-Memory BM25 index**.

### Why NATS?
NATS is chosen over Kafka for this internal signaling because:
- **Latency**: NATS is faster for request-reply patterns.
- **Simplicity**: No Zookeeper, single binary.
- **JetStream**: Guarantees delivery even if a service is temporarily down.

---

## The Role of Redis

Redis is currently configured but can be leveraged for:

1.  **Semantic Caching (The 80/20 Rule)**:
    - 20% of queries account for 80% of traffic.
    - **Logic**: Before calling Qdrant (which takes 5-10ms), check Redis (0.5ms).
    - `Key`: Hash of query ("cheap flights") -> `Value`: IDs of top 20 ads.

2.  **User Frequency Capping**:
    - "Don't show Ad X to User Y more than 3 times per hour."
    - Redis `INCR` and `EXPIRE` are perfect for this high-speed logic.

3.  **Bid Floor Configuration**:
    - Store dynamic bid floors per category in Redis. Update them in real-time without redeploying the Gateway.
