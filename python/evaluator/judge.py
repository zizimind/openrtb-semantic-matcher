import json
import requests
import argparse
from typing import List, Dict

# Configuration
GATEWAY_URL = "http://localhost:8080/v1/match"
# LLM Endpoint (Ollama)
OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL_NAME = "llama3"

def get_match_results(query: str) -> List[Dict]:
    """Call the Gateway to get top results for a query."""
    payload = {
        "id": "eval-judge",
        "site": {"page": query},
        "device": {"ua": "python-judge", "ip": "127.0.0.1"}
    }
    try:
        resp = requests.post(GATEWAY_URL, json=payload, timeout=5)
        resp.raise_for_status()
        data = resp.json()
        # Return candidates (limit 3)
        return data.get("candidates", [])[:3]
    except Exception as e:
        print(f"Error calling gateway: {e}")
        return []

def grade_relevance(query: str, ad: Dict) -> int:
    """Ask LLM to grade relevance of Ad to Query on 0-3 scale."""
    
    prompt = f"""
    You are a relevance judge. Rate the relevance of the Ad to the User Query.
    
    Query: "{query}"
    
    Ad Title: "{ad.get('title')}"
    Ad Description: "{ad.get('description')}"
    Ad Category: "{ad.get('category')}"
    
    Scale:
    3: Perfect match (Category and intent match accurately)
    2: Relevant (Category matches, intent related)
    1: Weakly relevant (Category valid but intent mismatch)
    0: Irrelevant
    
    Output ONLY a single integer (0, 1, 2, or 3). Do not explain.
    """
    
    payload = {
        "model": MODEL_NAME,
        "prompt": prompt,
        "stream": False
    }
    
    try:
        resp = requests.post(OLLAMA_URL, json=payload, timeout=30)
        resp.raise_for_status()
        result = resp.json().get("response", "").strip()
        # Attempt to parse integer
        return int(result[0]) if result else 0
    except Exception as e:
        print(f"LLM Error: {e}")
        return 0

def main():
    parser = argparse.ArgumentParser(description="LLM-as-a-Judge Evaluator")
    parser.add_argument("--queries", type=int, default=10, help="Number of queries to evaluate")
    args = parser.parse_args()
    
    test_queries = [
        "luxury beach vacation",
        "best running shoes for men",
        "cloud computing software for startups",
        "organic dog food",
        "buy bitcoin investment",
        "italian restaurant downtown",
        "home renovation contractors",
        "science fiction movies",
        "learn python programming course",
        "cheap flights to london"
    ]
    
    # Repeat queries if more requested
    test_set = (test_queries * (args.queries // len(test_queries) + 1))[:args.queries]
    
    results = []
    
    print(f"Running evaluation on {len(test_set)} queries...")
    print("-" * 60)
    print(f"{'Query':<30} | {'Ad Title':<30} | {'Score':<5} | {'Latency':<5}")
    print("-" * 60)
    
    total_score = 0
    count = 0
    
    for query in test_set:
        candidates = get_match_results(query)
        if not candidates:
            print(f"{query:<30} | {'(No Candidates)':<30} | {'0':<5} | {'-'}")
            continue
            
        # Grade the top result ONLY for NDCG@1 approximation
        top_ad = candidates[0]
        score = grade_relevance(query, top_ad)
        
        results.append({
            "query": query,
            "top_ad": top_ad['title'],
            "score": score
        })
        
        total_score += score
        count += 1
        
        print(f"{query:<30} | {top_ad['title'][:30]:<30} | {score:<5} | -")
        
    print("-" * 60)
    avg_score = total_score / count if count > 0 else 0
    # Normalize 0-3 scale to 0-1
    normalized_relevance = avg_score / 3.0
    print(f"Average Relevance Score: {avg_score:.2f} / 3.0")
    print(f"Normalized Quality (NDCG-proxy): {normalized_relevance:.2%}")

if __name__ == "__main__":
    main()
