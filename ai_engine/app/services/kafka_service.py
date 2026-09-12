import json
import asyncio
import time
from aiokafka import AIOKafkaConsumer, AIOKafkaProducer
from app.core.config import settings
from app.core.telemetry import get_tracer, extract_trace_context_from_headers, inject_trace_context_to_headers
from app.services.llm_service import LLMService

class KafkaConsumerService:
    
    def __init__(self, llm_service: LLMService, temporal_worker = None):
        self.llm_service = llm_service
        self.temporal_worker = temporal_worker
        self.debounce_tasks = {}
        self.last_diagram_state = {}
        self.consumer = AIOKafkaConsumer(
            'ai.requests',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="ai_engine_group",
        )
        self.retry_10s_consumer = AIOKafkaConsumer(
            'ai.requests.retry.10s',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="ai_engine_retry_10s_group",
        )
        self.retry_60s_consumer = AIOKafkaConsumer(
            'ai.requests.retry.60s',
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="ai_engine_retry_60s_group",
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
        await self.retry_10s_consumer.start()
        await self.retry_60s_consumer.start()
        await self.eval_consumer.start()
        await self.diagram_consumer.start()
        await self.producer.start()
        
        try:
            await asyncio.gather(
                self.consume_chat_requests(),
                self.consume_retry_requests(self.retry_10s_consumer, "10s"),
                self.consume_retry_requests(self.retry_60s_consumer, "60s"),
                self.consume_evaluation_requests(),
                self.consume_diagram_events()
            )
        finally:
            await self.consumer.stop()
            await self.retry_10s_consumer.stop()
            await self.retry_60s_consumer.stop()
            await self.eval_consumer.stop()
            await self.diagram_consumer.stop()
            await self.producer.stop()

    async def consume_chat_requests(self):
        async for msg in self.consumer:
            await self.process_message(msg)

    async def consume_retry_requests(self, consumer: AIOKafkaConsumer, tier: str):
        async for msg in consumer:
            try:
                data = json.loads(msg.value.decode('utf-8'))
                retry_after = data.get("retry_after", 0)
                now = time.time()
                if now < retry_after:
                    delay = retry_after - now
                    await asyncio.sleep(delay)
            except Exception:
                pass
            await self.process_message(msg)

    async def consume_evaluation_requests(self):
        async for msg in self.eval_consumer:
            await self.process_evaluation(msg)

    async def handle_failure_and_retry(self, msg, error: Exception, data: dict):
        retry_count = data.get("retry_count", 0)
        session_id = data.get("session_id") or data.get("interview_id", "default")
        out_headers = inject_trace_context_to_headers()
        
        if retry_count == 0:
            data["retry_count"] = 1
            data["retry_after"] = time.time() + 10
            data["last_error"] = str(error)
            print(f"Routing session {session_id} to ai.requests.retry.10s (attempt 1)")
            await self.producer.send_and_wait(
                'ai.requests.retry.10s',
                json.dumps(data).encode('utf-8'),
                headers=out_headers
            )
        elif retry_count == 1:
            data["retry_count"] = 2
            data["retry_after"] = time.time() + 60
            data["last_error"] = str(error)
            print(f"Routing session {session_id} to ai.requests.retry.60s (attempt 2)")
            await self.producer.send_and_wait(
                'ai.requests.retry.60s',
                json.dumps(data).encode('utf-8'),
                headers=out_headers
            )
        else:
            print(f"Routing session {session_id} to ai.requests.dlq (terminal failure)")
            dlq_payload = {
                "original_topic": msg.topic,
                "failed_at": time.time(),
                "error": str(error),
                "retry_count": retry_count,
                "data": data,
            }
            await self.producer.send_and_wait(
                'ai.requests.dlq',
                json.dumps(dlq_payload).encode('utf-8'),
                headers=out_headers
            )
            fallback_event = {
                "session_id": session_id,
                "response": "I encountered a technical issue processing your request. Please submit your message again.",
            }
            await self.producer.send_and_wait(
                'ai.responses',
                json.dumps(fallback_event).encode('utf-8'),
                headers=out_headers
            )

    async def process_message(self, msg):
        tracer = get_tracer()
        parent_ctx = extract_trace_context_from_headers(msg.headers)
        data = None
        
        with tracer.start_as_current_span("ai_engine.process_chat_request", context=parent_ctx) as span:
            try:
                data = json.loads(msg.value.decode('utf-8'))
                if isinstance(data, dict) and "payload" in data:
                    payload_val = data["payload"]
                    if isinstance(payload_val, str):
                        try:
                            data = json.loads(payload_val)
                        except Exception:
                            pass
                    elif isinstance(payload_val, dict):
                        data = payload_val

                out_headers = inject_trace_context_to_headers()
                
                if "interview_id" in data and "question_id" in data:
                    session_id = data["interview_id"]
                    question_id = data["question_id"]
                    span.set_attribute("session_id", session_id)
                    span.set_attribute("event_type", "INTERVIEW_STARTED")
                    print(f"Processing INTERVIEW_STARTED event for session: {session_id}")
                    
                    response = await self.llm_service.generate_initial_greeting(question_id, session_id)
                    result_event = {
                        "session_id": session_id,
                        "response": response,
                        "is_final": True,
                    }
                    await self.producer.send_and_wait(
                        'ai.responses',
                        json.dumps(result_event).encode('utf-8'),
                        headers=out_headers
                    )
                elif "status" in data and data["status"] == "SUBMITTED":
                    session_id = data["interview_id"]
                    span.set_attribute("session_id", session_id)
                    span.set_attribute("event_type", "INTERVIEW_SUBMITTED")
                    print(f"Processing INTERVIEW_SUBMITTED event for session: {session_id}")
                    
                    response = await self.llm_service.submit_interview(session_id)
                    result_event = {
                        "session_id": session_id,
                        "response": response,
                        "is_final": True,
                    }
                    await self.producer.send_and_wait(
                        'ai.responses',
                        json.dumps(result_event).encode('utf-8'),
                        headers=out_headers
                    )
                else:
                    prompt = data.get("prompt")
                    session_id = data.get("session_id", "default")
                    span.set_attribute("session_id", session_id)
                    span.set_attribute("event_type", "CHAT_PROMPT")
                    print(f"Processing streaming request for session: {session_id}")
                    
                    async for event in self.llm_service.generate_response_stream(prompt, session_id):
                        if not event.get("is_final"):
                            chunk_event = {
                                "session_id": session_id,
                                "delta": event["delta"],
                                "is_final": False,
                            }
                            await self.producer.send(
                                'ai.responses',
                                json.dumps(chunk_event).encode('utf-8'),
                                headers=out_headers
                            )
                        else:
                            final_event = {
                                "session_id": session_id,
                                "response": event["response"],
                                "is_final": True,
                                "state": event.get("state", ""),
                            }
                            await self.producer.send_and_wait(
                                'ai.responses',
                                json.dumps(final_event).encode('utf-8'),
                                headers=out_headers
                            )

            except Exception as e:
                span.record_exception(e)
                print(f"Error processing message on {msg.topic}: {e}")
                if data is not None:
                    await self.handle_failure_and_retry(msg, e, data)

    async def process_evaluation(self, msg):
        tracer = get_tracer()
        parent_ctx = extract_trace_context_from_headers(msg.headers)
        
        with tracer.start_as_current_span("ai_engine.process_evaluation", context=parent_ctx) as span:
            try:
                data = json.loads(msg.value.decode('utf-8'))
                if isinstance(data, dict) and "payload" in data:
                    payload_val = data["payload"]
                    if isinstance(payload_val, str):
                        try:
                            data = json.loads(payload_val)
                        except Exception:
                            pass
                    elif isinstance(payload_val, dict):
                        data = payload_val

                session_id = data.get("session_id")
                if not session_id:
                    print("Invalid evaluation request: session_id missing")
                    return

                span.set_attribute("session_id", session_id)
                print(f"Processing evaluation request from Kafka for session: {session_id}")

                if self.temporal_worker:
                    try:
                        await self.temporal_worker.execute_evaluation_workflow(session_id)
                        print(f"Successfully triggered Temporal workflow for session: {session_id}")
                        return
                    except Exception as temp_err:
                        span.record_exception(temp_err)
                        print(f"Failed to start Temporal workflow: {temp_err}. Falling back to direct evaluation.")

                max_retries = 3
                backoff = 2
                
                for attempt in range(max_retries):
                    try:
                        await self.llm_service.eval_service.evaluate_session(session_id)
                        print(f"Successfully processed evaluation for session: {session_id}")
                        return
                    except Exception as eval_err:
                        span.record_exception(eval_err)
                        print(f"Evaluation attempt {attempt + 1} failed for session {session_id}: {eval_err}")
                        if attempt < max_retries - 1:
                            await asyncio.sleep(backoff)
                            backoff *= 2
                        else:
                            raise eval_err
            except Exception as e:
                span.record_exception(e)
                print(f"Error processing evaluation message: {e}")

    async def consume_diagram_events(self):
        async for msg in self.diagram_consumer:
            try:
                data = json.loads(msg.value.decode('utf-8'))
                if isinstance(data, dict) and "payload" in data:
                    payload_val = data["payload"]
                    if isinstance(payload_val, str):
                        try:
                            data = json.loads(payload_val)
                        except Exception:
                            pass
                    elif isinstance(payload_val, dict):
                        data = payload_val

                session_id = data.get("session_id")
                event_type = data.get("event_type")
                if not session_id or not event_type:
                    continue

                if event_type in ["node_added", "node_updated", "node_deleted", "edge_added", "edge_updated", "edge_deleted"]:
                    if session_id in self.debounce_tasks:
                        self.debounce_tasks[session_id].cancel()
                    self.debounce_tasks[session_id] = asyncio.create_task(
                        self.trigger_diagram_critique_after_delay(session_id, 8, msg.headers)
                    )
            except asyncio.CancelledError:
                raise
            except Exception as e:
                print(f"Error consuming diagram event: {e}")

    async def trigger_diagram_critique_after_delay(self, session_id: str, delay: int, headers=None):
        tracer = get_tracer()
        parent_ctx = extract_trace_context_from_headers(headers)
        
        with tracer.start_as_current_span("ai_engine.diagram_critique_debounced", context=parent_ctx) as span:
            span.set_attribute("session_id", session_id)
            try:
                await asyncio.sleep(delay)
                diagram_ctx = self.llm_service.repo.get_diagram_context(session_id)
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
                    out_headers = inject_trace_context_to_headers()
                    await self.producer.send_and_wait(
                        'ai.responses',
                        json.dumps(result_event).encode('utf-8'),
                        headers=out_headers
                    )
            except asyncio.CancelledError:
                pass
            except Exception as e:
                span.record_exception(e)
                print(f"Error in diagram critique task: {e}")
            finally:
                self.debounce_tasks.pop(session_id, None)
