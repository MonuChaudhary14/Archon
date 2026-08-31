from langchain_community.chat_message_histories.redis import RedisChatMessageHistory
from app.core.config import settings
from sqlalchemy import text

def get_message_history(session_id: str) -> RedisChatMessageHistory:
    return RedisChatMessageHistory(session_id, url=settings.REDIS_URL)

def get_interview_context(db, session_id: str) -> dict | None:
    if not db:
        return None
    try:
        query = """
            SELECT q.title, q.difficulty, q.expected_topics 
            FROM interviews i 
            JOIN questions q ON i.question_id = q.id 
            WHERE i.id = :session_id
        """
        with db._engine.connect() as conn:
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
