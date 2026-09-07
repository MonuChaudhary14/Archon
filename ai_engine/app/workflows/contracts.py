from dataclasses import dataclass
from typing import List, Dict, Any, Optional

@dataclass
class EvaluationWorkflowInput:
    session_id: str

@dataclass
class InterviewContextData:
    session_id: str
    title: str
    difficulty: str
    expected_topics: List[str]
    transcript: str
    diagram_nodes: List[Dict[str, Any]]
    diagram_edges: List[Dict[str, Any]]
    has_candidate_messages: bool
    has_diagram: bool

@dataclass
class EvaluationExecutionPayload:
    context: Dict[str, Any]
    knowledge_rubrics: List[Dict[str, Any]]

@dataclass
class PersistReportPayload:
    session_id: str
    score: int
    feedback: Dict[str, Any]

@dataclass
class CompensationPayload:
    session_id: str
    error_message: str
