import json
import re
from typing import Dict, Any, List
from temporalio import activity
from app.core.config import settings
from app.core.helpers import get_message_history
from app.core.key_rotator import LLMKeyRotator
from app.services.knowledge_service import KnowledgeService
from app.services.interview_repository import SQLInterviewRepository
from langchain_community.utilities.sql_database import SQLDatabase

def _parse_json(content: str) -> Dict[str, Any] | None:
    content = content.strip()
    if content.startswith("```"):
        match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
        if match:
            content = match.group(1).strip()
    try:
        return json.loads(content)
    except Exception:
        return None

class EvaluationActivities:
    def __init__(self, repo: SQLInterviewRepository | None = None, llm: LLMKeyRotator | None = None, knowledge: KnowledgeService | None = None):
        self.db = None
        db_url = settings.get_database_url()
        if db_url:
            try:
                self.db = SQLDatabase.from_uri(db_url)
            except Exception:
                pass
        self.repo = repo or SQLInterviewRepository(self.db)
        self.llm = llm or LLMKeyRotator()
        self.knowledge = knowledge or KnowledgeService()

    @activity.defn
    async def fetch_interview_context(self, session_id: str) -> Dict[str, Any]:
        ctx = self.repo.get_interview_context(session_id)
        if not ctx:
            raise ValueError(f"Interview session context not found for session: {session_id}")

        history = get_message_history(session_id)
        messages = history.messages if history else []

        human_messages = [msg for msg in messages if getattr(msg, "type", "") in ("human", "user")]

        diagram_ctx = self.repo.get_diagram_context(session_id)
        diagram_nodes = diagram_ctx.get("nodes", []) if diagram_ctx else []
        diagram_edges = diagram_ctx.get("edges", []) if diagram_ctx else []

        transcript_list = [f"{msg.type}: {msg.content}" for msg in messages]
        transcript = "\n".join(transcript_list)

        return {
            "session_id": session_id,
            "title": ctx.get("title", ""),
            "difficulty": ctx.get("difficulty", ""),
            "expected_topics": ctx.get("expected_topics", []),
            "transcript": transcript,
            "human_message_count": len(human_messages),
            "diagram_nodes": diagram_nodes,
            "diagram_edges": diagram_edges,
            "has_diagram": bool(diagram_nodes or diagram_edges)
        }

    @activity.defn
    async def search_knowledge_rubrics(self, topics: List[str]) -> List[Dict[str, Any]]:
        rubrics = []
        for topic in topics:
            try:
                results = self.knowledge.search(topic, limit=2)
                for r in results:
                    rubrics.append({
                        "topic": topic,
                        "content": r.get("content", ""),
                        "score": r.get("score", 0.0)
                    })
            except Exception:
                pass
        return rubrics

    @activity.defn
    async def execute_llm_evaluation(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        ctx = payload.get("context", {})
        title = ctx.get("title", "")
        difficulty = ctx.get("difficulty", "")
        expected_topics = ctx.get("expected_topics", [])
        transcript = ctx.get("transcript", "")
        human_msg_count = ctx.get("human_message_count", 0)
        diagram_nodes = ctx.get("diagram_nodes", [])
        diagram_edges = ctx.get("diagram_edges", [])
        has_diagram = ctx.get("has_diagram", False)
        knowledge_rubrics = payload.get("knowledge_rubrics", [])

        if human_msg_count == 0 and not has_diagram:
            return {
                "overall_score": 0,
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
                "diagram_components": []
            }

        diagram_info = ""
        diagram_comp_names = [n.get("label", n.get("type", "node")) for n in diagram_nodes if isinstance(n, dict)]
        if has_diagram:
            diagram_info = f"\n\nCandidate's final whiteboard diagram:\nNodes:\n{json.dumps(diagram_nodes, indent=2)}\nEdges:\n{json.dumps(diagram_edges, indent=2)}"

        knowledge_context = ""
        if knowledge_rubrics:
            knowledge_snippets = "\n".join([f"- {k.get('topic')}: {k.get('content')}" for k in knowledge_rubrics[:4]])
            knowledge_context = f"\n\nReference Architecture Knowledge Base:\n{knowledge_snippets}"

        system_prompt = (
            f"You are a Senior System Design Interview Evaluator.\n"
            f"Your job is to objectively score and provide feedback for a candidate's system design interview.\n\n"
            f"Interview Topic: {title}\n"
            f"Difficulty: {difficulty}\n"
            f"Expected Topics: {expected_topics}\n\n"
            f"Here is the complete chat transcript of the interview:\n"
            f"\"\"\"\n{transcript}\n\"\"\"{diagram_info}{knowledge_context}\n\n"
            f"CRITICAL EVALUATION RULES:\n"
            f"1. Base your scoring SOLELY on the candidate's actual words and diagrams provided above.\n"
            f"2. DO NOT invent, hallucinate, or assume candidate solutions that are not explicitly present in the transcript or diagram.\n"
            f"3. If a candidate gave minimal answers, assign low scores (0-40) reflecting the lack of depth.\n"
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
        parsed = _parse_json(response.content)
        if not parsed or not isinstance(parsed, dict):
            raise ValueError("LLM generated response was not in the expected JSON format")

        rubrics = parsed.get("rubrics", [])
        if not isinstance(rubrics, list) or len(rubrics) == 0:
            raise ValueError("Evaluation response is missing rubrics breakdown")

        overall_score = parsed.get("overall_score")
        if overall_score is None:
            total_weighted = sum(r.get("score", 0) * (r.get("weight", 25) / 100.0) for r in rubrics if isinstance(r, dict))
            overall_score = int(round(total_weighted))
        parsed["overall_score"] = overall_score

        if not parsed.get("diagram_components") and diagram_comp_names:
            parsed["diagram_components"] = diagram_comp_names

        return parsed

    @activity.defn
    async def persist_evaluation_report(self, payload: Dict[str, Any]) -> bool:
        session_id = payload.get("session_id")
        score = payload.get("score", 0)
        feedback = payload.get("feedback", {})
        if not session_id:
            raise ValueError("session_id is required to persist evaluation report")
        self.repo.save_evaluation_report(session_id, score, feedback)
        return True

    @activity.defn
    async def compensate_evaluation_failure(self, payload: Dict[str, Any]) -> bool:
        session_id = payload.get("session_id")
        error_message = payload.get("error_message", "Unknown evaluation failure")
        if not session_id:
            return False
        self.repo.save_evaluation_error(session_id, error_message)
        return True
