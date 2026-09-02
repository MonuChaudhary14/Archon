from langchain_groq import ChatGroq
from app.core.config import settings
from langchain_community.utilities.sql_database import SQLDatabase
from app.core.helpers import get_message_history, get_interview_context
import json
import re
import redis
import asyncio
from app.services.evaluation_service import EvaluationService
from app.services.knowledge_service import KnowledgeService
from app.services.diagram_service import DiagramService
from app.services.interview_repository import SQLInterviewRepository
from app.core.interfaces import InterviewRepository

class LLMService:
    def __init__(self, repo: InterviewRepository | None = None):
        self.llm = ChatGroq(
            model=settings.GROQ_MODEL or "llama-3.3-70b-versatile",
            temperature=0.7,
            groq_api_key=settings.GROQ_API_KEY
        )

        self.db = None

        db_url = settings.get_database_url()
        if db_url:
            self.db = SQLDatabase.from_uri(db_url)

        self.redis_client = None
        if settings.REDIS_URL:
            self.redis_client = redis.from_url(settings.REDIS_URL)

        self.repo = repo or SQLInterviewRepository(self.db)
        self.eval_service = EvaluationService(self.llm, self.repo)
        self.knowledge_service = KnowledgeService()
        self.diagram_service = DiagramService(self.llm)
        self.producer = None

    def set_kafka_producer(self, producer):
        self.producer = producer

    async def publish_evaluation_request(self, session_id: str):
        if not self.producer:
            print(f"Warning: Kafka producer not set in LLMService. Falling back to asyncio task.")
            asyncio.create_task(self.eval_service.evaluate_session(session_id))
            return
        
        try:
            payload = {"session_id": session_id}
            await self.producer.send_and_wait(
                "ai.evaluations",
                json.dumps(payload).encode("utf-8")
            )
            print(f"Published evaluation request to ai.evaluations for session: {session_id}")
        except Exception as e:
            print(f"Failed to publish evaluation request: {e}. Falling back to asyncio task.")
            asyncio.create_task(self.eval_service.evaluate_session(session_id))

    def get_message_history(self, session_id: str):
        return get_message_history(session_id)


    def _parse_json_response(self, content: str):
        content = content.strip()
        if content.startswith("```"):
            match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
            if match:
                content = match.group(1).strip()
        try:
            return json.loads(content)
        except Exception as e:
            print(f"JSON parsing error: {e}. Content was: {content}")
            return None

    def get_interview_context(self, session_id: str):
        return get_interview_context(self.db, session_id)

    def get_question_details(self, question_id: str):
        if not self.db:
            return None
        try:
            from sqlalchemy import text
            with self.db._engine.connect() as conn:
                result = conn.execute(
                    text("SELECT title, difficulty FROM questions WHERE id = :id"),
                    {"id": question_id}
                )
                row = result.fetchone()
                if row:
                    return {
                        "title": row[0],
                        "difficulty": row[1]
                    }
        except Exception as e:
            print(f"Error fetching question details: {e}")
        return None

    def get_interview_state(self, session_id: str) -> str:
        if not self.redis_client:
            return "REQUIREMENTS"
        try:
            state = self.redis_client.get(f"interview_state:{session_id}")
            if state:
                return state.decode("utf-8")
        except Exception as e:
            print(f"Error getting interview state: {e}")
        return "REQUIREMENTS"

    def set_interview_state(self, session_id: str, state: str):
        if not self.redis_client:
            return
        try:
            self.redis_client.set(f"interview_state:{session_id}", state)
        except Exception as e:
            print(f"Error setting interview state: {e}")

    async def generate_initial_greeting(self, question_id: str, session_id: str) -> str:
        details = self.get_question_details(question_id)
        title = details["title"] if details else "System Design"
        difficulty = details["difficulty"] if details else "Senior"

        # Idempotency check: check if the session already has messages
        if settings.REDIS_URL:
            try:
                history = self.get_message_history(session_id)
                messages = history.messages
                if len(messages) > 0:
                    print(f"Session {session_id} already has chat history. Skipping duplicate initial greeting generation.")
                    for msg in messages:
                        if msg.type == "ai":
                            return msg.content
                    return f"Welcome back! Let's continue our system design interview for '{title}'."
            except Exception as history_err:
                print(f"Failed to check message history for idempotency: {history_err}")

        system_prompt = (
            f"You are a professional system design interviewer. "
            f"The candidate is starting a new interview to design '{title}' (Difficulty: {difficulty}). "
            f"Please generate a brief, professional opening greeting (1-2 sentences) introducing the question "
            f"and inviting the candidate to clarify requirements. Do not provide design solutions yet."
        )

        try:
            response = await self.llm.ainvoke(system_prompt)
            output = response.content

            if settings.REDIS_URL:
                history = self.get_message_history(session_id)
                history.clear()
                history.add_ai_message(output)
                self.set_interview_state(session_id, "REQUIREMENTS")

            return output
        except Exception as e:
            fallback = f"Welcome! Let's start the system design interview for '{title}'. What functional and non-functional requirements do you suggest we support?"
            if settings.REDIS_URL:
                try:
                    history = self.get_message_history(session_id)
                    history.clear()
                    history.add_ai_message(fallback)
                    self.set_interview_state(session_id, "REQUIREMENTS")
                except Exception as redis_err:
                    print(f"Failed to save fallback greeting to Redis history: {redis_err}")
            return fallback

    async def generate_response(self, prompt: str, session_id: str = "default") -> str:
        if not settings.GROQ_API_KEY:
            return "Error: GROQ_API_KEY is not set"

        try:
            title = "System Design"
            difficulty = "Senior"
            expected_topics = []

            ctx = self.get_interview_context(session_id)
            if ctx:
                title = ctx["title"]
                difficulty = ctx["difficulty"]
                expected_topics = ctx["expected_topics"]

            current_state = self.get_interview_state(session_id)

            retrieved_concepts = self.knowledge_service.retrieve_relevant_concepts(prompt, limit=2)
            concepts_context = ""
            if retrieved_concepts:
                concepts_context = "\n- ".join(retrieved_concepts)
                concepts_context = f"\nRelevant System Design Reference Material:\n- {concepts_context}\n"

            stages_guidelines = {
                "REQUIREMENTS": (
                    "Stage: Requirements Gathering.\n"
                    "Your goal is to guide the candidate to define functional and non-functional requirements (e.g. read/write ratio, scale, active users, latency constraints).\n"
                    "Do not let them skip to estimation or system design before they define functional and non-functional requirements.\n"
                    "If they have successfully listed both functional and non-functional requirements, transition to 'ESTIMATION'."
                ),
                "ESTIMATION": (
                    "Stage: Capacity Estimation.\n"
                    "Your goal is to guide the candidate to estimate the scaling parameters (QPS, database size, network bandwidth, memory for caching) based on the requirements they gathered.\n"
                    "If they have completed the calculations correctly (or made a reasonable attempt) and understand the scale, transition to 'HIGH_LEVEL'."
                ),
                "HIGH_LEVEL": (
                    "Stage: High-Level Design.\n"
                    "Your goal is to guide the candidate to describe the main architectural components (Load Balancer, Web Server, Database, Cache, Message Queue) and their connection flow.\n"
                    "Once they have laid out the core components and APIs, transition to 'DEEP_DIVE'."
                ),
                "DEEP_DIVE": (
                    "Stage: Detailed Component Design (Deep Dive).\n"
                    "Your goal is to guide the candidate to discuss detailed scaling issues (e.g. database sharding/partitioning, replication, cache eviction, handling hotspot users, data consistency, rate limiting).\n"
                    "Once you have deep dived into 2-3 critical areas of the system design, transition to 'COMPLETED'."
                ),
                "COMPLETED": (
                    "Stage: Interview Completed.\n"
                    "The interview is finished. Politely wrap up the conversation, thank the candidate, and inform them that their report is being generated."
                )
            }

            current_stage_info = stages_guidelines.get(current_state, stages_guidelines["REQUIREMENTS"])

            system_prompt = (
                f"You are a professional system design interviewer. "
                f"You are conducting a system design interview with a candidate for the topic: '{title}' (Difficulty: {difficulty}). "
                f"Expected Topics/Concepts to cover: {expected_topics}.\n\n"
                f"{concepts_context}\n"
                f"Current Interview State: {current_state}\n"
                f"{current_stage_info}\n\n"
                f"Guidelines:\n"
                f"1. Adopt a realistic, helpful but demanding interviewer persona.\n"
                f"2. Keep your conversational responses concise (maximum 3-4 sentences). Focus on asking one clear follow-up question or prompting the candidate for details related to the current stage.\n"
                f"3. Evaluate the candidate's last input and previous history to decide whether they have sufficiently completed the current stage and are ready to move to the next one.\n\n"
                f"You MUST output your response as a valid JSON object with the following structure:\n"
                f"{{\n"
                f'  "next_state": "REQUIREMENTS" | "ESTIMATION" | "HIGH_LEVEL" | "DEEP_DIVE" | "COMPLETED",\n'
                f'  "transition_reason": "Brief reason for staying in the current state or moving to the next state",\n'
                f'  "response": "Your follow-up question, guidance, or response to the candidate"\n'
                f"}}\n"
                f"Ensure the JSON output is well-formed. Do not wrap the JSON output in markdown formatting or code blocks."
            )

            diagram_ctx = self.eval_service.get_diagram_context(session_id)
            diagram_info = ""
            if diagram_ctx and (diagram_ctx["nodes"] or diagram_ctx["edges"]):
                parsed = self.diagram_service.parse_diagram_to_text(diagram_ctx)
                diagram_info = f"\nCandidate's current whiteboard diagram:\n{parsed}"

            history = None
            if settings.REDIS_URL:
                history = self.get_message_history(session_id)
                messages = history.messages
                context_list = []
                for msg in messages[-8:]:
                    context_list.append(f"{msg.type}: {msg.content}")
                context = "\n".join(context_list)
                full_prompt = (
                    f"{system_prompt}\n\n"
                    f"{diagram_info}\n\n"
                    f"Previous Conversation:\n{context}\n\n"
                    f"Candidate's Message: {prompt}\n"
                    f"Please evaluate and respond in the required JSON format."
                )
            else:
                full_prompt = (
                    f"{system_prompt}\n\n"
                    f"{diagram_info}\n\n"
                    f"Candidate's Message: {prompt}\n"
                    f"Please evaluate and respond in the required JSON format."
                )

            response = await self.llm.ainvoke(full_prompt)
            output = response.content

            parsed = self._parse_json_response(output)
            if parsed:
                next_state = parsed.get("next_state", current_state)
                ai_response = parsed.get("response", output)

                if next_state != current_state:
                    print(f"State transition for {session_id}: {current_state} -> {next_state} (Reason: {parsed.get('transition_reason')})")
                    self.set_interview_state(session_id, next_state)
                    
                    if next_state == "COMPLETED":
                        await self.publish_evaluation_request(session_id)

                output = ai_response

            if history:
                history.add_user_message(prompt)
                history.add_ai_message(output)

            return output

        except Exception as e:
            return f"Failed to generate response: {str(e)}"

    async def submit_interview(self, session_id: str) -> str:
        self.set_interview_state(session_id, "COMPLETED")
        await self.publish_evaluation_request(session_id)
        
        response = "Thank you! Your interview has been submitted. We are generating your feedback report now."
        if settings.REDIS_URL:
            try:
                history = self.get_message_history(session_id)
                history.add_ai_message(response)
            except Exception as e:
                print(f"Failed to append final submit greeting to Redis: {e}")
                
        return response

    async def process_diagram_event(self, session_id: str) -> str:
        diagram_ctx = self.eval_service.get_diagram_context(session_id)
        if not diagram_ctx:
            return ""
        parsed_diagram = self.diagram_service.parse_diagram_to_text(diagram_ctx)
        history = self.get_message_history(session_id)
        messages = history.messages
        if not messages:
            return ""
        critique = await self.diagram_service.generate_diagram_critique(
            session_id, parsed_diagram, messages
        )
        if critique.get("should_respond"):
            response = critique.get("response", "")
            if response and settings.REDIS_URL:
                try:
                    history.add_ai_message(response)
                except Exception as redis_err:
                    print(f"Failed to save diagram critique to Redis history: {redis_err}")
            return response
        return ""
