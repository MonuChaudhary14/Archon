import os
import sys

sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from app.services.knowledge_service import KnowledgeService

if __name__ == "__main__":
    print("Initializing Knowledge base seeding...")
    if not os.getenv("QDRANT_URL"):
        os.environ["QDRANT_URL"] = "http://localhost:6333"
    
    service = KnowledgeService()
    service.seed_knowledge_base()
    print("Ingestion script completed!")
