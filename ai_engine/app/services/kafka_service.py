import json
import asyncio
from aiokafka import AIOKafkaConsumer, AIOKafkaProducer
from app.core.config import settings
from app.services.llm_service import LLMService

class KafkaConsumerService:
    
    def __init__(self,llm_service: LLMService):
        self.llm_service = llm_service
        self.debounce_tasks = {}
        self.last_diagram_state = {}
        self.consumer = AIOKafkaConsumer(
            'ai.requests',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="ai_engine_group",
        )
        self.eval_consumer = AIOKafkaConsumer(
            'ai.evaluations',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="evaluation_group",
        )
        self.diagram_consumer = AIOKafkaConsumer(
            'diagram.events',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="diagram_analysis_group",
        )
        self.producer = AIOKafkaProducer(
            bootstrap_servers = settings.KAFKA_BROKERS
        )

    async def start(self):
        await self.consumer.start()
        await self.eval_consumer.start()
        await self.diagram_consumer.start()
        await self.producer.start()
        
        try:
            await asyncio.gather(
                self.consume_chat_requests(),
                self.consume_evaluation_requests(),
                self.consume_diagram_events()
            )
        finally:
            await self.consumer.stop()
            await self.eval_consumer.stop()
            await self.diagram_consumer.stop()
            await self.producer.stop()

    async def consume_chat_requests(self):
        async for msg in self.consumer:
            await self.process_message(msg)

    async def consume_evaluation_requests(self):
        async for msg in self.eval_consumer:
            await self.process_evaluation(msg)

    async def process_message(self, msg):
        try:
            data = json.loads(msg.value.decode('utf-8'))
            
            if "interview_id" in data and "question_id" in data:
                session_id = data["interview_id"]
                question_id = data["question_id"]
                print(f"Processing INTERVIEW_STARTED event for session: {session_id}")
                
                response = await self.llm_service.generate_initial_greeting(question_id, session_id)
            elif "status" in data and data["status"] == "SUBMITTED":
                session_id = data["interview_id"]
                print(f"Processing INTERVIEW_SUBMITTED event for session: {session_id}")
                
                response = await self.llm_service.submit_interview(session_id)
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

    async def process_evaluation(self, msg):
        try:
            data = json.loads(msg.value.decode('utf-8'))
            session_id = data.get("session_id")
            if not session_id:
                print("Invalid evaluation request: session_id missing")
                return

            print(f"Processing evaluation request from Kafka for session: {session_id}")
            
            max_retries = 3
            backoff = 2
            
            for attempt in range(max_retries):
                try:
                    await self.llm_service.eval_service.evaluate_session(session_id)
                    print(f"Successfully processed evaluation for session: {session_id}")
                    return
                except Exception as eval_err:
                    print(f"Evaluation attempt {attempt + 1} failed for session {session_id}: {eval_err}")
                    if attempt < max_retries - 1:
                        await asyncio.sleep(backoff)
                        backoff *= 2
                    else:
                        raise eval_err
        except Exception as e:
            print(f"Error processing evaluation message: {e}")

    async def consume_diagram_events(self):
        async for msg in self.diagram_consumer:
            try:
                data = json.loads(msg.value.decode('utf-8'))
                session_id = data.get("session_id")
                event_type = data.get("event_type")
                if not session_id or not event_type:
                    continue

                if event_type in ["node_added", "node_updated", "node_deleted", "edge_added", "edge_updated", "edge_deleted"]:
                    if session_id in self.debounce_tasks:
                        self.debounce_tasks[session_id].cancel()
                    self.debounce_tasks[session_id] = asyncio.create_task(
                        self.trigger_diagram_critique_after_delay(session_id, 8)
                    )
            except asyncio.CancelledError:
                raise
            except Exception as e:
                print(f"Error consuming diagram event: {e}")

    async def trigger_diagram_critique_after_delay(self, session_id: str, delay: int):
        try:
            await asyncio.sleep(delay)
            diagram_ctx = self.llm_service.eval_service.get_diagram_context(session_id)
            if not diagram_ctx:
                return

            parsed_diagram = self.llm_service.diagram_service.parse_diagram_to_text(diagram_ctx)
            if self.last_diagram_state.get(session_id) == parsed_diagram:
                return

            self.last_diagram_state[session_id] = parsed_diagram
            response = await self.llm_service.process_diagram_event(session_id)
            if response:
                result_event = {
                    "session_id": session_id,
                    "response": response,
                }
                await self.producer.send_and_wait(
                    'ai.responses',
                    json.dumps(result_event).encode('utf-8')
                )
        except asyncio.CancelledError:
            pass
        except Exception as e:
            print(f"Error in diagram critique task: {e}")
        finally:
            self.debounce_tasks.pop(session_id, None)

