from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    PROJECT_NAME: str = "AI Engine API"
    VERSION: str = "1.0.0"
    API_V1_STR: str = "/api/v1"

    LLM_PROVIDER: str = "gemini"
    GEMINI_API_KEY: str = ""
    GEMINI_API_KEYS: str = ""
    GEMINI_MODEL: str = "gemini-flash-latest"
    GROQ_API_KEY: str = ""
    GROQ_API_KEYS: str = ""
    GROQ_MODEL: str = "openai/gpt-oss-20b"
    NVIDIA_API_KEY: str = ""
    NVIDIA_API_KEYS: str = ""
    NVIDIA_MODEL: str = "mistralai/mistral-nemotron"
    NVIDIA_BASE_URL: str = "https://integrate.api.nvidia.com/v1"
    DATABASE_URL: str = ""
    REDIS_URL: str = ""
    KAFKA_BROKERS: str = ""
    QDRANT_URL: str = ""
    HUGGINGFACEHUB_API_TOKEN: str = ""
    TEMPORAL_HOST: str = ""
    TEMPORAL_NAMESPACE: str = "default"
    model_config = SettingsConfigDict(
        env_file=(".env", "../.env"), 
        env_file_encoding="utf-8", 
        extra="ignore"
    )

    def get_database_url(self) -> str:
        if self.DATABASE_URL.startswith("postgres://"):
            return self.DATABASE_URL.replace("postgres://", "postgresql://", 1)
        return self.DATABASE_URL

settings = Settings()