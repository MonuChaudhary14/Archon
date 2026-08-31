import json
import re
from app.core.helpers import get_message_history
from app.core.interfaces import LLMClient, InterviewRepository

class EvaluationService:
    def __init__(self, llm: LLMClient, repo: InterviewRepository):
        self.llm = llm
        self.repo = repo

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
            print(f"Evaluation JSON parsing error: {e}. Content was: {content}")
            return None

    async def evaluate_session(self, session_id: str):
        print(f"Starting background evaluation for session: {session_id}")
        
        ctx = self.repo.get_interview_context(session_id)
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

        diagram_ctx = self.repo.get_diagram_context(session_id)
        diagram_info = ""
        if diagram_ctx and (diagram_ctx["nodes"] or diagram_ctx["edges"]):
            diagram_info = f"\n\nHere is the candidate's final system design whiteboard diagram:\nNodes:\n{json.dumps(diagram_ctx['nodes'], indent=2)}\nEdges:\n{json.dumps(diagram_ctx['edges'], indent=2)}"

        system_prompt = (
            f"You are a Senior System Design Interview Evaluator.\n"
            f"Your job is to objectively score and provide feedback for a candidate's system design interview.\n\n"
            f"Interview Topic: {title}\n"
            f"Difficulty: {difficulty}\n"
            f"Expected Topics: {expected_topics}\n\n"
            f"Here is the complete chat transcript of the interview:\n"
            f"\"\"\"\n{transcript}\n\"\"\"{diagram_info}\n\n"
            f"Please evaluate the candidate across 4 standard system design core dimensions:\n"
            f"1. Requirements Gathering (clarifying goals, scope, scale, latency constraints)\n"
            f"2. Capacity Estimation (estimating traffic, storage, memory, bandwidth)\n"
            f"3. High-Level Design (defining services, storage layers, queue connections)\n"
            f"4. Detailed Design & Scaling (resolving hotspots, partition keys, CDNs, caching policies)\n\n"
            f"For each dimension, assign a score from 1 (poor) to 5 (excellent) and write a short, constructive paragraph of feedback (2-3 sentences).\n"
            f"Calculate an overall score out of 100 (weighted average of the dimensions, scaled to 100).\n"
            f"Provide a summary of their performance, listing their main strengths and areas for improvement.\n"
            f"Based on their weaknesses, recommend a personalized learning path containing system design topics, study reasons, and references.\n\n"
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
            f'  "summary_weaknesses": "string",\n'
            f'  "personalized_learning_path": [\n'
            f'    {{\n'
            f'      "topic": "string",\n'
            f'      "reason": "string",\n'
            f'      "resources": ["string"]\n'
            f'    }}\n'
            f'  ]\n'
            f"}}\n"
            f"Ensure the JSON output is well-formed. Do not wrap the JSON output in markdown formatting or code blocks."
        )

        try:
            response = await self.llm.ainvoke(system_prompt)
            output = response.content

            parsed = self._parse_json_response(output)
            if not parsed:
                raise ValueError("LLM generated response was not in the expected JSON format")

            overall_score = parsed.get("overall_score", 70)

            self.repo.save_evaluation_report(session_id, overall_score, parsed)
            print(f"Successfully saved evaluation report for session: {session_id}")

        except Exception as e:
            print(f"Failed to run evaluation for session {session_id}: {e}")
            try:
                self.repo.save_evaluation_error(session_id, str(e))
                print(f"Successfully saved error status for session: {session_id}")
            except Exception as db_err:
                print(f"Failed to save error status to database: {db_err}")
