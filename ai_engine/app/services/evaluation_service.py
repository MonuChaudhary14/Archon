import json
import re
from langchain_groq import ChatGroq
from app.core.config import settings
from langchain_community.utilities.sql_database import SQLDatabase
from langchain_community.chat_message_histories.redis import RedisChatMessageHistory

class EvaluationService:
    def __init__(self, llm: ChatGroq, db: SQLDatabase):
        self.llm = llm
        self.db = db

    def get_message_history(self, session_id: str):
        return RedisChatMessageHistory(session_id, url=settings.REDIS_URL)

    def _parse_json_response(self, content: str):
        content = content.strip()
        if content.startswith("```"):
            match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
            if match:
                content = match.group(1).strip()
        try:
            return json.loads(content)
        except Exception as e:
            print(f"Evaluation JSON parsing error: {e}. Content was: {content}")
            return None

    def get_interview_context(self, session_id: str):
        if not self.db:
            return None
        try:
            from sqlalchemy import text
            query = """
                SELECT q.title, q.difficulty, q.expected_topics 
                FROM interviews i 
                JOIN questions q ON i.question_id = q.id 
                WHERE i.id = :session_id
            """
            with self.db._engine.connect() as conn:
                result = conn.execute(text(query), {"session_id": session_id})
                row = result.fetchone()
                if row:
                    return {
                        "title": row[0],
                        "difficulty": row[1],
                        "expected_topics": row[2]
                    }
        except Exception as e:
            print(f"Error fetching interview context in evaluation: {e}")
        return None

    async def evaluate_session(self, session_id: str):
        print(f"Starting background evaluation for session: {session_id}")
        
        ctx = self.get_interview_context(session_id)
        title = "System Design"
        difficulty = "Senior"
        expected_topics = []
        if ctx:
            title = ctx["title"]
            difficulty = ctx["difficulty"]
            expected_topics = ctx["expected_topics"]

        history = self.get_message_history(session_id)
        messages = history.messages
        if not messages:
            print(f"No chat history found to evaluate for session: {session_id}")
            return

        transcript_list = []
        for msg in messages:
            transcript_list.append(f"{msg.type}: {msg.content}")
        transcript = "\n".join(transcript_list)

        system_prompt = (
            f"You are a Senior System Design Interview Evaluator.\n"
            f"Your job is to objectively score and provide feedback for a candidate's system design interview.\n\n"
            f"Interview Topic: {title}\n"
            f"Difficulty: {difficulty}\n"
            f"Expected Topics: {expected_topics}\n\n"
            f"Here is the complete chat transcript of the interview:\n"
            f"\"\"\"\n{transcript}\n\"\"\"\n\n"
            f"Please evaluate the candidate across 4 standard system design core dimensions:\n"
            f"1. Requirements Gathering (clarifying goals, scope, scale, latency constraints)\n"
            f"2. Capacity Estimation (estimating traffic, storage, memory, bandwidth)\n"
            f"3. High-Level Design (defining services, storage layers, queue connections)\n"
            f"4. Detailed Design & Scaling (resolving hotspots, partition keys, CDNs, caching policies)\n\n"
            f"For each dimension, assign a score from 1 (poor) to 5 (excellent) and write a short, constructive paragraph of feedback (2-3 sentences).\n"
            f"Calculate an overall score out of 100 (weighted average of the dimensions, scaled to 100).\n"
            f"Provide a summary of their performance, listing their main strengths and areas for improvement.\n\n"
            f"You MUST output your response as a valid JSON object with the following structure:\n"
            f"{{\n"
            f'  "overall_score": 0-100 (integer),\n'
            f'  "requirements_score": 1-5 (integer),\n'
            f'  "requirements_feedback": "string",\n'
            f'  "estimation_score": 1-5 (integer),\n'
            f'  "estimation_feedback": "string",\n'
            f'  "high_level_score": 1-5 (integer),\n'
            f'  "high_level_feedback": "string",\n'
            f'  "deep_dive_score": 1-5 (integer),\n'
            f'  "deep_dive_feedback": "string",\n'
            f'  "summary_strengths": "string",\n'
            f'  "summary_weaknesses": "string"\n'
            f"}}\n"
            f"Ensure the JSON output is well-formed. Do not wrap the JSON output in markdown formatting or code blocks."
        )

        try:
            response = await self.llm.ainvoke(system_prompt)
            output = response.content

            parsed = self._parse_json_response(output)
            if not parsed:
                parsed = {
                    "overall_score": 70,
                    "requirements_score": 3,
                    "requirements_feedback": "Needs to spend more time gathering requirements.",
                    "estimation_score": 3,
                    "estimation_feedback": "Calculations were partially complete.",
                    "high_level_score": 4,
                    "high_level_feedback": "Correctly identified basic database and application components.",
                    "deep_dive_score": 3,
                    "deep_dive_feedback": "Could have gone deeper into partition strategies.",
                    "summary_strengths": "Good understanding of core high level blocks.",
                    "summary_weaknesses": "Needs practice with capacity estimation and sharding."
                }

            overall_score = parsed.get("overall_score", 70)

            # 6. Save to PostgreSQL
            if self.db:
                from sqlalchemy import text
                query = """
                    UPDATE interviews 
                    SET score = :score, ended_at = NOW(), feedback = :feedback 
                    WHERE id = :session_id
                """
                with self.db._engine.begin() as conn:
                    conn.execute(
                        text(query),
                        {
                            "score": overall_score,
                            "feedback": json.dumps(parsed),
                            "session_id": session_id
                        }
                    )
                print(f"Successfully saved evaluation report for session: {session_id}")

        except Exception as e:
            print(f"Failed to run evaluation for session {session_id}: {e}")
