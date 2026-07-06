from langchain_groq import ChatGroq
from app.core.config import settings

class LLMService:
    def __init__(self):
        self.llm = ChatGroq(
            model="llama-3.1-8b-instant",
            temperature=0.7,
            groq_api_key=settings.GROQ_API_KEY
        )
    
    async def generate_response(self, prompt: str) -> str:
        """Sends a prompt to the LLM and returns the response"""
        if not settings.GROQ_API_KEY:
            return "Error: GROQ_API_KEY is not set"

        try:
            response = await self.llm.ainvoke(prompt)
            return response.content
        except Exception as e:
            return f"Failed to generate response: {str(e)}"
