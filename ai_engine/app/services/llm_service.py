from langchain_community.llms import HuggingFaceEndpoint
from app.core.config import settings

class LLMService:
    def __init__(self):
        
        self.llm = HuggingFaceEndpoint(
            repo_id="mistralai/Mistral-7B-Instruct-v0.2",
            huggingfacehub_api_token=settings.HUGGINGFACEHUB_API_TOKEN,
            temperature=0.7
        )
    
    async def generate_response(self, prompt: str) -> str:
        """Sends a prompt to the LLM and returns the response"""
        if not settings.HUGGINGFACEHUB_API_TOKEN:
            return "Error: HUGGINGFACEHUB_API_TOKEN is not set"

        try:
            response = await self.llm.ainvoke(prompt)
            return response
        except Exception as e:
            return f"Failed to generate response: {str(e)}"
