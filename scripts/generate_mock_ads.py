#!/usr/bin/env python3
"""
Mock Ad Generator for OpenRTB Semantic Matcher

Generates 10,000 mock ads with realistic titles, descriptions,
categories, and bid prices for testing.
"""

import json
import random
import uuid
from pathlib import Path
from datetime import datetime

# Categories and their associated keywords
CATEGORIES = {
    "technology": ["software", "app", "cloud", "AI", "machine learning", "SaaS", "startup", "digital"],
    "fashion": ["clothing", "style", "designer", "luxury", "boutique", "trendy", "modern", "elegant"],
    "automotive": ["car", "vehicle", "auto", "motor", "drive", "electric", "hybrid", "SUV"],
    "travel": ["vacation", "trip", "hotel", "flight", "adventure", "destination", "resort", "booking"],
    "finance": ["banking", "investment", "loan", "credit", "insurance", "wealth", "savings", "trading"],
    "food": ["restaurant", "delivery", "organic", "gourmet", "cuisine", "healthy", "fresh", "local"],
    "health": ["wellness", "fitness", "medical", "healthcare", "vitamin", "supplement", "therapy", "care"],
    "education": ["learning", "course", "training", "school", "university", "online", "skill", "certification"],
    "entertainment": ["streaming", "gaming", "movie", "music", "concert", "show", "festival", "event"],
    "home": ["furniture", "decor", "interior", "appliance", "renovation", "smart home", "garden", "living"],
}

# Ad title templates
TITLE_TEMPLATES = [
    "{adj} {category} Solutions for Modern {audience}",
    "Discover {adj} {keyword} - {benefit}",
    "{brand} - {adj} {category} {product_type}",
    "Premium {keyword} Services | {benefit}",
    "{adj} {keyword} Platform - {cta}",
    "Best {keyword} for {audience} | {brand}",
    "{audience}'s Choice: {adj} {keyword}",
    "Transform Your {area} with {adj} {keyword}",
    "{brand}: Leading {keyword} Provider",
    "Innovative {keyword} - {benefit}",
]

ADJECTIVES = ["Premium", "Modern", "Professional", "Innovative", "Smart", "Advanced", "Trusted", "Leading", "Affordable", "Expert"]
BENEFITS = ["Save Time", "Boost Performance", "Increase ROI", "Start Free", "Get Results", "See the Difference", "Join Now", "Transform Today"]
AUDIENCES = ["Businesses", "Professionals", "Entrepreneurs", "Teams", "Creators", "Families", "Students", "Developers"]
CTAS = ["Get Started", "Learn More", "Try Free", "Join Today", "Subscribe Now", "Explore", "Discover", "Sign Up"]
BRAND_PREFIXES = ["Neo", "Zen", "Peak", "Nova", "Apex", "Prime", "Core", "Flux", "Vibe", "Pulse"]
BRAND_SUFFIXES = ["Tech", "Hub", "Pro", "Labs", "Cloud", "AI", "Works", "Space", "Flow", "Link"]
PRODUCT_TYPES = ["Platform", "Suite", "Tools", "Solutions", "Services", "System", "App", "Software"]
AREAS = ["Business", "Workflow", "Strategy", "Operations", "Marketing", "Sales", "Growth", "Brand"]

def generate_brand():
    return f"{random.choice(BRAND_PREFIXES)}{random.choice(BRAND_SUFFIXES)}"

def generate_title(category, keyword):
    template = random.choice(TITLE_TEMPLATES)
    return template.format(
        adj=random.choice(ADJECTIVES),
        category=category.title(),
        keyword=keyword.title(),
        benefit=random.choice(BENEFITS),
        brand=generate_brand(),
        audience=random.choice(AUDIENCES),
        cta=random.choice(CTAS),
        product_type=random.choice(PRODUCT_TYPES),
        area=random.choice(AREAS),
    )

def generate_description(category, keywords):
    templates = [
        f"Discover our {random.choice(ADJECTIVES).lower()} {category} solutions. {random.choice(keywords).title()} made easy for {random.choice(AUDIENCES).lower()}. {random.choice(BENEFITS)}!",
        f"Looking for {category}? We offer {random.choice(ADJECTIVES).lower()} {random.choice(keywords)} services. Trusted by thousands of {random.choice(AUDIENCES).lower()}.",
        f"Transform your {random.choice(AREAS).lower()} with our {random.choice(ADJECTIVES).lower()} {random.choice(keywords)} platform. {random.choice(CTAS)}!",
        f"The leading {category} solution for {random.choice(AUDIENCES).lower()}. Experience {random.choice(ADJECTIVES).lower()} {random.choice(keywords)} today.",
    ]
    return random.choice(templates)

def generate_tags(category, keywords):
    base_tags = [category] + random.sample(keywords, min(3, len(keywords)))
    extra_tags = random.sample(ADJECTIVES, 2) + random.sample(BENEFITS, 1)
    return list(set([t.lower().replace(" ", "-") for t in base_tags + extra_tags]))

def generate_mock_ad(index):
    category = random.choice(list(CATEGORIES.keys()))
    keywords = CATEGORIES[category]
    keyword = random.choice(keywords)
    
    # Generate bid price with realistic distribution
    # Most ads have low bids, few have high bids
    base_bid = random.uniform(0.01, 2.0)
    if random.random() < 0.1:  # 10% premium ads
        base_bid *= random.uniform(3, 10)
    
    # Quality score (simulated CTR-based)
    quality = random.uniform(0.3, 1.0)
    if random.random() < 0.2:  # 20% high-quality ads
        quality = random.uniform(0.8, 1.0)
    
    ad_id = str(uuid.uuid4())
    advertiser_id = f"adv_{random.randint(1000, 9999)}"
    
    return {
        "id": ad_id,
        "title": generate_title(category, keyword),
        "description": generate_description(category, keywords),
        "category": category,
        "advertiser_id": advertiser_id,
        "bid_price": round(base_bid, 4),
        "quality_score": round(quality, 3),
        "tags": generate_tags(category, keywords),
        "image_url": f"https://picsum.photos/seed/{ad_id[:8]}/300/250",
        "click_url": f"https://example.com/click/{ad_id}",
        "creative_w": random.choice([300, 728, 320]),
        "creative_h": random.choice([250, 90, 50]),
        "active": True,
        "created_at": int(datetime.now().timestamp()),
        "updated_at": int(datetime.now().timestamp()),
    }

def main():
    print("🚀 Generating 10,000 mock ads...")
    
    # Create output directory
    output_dir = Path(__file__).parent.parent / "data"
    output_dir.mkdir(exist_ok=True)
    
    # Generate ads
    ads = [generate_mock_ad(i) for i in range(10000)]
    
    # Write to JSON
    output_file = output_dir / "mock_ads.json"
    with open(output_file, "w") as f:
        json.dump(ads, f, indent=2)
    
    print(f"✅ Generated {len(ads)} mock ads")
    print(f"📁 Output: {output_file}")
    
    # Print category distribution
    category_counts = {}
    for ad in ads:
        cat = ad["category"]
        category_counts[cat] = category_counts.get(cat, 0) + 1
    
    print("\n📊 Category Distribution:")
    for cat, count in sorted(category_counts.items()):
        print(f"   {cat}: {count}")
    
    # Print bid price stats
    bid_prices = [ad["bid_price"] for ad in ads]
    print(f"\n💰 Bid Price Stats:")
    print(f"   Min: ${min(bid_prices):.4f}")
    print(f"   Max: ${max(bid_prices):.4f}")
    print(f"   Avg: ${sum(bid_prices)/len(bid_prices):.4f}")

if __name__ == "__main__":
    main()
