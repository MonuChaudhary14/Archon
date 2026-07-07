from langchain_groq import ChatGroq
from app.core.config import settings
from langchain_community.utilities.sql_database import SQLDatabase
from langchain_community.agent_toolkits.sql.base import create_sql_agent
from langchain_community.chat_message_histories.redis import RedisChatMessageHistory

class LLMService:
    def __init__(self):
        self.llm = ChatGroq(
            model="llama-3.1-8b-instant",
            temperature=0.7,
            groq_api_key=settings.GROQ_API_KEY
        )

        self.db = None

        if settings.DATABASE_URL:
            self.db = SQLDatabase.from_uri(settings.DATABASE_URL)
            self.agent_executor = create_sql_agent(
                llm=self.llm,
                db=self.db,
                agent_type="tool-calling",
                verbose=True
            )

    def get_message_history(self, session_id:str):
        return RedisChatMessageHistory(session_id, url=settings.REDIS_URL)


    async def generate_response(self, prompt: str, session_id : str = "default") -> str:
        if not settings.GROQ_API_KEY:
            return "Error: GROQ_API_KEY is not set"

        try:
            if settings.REDIS_URL:
                history = self.get_message_history(session_id)
                messages = history.messages

                context = "\n".join([f"{msg.type} : {msg.content}" for msg in messages[-5:]])
                full_prompt = f"Previous Conversation:\n{context}\n\nUser Question: {prompt}\n(Use the database if needed to answer the question, or just answer normally if it's a casual question.)"
            else:
                full_prompt = prompt

            if self.db:
                response = await self.agent_executor.ainvoke({"input": full_prompt})
                output = response.get("output", str(response))

            else:
                response = await self.llm.ainvoke(full_prompt)
                output = response.content

            if settings.REDIS_URL:
                history.add_user_message(prompt)
                history.add_ai_message(output)
                
            return output

        except Exception as e:
            return f"Failed to generate response: {str(e)}"
