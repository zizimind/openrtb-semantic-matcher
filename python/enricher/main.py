"""
OpenRTB Semantic Matcher - Python Enricher Service

This service provides:
1. LLM-based ad enrichment using DSPy + Ollama
2. Embedding generation using sentence-transformers
3. gRPC interface for the Go gateway
"""

import os
import json
import logging
from pathlib import Path
from contextlib import asynccontextmanager

import grpc
from concurrent import futures
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
QDRANT_HOST = os.getenv("QDRANT_HOST", "localhost")
QDRANT_PORT = int(os.getenv("QDRANT_PORT", 6333))
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")
COLLECTION_NAME = os.getenv("QDRANT_COLLECTION", "ads")
EMBEDDING_MODEL = "all-MiniLM-L6-v2"
VECTOR_SIZE = 384

# Global instances
embedding_model = None
qdrant_client = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize resources on startup."""
    global embedding_model, qdrant_client
    
    logger.info("Loading embedding model...")
    embedding_model = SentenceTransformer(EMBEDDING_MODEL)
    logger.info(f"Loaded {EMBEDDING_MODEL}")
    
    logger.info(f"Connecting to Qdrant at {QDRANT_HOST}:{QDRANT_PORT}...")
    qdrant_client = QdrantClient(host=QDRANT_HOST, port=QDRANT_PORT)
    
    # Ensure collection exists
    try:
        qdrant_client.get_collection(COLLECTION_NAME)
        logger.info(f"Collection '{COLLECTION_NAME}' exists")
    except Exception:
        logger.info(f"Creating collection '{COLLECTION_NAME}'...")
        qdrant_client.create_collection(
            collection_name=COLLECTION_NAME,
            vectors_config=VectorParams(size=VECTOR_SIZE, distance=Distance.COSINE),
        )
    
    yield
    
    logger.info("Shutting down...")


app = FastAPI(
    title="OpenRTB Semantic Enricher",
    description="LLM enrichment and embedding service for ad matching",
    version="1.0.0",
    lifespan=lifespan,
)


# ============================================
# Request/Response Models
# ============================================

class Ad(BaseModel):
    id: str
    title: str
    description: str
    category: str
    advertiser_id: str
    bid_price: float
    quality_score: float = 0.5
    tags: list[str] = []


class EnrichRequest(BaseModel):
    ads: list[Ad]


class EnrichResponse(BaseModel):
    processed: int
    indexed: int


class SearchRequest(BaseModel):
    query: str
    top_k: int = 10


class SearchResult(BaseModel):
    id: str
    title: str
    description: str
    category: str
    score: float
    bid_price: float


class SearchResponse(BaseModel):
    results: list[SearchResult]
    latency_ms: float


# ============================================
# API Endpoints
# ============================================

@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "healthy"}


class EmbedRequest(BaseModel):
    text: str


class EmbedResponse(BaseModel):
    embedding: list[float]


@app.post("/embed", response_model=EmbedResponse)
async def embed_text(request: EmbedRequest):
    """
    Generate embedding for query text.
    Called by Go gateway for dense search.
    """
    if not embedding_model:
        raise HTTPException(status_code=503, detail="Embedding model not loaded")
    
    embedding = embedding_model.encode(request.text).tolist()
    return EmbedResponse(embedding=embedding)


@app.post("/enrich", response_model=EnrichResponse)
async def enrich_ads(request: EnrichRequest):
    """
    Enrich ads with embeddings and index in Qdrant.
    
    In production, this would also call the LLM for semantic enrichment.
    """
    if not embedding_model or not qdrant_client:
        raise HTTPException(status_code=503, detail="Service not ready")
    
    points = []
    for ad in request.ads:
        # Create text for embedding
        text = f"{ad.title} {ad.description} {' '.join(ad.tags)}"
        
        # Generate embedding
        embedding = embedding_model.encode(text).tolist()
        
        # Create Qdrant point
        point = PointStruct(
            id=ad.id,
            vector=embedding,
            payload={
                "id": ad.id,
                "title": ad.title,
                "description": ad.description,
                "category": ad.category,
                "advertiser_id": ad.advertiser_id,
                "bid_price": ad.bid_price,
                "quality_score": ad.quality_score,
                "tags": ad.tags,
            },
        )
        points.append(point)
    
    # Batch upsert to Qdrant
    if points:
        qdrant_client.upsert(
            collection_name=COLLECTION_NAME,
            points=points,
        )
    
    return EnrichResponse(processed=len(request.ads), indexed=len(points))


@app.post("/search", response_model=SearchResponse)
async def search_ads(request: SearchRequest):
    """
    Semantic search for ads using dense retrieval.
    """
    import time
    start = time.time()
    
    if not embedding_model or not qdrant_client:
        raise HTTPException(status_code=503, detail="Service not ready")
    
    # Embed query
    query_embedding = embedding_model.encode(request.query).tolist()
    
    # Search Qdrant
    results = qdrant_client.search(
        collection_name=COLLECTION_NAME,
        query_vector=query_embedding,
        limit=request.top_k,
    )
    
    # Convert to response
    search_results = []
    for hit in results:
        search_results.append(SearchResult(
            id=hit.payload.get("id", ""),
            title=hit.payload.get("title", ""),
            description=hit.payload.get("description", ""),
            category=hit.payload.get("category", ""),
            score=hit.score,
            bid_price=hit.payload.get("bid_price", 0.0),
        ))
    
    latency = (time.time() - start) * 1000
    return SearchResponse(results=search_results, latency_ms=round(latency, 2))


@app.post("/index-from-file")
async def index_from_file(file_path: str = "/data/mock_ads.json"):
    """
    Index ads from a JSON file.
    """
    path = Path(file_path)
    if not path.exists():
        raise HTTPException(status_code=404, detail=f"File not found: {file_path}")
    
    with open(path) as f:
        ads_data = json.load(f)
    
    batch_size = 100
    total_indexed = 0
    
    for i in range(0, len(ads_data), batch_size):
        batch = ads_data[i:i + batch_size]
        ads = [Ad(**ad) for ad in batch]
        result = await enrich_ads(EnrichRequest(ads=ads))
        total_indexed += result.indexed
        logger.info(f"Indexed batch {i // batch_size + 1}, total: {total_indexed}")
    
    return {"total_indexed": total_indexed, "source": file_path}


# ============================================
# Main
# ============================================

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)
