from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    PROJECT_NAME: str = "AI Engine API"
    VERSION: str = "1.0.0"
    API_V1_STR: str = "/api/v1"

    HUGGINGFACEHUB_API_TOKEN: str = ""
    model_config = SettingsConfigDict(
        env_file=".env", 
        env_file_encoding="utf-8", 
        extra="ignore"
    )
settings = Settings()