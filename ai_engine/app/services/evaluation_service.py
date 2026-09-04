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
        
        try:
            ctx = self.repo.get_interview_context(session_id)
            if not ctx:
                raise ValueError(f"Interview session context not found for session: {session_id}")

            title = ctx.get("title", "")
            difficulty = ctx.get("difficulty", "")
            expected_topics = ctx.get("expected_topics", [])

            history = self.get_message_history(session_id)
            messages = history.messages if history else []

            human_messages = []
            for msg in messages:
                msg_type = getattr(msg, "type", "")
                if msg_type in ("human", "user"):
                    human_messages.append(msg)

            diagram_ctx = self.repo.get_diagram_context(session_id)
            diagram_nodes = diagram_ctx.get("nodes", []) if diagram_ctx else []
            diagram_edges = diagram_ctx.get("edges", []) if diagram_ctx else []
            has_diagram = bool(diagram_nodes or diagram_edges)

            if len(human_messages) == 0 and not has_diagram:
                zero_report = {
                    "interviewer_summary": "The interview was submitted without candidate responses or whiteboard diagrams. No technical evaluation could be performed.",
                    "rubrics": [
                        {
                            "name": "Requirements & Scope",
                            "score": 0,
                            "weight": 20,
                            "summary": "No functional or non-functional requirements were specified.",
                            "feedback_points": ["Candidate submitted the session without clarifying requirements or scope."]
                        },
                        {
                            "name": "Capacity Estimation",
                            "score": 0,
                            "weight": 20,
                            "summary": "No capacity or scale calculations were attempted.",
                            "feedback_points": ["Candidate submitted the session without estimating traffic, storage, or memory."]
                        },
                        {
                            "name": "High-Level Architecture",
                            "score": 0,
                            "weight": 30,
                            "summary": "No architecture or diagram components were designed.",
                            "feedback_points": ["No services, data stores, or communication patterns were designed."]
                        },
                        {
                            "name": "Scalability & Deep Dive",
                            "score": 0,
                            "weight": 30,
                            "summary": "No scaling strategies or trade-offs were discussed.",
                            "feedback_points": ["No caching, partitioning, or fault tolerance was addressed."]
                        }
                    ],
                    "strengths": [],
                    "weaknesses": ["Interview session was ended with no candidate answers or whiteboard diagrams."],
                    "recommendations": [
                        f"Start a full practice session on '{title}' by asking clarifying questions.",
                        "Draft high-level components on the whiteboard canvas.",
                        "Practice back-of-the-envelope capacity estimations."
                    ],
                    "diagram_components": [],
                    "overall_score": 0
                }
                self.repo.save_evaluation_report(session_id, 0, zero_report)
                print(f"Saved zero-participation evaluation report for session: {session_id}")
                return

            transcript_list = []
            for msg in messages:
                transcript_list.append(f"{msg.type}: {msg.content}")
            transcript = "\n".join(transcript_list)

            diagram_info = ""
            diagram_comp_names = [n.get("label", n.get("type", "node")) for n in diagram_nodes if isinstance(n, dict)]
            if has_diagram:
                diagram_info = f"\n\nCandidate's final whiteboard diagram:\nNodes:\n{json.dumps(diagram_nodes, indent=2)}\nEdges:\n{json.dumps(diagram_edges, indent=2)}"

            system_prompt = (
                f"You are a Senior System Design Interview Evaluator.\n"
                f"Your job is to objectively score and provide feedback for a candidate's system design interview.\n\n"
                f"Interview Topic: {title}\n"
                f"Difficulty: {difficulty}\n"
                f"Expected Topics: {expected_topics}\n\n"
                f"Here is the complete chat transcript of the interview:\n"
                f"\"\"\"\n{transcript}\n\"\"\"{diagram_info}\n\n"
                f"CRITICAL EVALUATION RULES:\n"
                f"1. Base your scoring SOLELY on the candidate's actual words and diagrams provided above.\n"
                f"2. DO NOT invent, hallucinate, or assume candidate solutions that are not explicitly present in the transcript or diagram.\n"
                f"3. If a candidate gave minimal answers (e.g. only 1-2 brief sentences or skipped stages), assign low scores (0-40) reflecting the lack of depth.\n"
                f"4. If a candidate performed well with thorough technical justification, score accordingly (70-95).\n\n"
                f"Evaluate across 4 core dimensions:\n"
                f"- Requirements & Scope (weight: 20)\n"
                f"- Capacity Estimation (weight: 20)\n"
                f"- High-Level Architecture (weight: 30)\n"
                f"- Scalability & Deep Dive (weight: 30)\n\n"
                f"You MUST output your response as a valid JSON object matching this EXACT schema:\n"
                f"{{\n"
                f'  "overall_score": 0-100 (integer, weighted score across the rubrics),\n'
                f'  "interviewer_summary": "Concise 2-3 sentence executive summary of candidate performance",\n'
                f'  "rubrics": [\n'
                f'    {{\n'
                f'      "name": "Requirements & Scope",\n'
                f'      "score": 0-100 (integer),\n'
                f'      "weight": 20,\n'
                f'      "summary": "Short 1-2 sentence summary for this category",\n'
                f'      "feedback_points": ["Specific point 1", "Specific point 2"]\n'
                f'    }},\n'
                f'    {{\n'
                f'      "name": "Capacity Estimation",\n'
                f'      "score": 0-100 (integer),\n'
                f'      "weight": 20,\n'
                f'      "summary": "Short 1-2 sentence summary for this category",\n'
                f'      "feedback_points": ["Specific point 1"]\n'
                f'    }},\n'
                f'    {{\n'
                f'      "name": "High-Level Architecture",\n'
                f'      "score": 0-100 (integer),\n'
                f'      "weight": 30,\n'
                f'      "summary": "Short 1-2 sentence summary for this category",\n'
                f'      "feedback_points": ["Specific point 1"]\n'
                f'    }},\n'
                f'    {{\n'
                f'      "name": "Scalability & Deep Dive",\n'
                f'      "score": 0-100 (integer),\n'
                f'      "weight": 30,\n'
                f'      "summary": "Short 1-2 sentence summary for this category",\n'
                f'      "feedback_points": ["Specific point 1"]\n'
                f'    }}\n'
                f'  ],\n'
                f'  "strengths": ["Clear strength observed in transcript"],\n'
                f'  "weaknesses": ["Clear area of improvement observed in transcript"],\n'
                f'  "recommendations": ["Actionable study topic or architectural concept to review"],\n'
                f'  "diagram_components": ["Component1", "Component2"]\n'
                f"}}\n"
                f"Ensure the JSON output is well-formed. Do not wrap the JSON output in markdown formatting or code blocks."
            )

            response = await self.llm.ainvoke(system_prompt)
            output = response.content

            parsed = self._parse_json_response(output)
            if not parsed or not isinstance(parsed, dict):
                raise ValueError("LLM generated response was not in the expected JSON format")

            rubrics = parsed.get("rubrics", [])
            if not isinstance(rubrics, list) or len(rubrics) == 0:
                raise ValueError("Evaluation response is missing rubrics breakdown")

            overall_score = parsed.get("overall_score")
            if overall_score is None:
                total_weighted = sum(r.get("score", 0) * (r.get("weight", 25) / 100.0) for r in rubrics if isinstance(r, dict))
                overall_score = int(round(total_weighted))

            if not parsed.get("diagram_components") and diagram_comp_names:
                parsed["diagram_components"] = diagram_comp_names

            self.repo.save_evaluation_report(session_id, overall_score, parsed)
            print(f"Successfully saved evaluation report for session: {session_id}")

        except Exception as e:
            print(f"Failed to run evaluation for session {session_id}: {e}")
            try:
                self.repo.save_evaluation_error(session_id, str(e))
                print(f"Successfully saved error status for session: {session_id}")
            except Exception as db_err:
                print(f"Failed to save error status to database: {db_err}")

