import os
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct
from app.core.config import settings

class KnowledgeService:
    def __init__(self):
        qdrant_url = settings.QDRANT_URL
        if not qdrant_url:
            qdrant_url = "http://localhost:6333"
        
        self.client = QdrantClient(url=qdrant_url)
        self.collection_name = "system_design_concepts"

    def seed_knowledge_base(self):
        try:
            collections = self.client.get_collections().collections
            exists = any(c.name == self.collection_name for c in collections)
            
            if exists:
                print(f"Qdrant collection '{self.collection_name}' already exists. Deleting it to ensure clean seeding...")
                self.client.delete_collection(self.collection_name)
                
            print(f"Creating Qdrant collection: {self.collection_name}")
            self.client.create_collection(
                collection_name=self.collection_name,
                vectors_config=VectorParams(size=384, distance=Distance.COSINE)
            )
                
            concepts = [
                {
                    "id": 1,
                    "title": "CAP Theorem",
                    "content": "CAP Theorem states that a distributed data store can simultaneously provide at most two of three guarantees: Consistency (every read receives the most recent write), Availability (every request receives a non-error response), and Partition tolerance (system continues to operate despite network splits). Partition tolerance is non-negotiable in distributed networks, so databases must choose between Consistency (CP, e.g., HBase, Spanner) and Availability (AP, e.g., Cassandra, DynamoDB)."
                },
                {
                    "id": 2,
                    "title": "Consistent Hashing",
                    "content": "Consistent Hashing is a key load balancing technique used in distributed caching and databases to minimize key remapping when servers are added or removed. It maps both servers and data keys to a circular virtual ring. Adding or removing a node only impacts keys in its immediate neighborhood on the ring, requiring only O(K/N) key movements instead of reshuffling all keys."
                },
                {
                    "id": 3,
                    "title": "CDN & Caching Hierarchy",
                    "content": "Content Delivery Network (CDN) is a globally distributed network of proxy servers that cache assets close to users to reduce latency. Cache invalidation strategies include Time-to-Live (TTL) expiration, active purge, and cache-busting. Caching strategies include Cache-Aside (application updates DB, invalidates cache), Write-Through (updates cache and DB synchronously), and Write-Back (updates cache, asynchronously flushes to DB)."
                },
                {
                    "id": 4,
                    "title": "Rate Limiting Algorithms",
                    "content": "Rate Limiting limits client request volume to prevent DDoS and brute-force attacks. Key algorithms: Token Bucket (allows spikes, simple), Leaky Bucket (smooths request flow into constant rate), Fixed Window Counter (simple, has edge traffic double-spikes), and Sliding Window Log/Counter (highly accurate but memory intensive). Distributed rate limiting is typically backed by Redis for fast atomic increments."
                },
                {
                    "id": 5,
                    "title": "Database Sharding & Partitioning",
                    "content": "Database Sharding is the horizontal partitioning of a database to scale writes. Data is routed to shards using range-based, list-based, or hash-based routing. Choosing a good shard key (like customer_id) is critical to avoid hotspot shards. Sharding challenges include complex cross-shard joins, lack of distributed ACID transactions (requires 2-Phase Commit), and re-sharding overhead."
                },
                {
                    "id": 6,
                    "title": "Message Queues & Event-Driven Architecture",
                    "content": "Message Queues (like Kafka or RabbitMQ) enable asynchronous decoupled service communication, buffer spike loads, and provide retry backoff. Kafka partitions messages by hashing a partition key (e.g. user_id) to guarantee strict message ordering within a partition while consumers in a consumer group process different partitions in parallel."
                }
            ]
            
            points = []
            for item in concepts:
                vector = self._get_embedding(item["content"])
                points.append(
                    PointStruct(
                        id=item["id"],
                        vector=vector,
                        payload={"title": item["title"], "content": item["content"]}
                    )
                )
            
            self.client.upsert(
                collection_name=self.collection_name,
                points=points
            )
            print("Successfully seeded Qdrant system design concepts collection!")
        except Exception as e:
            print(f"Error seeding Qdrant: {e}")

    def _get_embedding(self, text: str) -> list:
        token = settings.HUGGINGFACEHUB_API_TOKEN
        if not token:
            print("Warning: HUGGINGFACEHUB_API_TOKEN is not set.")
            return [0.0] * 384
            
        model_id = "sentence-transformers/all-MiniLM-L6-v2"
        url = f"https://api-inference.huggingface.co/models/{model_id}"
        
        import urllib.request
        import json
        import time
        
        req = urllib.request.Request(
            url,
            data=json.dumps({"inputs": text}).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json"
            },
            method="POST"
        )
        
        for attempt in range(5):
            try:
                with urllib.request.urlopen(req, timeout=10) as response:
                    res = json.loads(response.read().decode("utf-8"))
                    
                    if isinstance(res, dict) and "error" in res:
                        estimated_time = int(res.get("estimated_time", 20))
                        print(f"Model is loading, waiting {estimated_time} seconds...")
                        time.sleep(estimated_time)
                        continue
                        
                    if isinstance(res, list):
                        val = res
                        while isinstance(val, list) and len(val) > 0 and isinstance(val[0], list):
                            val = val[0]
                        return val
                    return res
            except Exception as e:
                print(f"HF embedding error (attempt {attempt + 1}/5): {e}")
                time.sleep(5)
                
        print("Failed to retrieve embedding after retries.")
        return [0.0] * 384

    def retrieve_relevant_concepts(self, query: str, limit: int = 2) -> list:
        try:
            query_vector = self._get_embedding(query)
            
            results = self.client.search(
                collection_name=self.collection_name,
                query_vector=query_vector,
                limit=limit
            )
            
            concepts_text = []
            for hit in results:
                title = hit.payload.get("title", "")
                content = hit.payload.get("content", "")
                concepts_text.append(f"[{title}] {content}")
            
            return concepts_text
        except Exception as e:
            print(f"Error retrieving concepts from Qdrant: {e}")
            return []
