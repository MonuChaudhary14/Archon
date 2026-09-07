from contextlib import asynccontextmanager
import asyncio
from fastapi import FastAPI
from app.core.config import settings
from app.services.llm_service import LLMService
from app.services.kafka_service import KafkaConsumerService
from app.workflows.worker import TemporalEvaluationWorker

@asynccontextmanager
async def lifespan(app: FastAPI):
    llm_service = LLMService()
    temporal_worker = TemporalEvaluationWorker()
    kafka_service = KafkaConsumerService(llm_service, temporal_worker=temporal_worker)
    llm_service.set_kafka_producer(kafka_service.producer)

    kafka_task = asyncio.create_task(kafka_service.start())
    temporal_task = asyncio.create_task(temporal_worker.start())

    yield

    kafka_task.cancel()
    temporal_task.cancel()

app = FastAPI(
    title=settings.PROJECT_NAME,
    version=settings.VERSION,
    lifespan=lifespan
)

@app.get("/health")
async def health_check():
    return {"status": "healthy"}