import json
from typing import Dict, Any
from sqlalchemy import text
from langchain_community.utilities.sql_database import SQLDatabase

class SQLInterviewRepository:
    def __init__(self, db: SQLDatabase):
        self.db = db

    def get_interview_context(self, session_id: str) -> Dict[str, Any] | None:
        if not self.db:
            return None
        try:
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
            print(f"Error fetching interview context: {e}")
        return None

    def get_diagram_context(self, session_id: str) -> Dict[str, Any] | None:
        if not self.db:
            return None
        try:
            query_nodes = "SELECT id, type, label FROM diagram_nodes WHERE interview_id = :session_id"
            query_edges = "SELECT id, source, target, type FROM diagram_edges WHERE interview_id = :session_id"
            with self.db._engine.connect() as conn:
                nodes_res = conn.execute(text(query_nodes), {"session_id": session_id})
                edges_res = conn.execute(text(query_edges), {"session_id": session_id})
                nodes = [{"id": r[0], "type": r[1], "label": r[2]} for r in nodes_res.fetchall()]
                edges = [{"id": r[0], "source": r[1], "target": r[2], "type": r[3]} for r in edges_res.fetchall()]
                return {"nodes": nodes, "edges": edges}
        except Exception as e:
            print(f"Error fetching diagram context: {e}")
        return None

    def save_evaluation_report(self, session_id: str, score: int, feedback: Dict[str, Any]) -> None:
        if not self.db:
            return
        query = """
            UPDATE interviews 
            SET score = :score, ended_at = NOW(), feedback = :feedback 
            WHERE id = :session_id
        """
        with self.db._engine.begin() as conn:
            conn.execute(
                text(query),
                {
                    "score": score,
                    "feedback": json.dumps(feedback),
                    "session_id": session_id
                }
            )

    def save_evaluation_error(self, session_id: str, error_message: str) -> None:
        if not self.db:
            return
        query = """
            UPDATE interviews 
            SET score = -1, ended_at = NOW(), feedback = :feedback 
            WHERE id = :session_id
        """
        error_payload = {"error": f"Evaluation failed: {error_message}"}
        with self.db._engine.begin() as conn:
            conn.execute(
                text(query),
                {
                    "feedback": json.dumps(error_payload),
                    "session_id": session_id
                }
            )
