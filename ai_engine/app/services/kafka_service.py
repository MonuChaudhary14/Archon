import json
import asyncio
from aiokafka import AIOKafkaConsumer, AIOKafkaProducer
from app.core.config import settings
from app.services.llm_service import LLMService

class KafkaConsumerService:
    
    def __init__(self,llm_service: LLMService):
        self.llm_service = llm_service
        self.consumer = AIOKafkaConsumer(
            'ai.requests',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="ai_engine_group",
            
        )
        self.producer = AIOKafkaProducer(
            bootstrap_servers = settings.KAFKA_BROKERS
        )

    async def start(self):
        await self.consumer.start()
        await self.producer.start()
        
        try:
            async for msg in self.consumer:
                await self.process_message(msg)

        finally:
            await self.consumer.stop()
            await self.producer.stop()

    async def process_message(self, msg):
        try:
            data = json.loads(msg.value.decode('utf-8'))
            
            if "interview_id" in data and "question_id" in data:
                session_id = data["interview_id"]
                question_id = data["question_id"]
                print(f"Processing INTERVIEW_STARTED event for session: {session_id}")
                
                response = await self.llm_service.generate_initial_greeting(question_id, session_id)
            else:
                prompt = data.get("prompt")
                session_id = data.get("session_id", "default")
                print(f"Processing request for session: {session_id}")
                
                response = await self.llm_service.generate_response(prompt, session_id)

            result_event = {
                "session_id": session_id,
                "response": response,
            }

            await self.producer.send_and_wait(
                'ai.responses',
                json.dumps(result_event).encode('utf-8')
            )

        except Exception as e:
            print(f"Error processing message: {e}")

