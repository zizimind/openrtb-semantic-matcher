"""
DSPy Pipeline for Ad Enrichment

Uses Ollama/Llama-3 to generate:
1. Semantic summaries
2. Expert tags for sparse retrieval
3. Quality assessments
"""

import dspy
from typing import Optional
import logging

logger = logging.getLogger(__name__)


# ============================================
# DSPy Signatures (Input/Output Contracts)
# ============================================

class AdEnricher(dspy.Signature):
    """Enrich an ad with semantic summary and expert tags."""
    
    title: str = dspy.InputField(desc="Original ad title")
    description: str = dspy.InputField(desc="Original ad description")
    category: str = dspy.InputField(desc="Ad category")
    
    semantic_summary: str = dspy.OutputField(
        desc="A concise, semantically rich summary that captures the ad's intent and value proposition"
    )
    expert_tags: str = dspy.OutputField(
        desc="Comma-separated high-intent keywords for search matching (5-10 tags)"
    )


class AdQualityAssessor(dspy.Signature):
    """Assess the quality of an ad for ranking."""
    
    title: str = dspy.InputField(desc="Ad title")
    description: str = dspy.InputField(desc="Ad description")
    
    quality_score: float = dspy.OutputField(
        desc="Quality score from 0.0 to 1.0 based on clarity, relevance, and professionalism"
    )
    reasoning: str = dspy.OutputField(
        desc="Brief explanation of the quality assessment"
    )


class QueryExpander(dspy.Signature):
    """Expand a search query for better retrieval."""
    
    query: str = dspy.InputField(desc="Original search query")
    context: str = dspy.InputField(desc="Context about the user or page")
    
    expanded_query: str = dspy.OutputField(
        desc="Expanded query with synonyms and related terms"
    )
    intent: str = dspy.OutputField(
        desc="Inferred user intent (e.g., 'purchase', 'research', 'compare')"
    )


# ============================================
# DSPy Modules (Pipelines)
# ============================================

class AdEnrichmentPipeline(dspy.Module):
    """Complete pipeline for enriching ads with LLM."""
    
    def __init__(self):
        super().__init__()
        self.enricher = dspy.ChainOfThought(AdEnricher)
        self.assessor = dspy.Predict(AdQualityAssessor)
    
    def forward(self, title: str, description: str, category: str) -> dict:
        # Step 1: Generate semantic summary and tags
        enrichment = self.enricher(
            title=title,
            description=description,
            category=category,
        )
        
        # Step 2: Assess quality
        quality = self.assessor(
            title=title,
            description=description,
        )
        
        return {
            "semantic_summary": enrichment.semantic_summary,
            "expert_tags": [t.strip() for t in enrichment.expert_tags.split(",")],
            "quality_score": min(max(float(quality.quality_score), 0.0), 1.0),
            "quality_reasoning": quality.reasoning,
        }


class QueryExpansionPipeline(dspy.Module):
    """Pipeline for expanding search queries."""
    
    def __init__(self):
        super().__init__()
        self.expander = dspy.ChainOfThought(QueryExpander)
    
    def forward(self, query: str, context: str = "") -> dict:
        result = self.expander(query=query, context=context)
        return {
            "original_query": query,
            "expanded_query": result.expanded_query,
            "intent": result.intent,
        }


# ============================================
# Ollama Configuration
# ============================================

def configure_ollama(host: str = "http://localhost:11434", model: str = "llama3:8b"):
    """Configure DSPy to use Ollama as the LLM backend."""
    try:
        lm = dspy.OllamaLocal(
            model=model,
            base_url=host,
            timeout_s=60,
        )
        dspy.configure(lm=lm)
        logger.info(f"Configured DSPy with Ollama: {model} at {host}")
        return True
    except Exception as e:
        logger.error(f"Failed to configure Ollama: {e}")
        return False


# ============================================
# Batch Processing
# ============================================

def enrich_ads_batch(ads: list[dict], batch_size: int = 10) -> list[dict]:
    """
    Enrich a batch of ads using the DSPy pipeline.
    
    Args:
        ads: List of ad dictionaries with 'title', 'description', 'category'
        batch_size: Number of ads to process before yielding
        
    Returns:
        List of enriched ad dictionaries
    """
    pipeline = AdEnrichmentPipeline()
    enriched = []
    
    for i, ad in enumerate(ads):
        try:
            result = pipeline(
                title=ad.get("title", ""),
                description=ad.get("description", ""),
                category=ad.get("category", ""),
            )
            
            enriched_ad = {
                **ad,
                "semantic_summary": result["semantic_summary"],
                "expert_tags": result["expert_tags"],
                "llm_quality_score": result["quality_score"],
            }
            enriched.append(enriched_ad)
            
            if (i + 1) % batch_size == 0:
                logger.info(f"Enriched {i + 1}/{len(ads)} ads")
                
        except Exception as e:
            logger.warning(f"Failed to enrich ad {ad.get('id', 'unknown')}: {e}")
            enriched.append(ad)  # Keep original if enrichment fails
    
    return enriched


# ============================================
# CLI for Testing
# ============================================

if __name__ == "__main__":
    import json
    import argparse
    
    parser = argparse.ArgumentParser(description="Enrich ads with LLM")
    parser.add_argument("--host", default="http://localhost:11434", help="Ollama host")
    parser.add_argument("--model", default="llama3:8b", help="Ollama model")
    parser.add_argument("--input", default="/data/mock_ads.json", help="Input JSON file")
    parser.add_argument("--output", default="/data/enriched_ads.json", help="Output JSON file")
    parser.add_argument("--limit", type=int, default=10, help="Number of ads to enrich")
    args = parser.parse_args()
    
    # Configure Ollama
    if not configure_ollama(args.host, args.model):
        print("Failed to connect to Ollama. Make sure it's running.")
        exit(1)
    
    # Load ads
    with open(args.input) as f:
        ads = json.load(f)[:args.limit]
    
    print(f"Enriching {len(ads)} ads...")
    
    # Enrich
    enriched = enrich_ads_batch(ads)
    
    # Save
    with open(args.output, "w") as f:
        json.dump(enriched, f, indent=2)
    
    print(f"Saved enriched ads to {args.output}")
