#!/usr/bin/env python3
"""
Index Mock Ads into Qdrant with Semantic Embeddings

This script:
1. Loads mock ads from JSON
2. Generates 384-dim embeddings using sentence-transformers
3. Indexes into Qdrant for dense retrieval
"""

import json
import sys
from pathlib import Path

# Check for required packages
try:
    from sentence_transformers import SentenceTransformer
    from qdrant_client import QdrantClient
    from qdrant_client.models import Distance, VectorParams, PointStruct
except ImportError:
    print("Installing required packages...")
    import subprocess
    subprocess.check_call([sys.executable, "-m", "pip", "install", 
                          "sentence-transformers", "qdrant-client", "-q"])
    from sentence_transformers import SentenceTransformer
    from qdrant_client import QdrantClient
    from qdrant_client.models import Distance, VectorParams, PointStruct

# Configuration
QDRANT_HOST = "localhost"
QDRANT_PORT = 6333
COLLECTION_NAME = "ads"
EMBEDDING_MODEL = "all-MiniLM-L6-v2"  # 384 dimensions
VECTOR_SIZE = 384
BATCH_SIZE = 100

def main():
    print("🚀 Starting Qdrant indexing...")
    
    # Load mock ads
    ads_path = Path(__file__).parent.parent / "data" / "mock_ads.json"
    with open(ads_path) as f:
        ads = json.load(f)
    print(f"📦 Loaded {len(ads)} ads")
    
    # Initialize embedding model
    print(f"🧠 Loading embedding model: {EMBEDDING_MODEL}")
    model = SentenceTransformer(EMBEDDING_MODEL)
    
    # Connect to Qdrant
    print(f"🔗 Connecting to Qdrant at {QDRANT_HOST}:{QDRANT_PORT}")
    client = QdrantClient(host=QDRANT_HOST, port=QDRANT_PORT)
    
    # Create collection (delete if exists)
    try:
        client.delete_collection(COLLECTION_NAME)
        print(f"🗑️  Deleted existing collection '{COLLECTION_NAME}'")
    except Exception:
        pass
    
    client.create_collection(
        collection_name=COLLECTION_NAME,
        vectors_config=VectorParams(size=VECTOR_SIZE, distance=Distance.COSINE),
    )
    print(f"✅ Created collection '{COLLECTION_NAME}'")
    
    # Index ads in batches
    total_indexed = 0
    for i in range(0, len(ads), BATCH_SIZE):
        batch = ads[i:i + BATCH_SIZE]
        
        # Generate text for embedding
        texts = [
            f"{ad['title']} {ad['description']} {' '.join(ad.get('tags', []))}"
            for ad in batch
        ]
        
        # Generate embeddings
        embeddings = model.encode(texts)
        
        # Create points
        points = []
        for j, (ad, embedding) in enumerate(zip(batch, embeddings)):
            point = PointStruct(
                id=i + j,  # Use integer ID
                vector=embedding.tolist(),
                payload={
                    "id": ad["id"],
                    "title": ad["title"],
                    "description": ad["description"],
                    "category": ad["category"],
                    "advertiser_id": ad["advertiser_id"],
                    "bid_price": ad["bid_price"],
                    "quality_score": ad.get("quality_score", 0.5),
                    "tags": ad.get("tags", []),
                }
            )
            points.append(point)
        
        # Upsert to Qdrant
        client.upsert(collection_name=COLLECTION_NAME, points=points)
        total_indexed += len(points)
        print(f"📊 Indexed {total_indexed}/{len(ads)} ads...")
    
    # Verify
    info = client.get_collection(COLLECTION_NAME)
    print(f"\n✅ Indexing complete!")
    print(f"   Collection: {COLLECTION_NAME}")
    print(f"   Points: {info.points_count}")
    print(f"   Vector size: {info.config.params.vectors.size}")
    
    # Test search
    print("\n🔍 Test search: 'modern technology software'")
    test_query = "modern technology software"
    query_vector = model.encode(test_query).tolist()
    
    results = client.search(
        collection_name=COLLECTION_NAME,
        query_vector=query_vector,
        limit=3,
    )
    
    print("Top 3 results:")
    for i, r in enumerate(results):
        print(f"  {i+1}. [{r.payload['category']}] {r.payload['title'][:40]}... (score: {r.score:.3f})")

if __name__ == "__main__":
    main()
