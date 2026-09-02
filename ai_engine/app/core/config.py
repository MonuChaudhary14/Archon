from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    PROJECT_NAME: str = "AI Engine API"
    VERSION: str = "1.0.0"
    API_V1_STR: str = "/api/v1"

    GROQ_API_KEY: str = ""
    GROQ_MODEL: str = "llama-3.3-70b-versatile"
    DATABASE_URL: str = ""
    REDIS_URL: str = ""
    KAFKA_BROKERS: str = ""
    QDRANT_URL: str = ""
    HUGGINGFACEHUB_API_TOKEN: str = ""
    model_config = SettingsConfigDict(
        env_file=".env", 
        env_file_encoding="utf-8", 
        extra="ignore"
    )

    def get_database_url(self) -> str:
        if self.DATABASE_URL.startswith("postgres://"):
            return self.DATABASE_URL.replace("postgres://", "postgresql://", 1)
        return self.DATABASE_URL

settings = Settings()