from fastapi import FastAPI, Depends
from pydantic import BaseModel
from app.core.config import settings
from app.services.llm_service import LLMService

app = FastAPI(
    title = settings.PROJECT_NAME,
    version=settings.VERSION,
    openapi_url = f"{settings.API_V1_STR}/openapi.json"
)

def get_llm_service() -> LLMService:
    return LLMService()

class PromptRequest(BaseModel):
    prompt : str
    session_id : str = "default"

class PromptResponse(BaseModel):
    response: str

@app.get("/health")
async def health_check():
    return {"status":"healthy"}

@app.post(f"{settings.API_V1_STR}/generate",response_model = PromptResponse)
async def generate_response(request : PromptRequest, llm_service: LLMService= Depends(get_llm_service)):
    result = await llm_service.generate_response(request.prompt,request.session_id)
    return PromptResponse(response=result) 

  