from contextlib import asynccontextmanager
import asyncio
from fastapi import FastAPI
from app.core.config import settings
from app.services.llm_service import LLMService
from app.services.kafka_service import KafkaConsumerService

@asynccontextmanager
async def lifespan(app: FastAPI):
    llm_service = LLMService()
    kafka_service = KafkaConsumerService(llm_service)

    task = asyncio.create_task(kafka_service.start())
    yield
    task.cancel()

app =FastAPI(
    title=settings.PROJECT_NAME,
    version=settings.VERSION,
    lifespan=lifespan
)

@app.get("/health")
async def health_check():
    return {"status" :"healthy"}