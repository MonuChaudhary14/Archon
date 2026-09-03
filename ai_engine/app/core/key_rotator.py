import time
import asyncio
import logging
import requests
from types import SimpleNamespace
from typing import List, Dict, Any, Optional
from langchain_groq import ChatGroq
from app.core.config import settings

logger = logging.getLogger(__name__)

class GeminiClient:
    def __init__(self, api_key: str, model: str = "gemini-flash-latest"):
        self.api_key = api_key
        self.model = model
        self.url = f"https://generativelanguage.googleapis.com/v1beta/models/{self.model}:generateContent?key={self.api_key}"

    async def ainvoke(self, prompt: Any) -> Any:
        prompt_text = prompt if isinstance(prompt, str) else str(prompt)
        payload = {
            "contents": [
                {
                    "parts": [
                        {"text": prompt_text}
                    ]
                }
            ]
        }
        loop = asyncio.get_running_loop()
        res = await loop.run_in_executor(
            None,
            lambda: requests.post(self.url, json=payload, timeout=30)
        )
        
        if res.status_code == 200:
            data = res.json()
            try:
                text = data["candidates"][0]["content"]["parts"][0]["text"]
                return SimpleNamespace(content=text)
            except (KeyError, IndexError):
                raise ValueError(f"Unexpected Gemini API response structure: {data}")
        else:
            raise RuntimeError(f"Gemini API HTTP {res.status_code}: {res.text}")

class LLMKeyRotator:
    def __init__(self):
        self.cooldown_duration = 60.0
        self.cooldowns: Dict[str, float] = {}

        self.gemini_keys: List[str] = self._parse_keys(settings.GEMINI_API_KEYS, settings.GEMINI_API_KEY)
        self.groq_keys: List[str] = self._parse_keys(settings.GROQ_API_KEYS, settings.GROQ_API_KEY)

        self.gemini_index = 0
        self.groq_index = 0

    def _parse_keys(self, keys_str: str, single_key: str) -> List[str]:
        keys = []
        if keys_str:
            for k in keys_str.split(","):
                k_clean = k.strip()
                if k_clean and k_clean not in keys:
                    keys.append(k_clean)
        if single_key and single_key.strip() and single_key.strip() not in keys:
            keys.append(single_key.strip())
        return keys

    def _is_key_active(self, key: str) -> bool:
        cooldown_until = self.cooldowns.get(key, 0.0)
        return time.time() >= cooldown_until

    def _mark_cooldown(self, key: str):
        self.cooldowns[key] = time.time() + self.cooldown_duration
        logger.warning(f"Key {key[:6]}... placed on cooldown for {self.cooldown_duration}s")

    def _create_client(self, provider: str, api_key: str):
        if provider == "gemini":
            model_name = settings.GEMINI_MODEL or "gemini-flash-latest"
            return GeminiClient(api_key=api_key, model=model_name)
        elif provider == "groq":
            model_name = settings.GROQ_MODEL or "llama-3.3-70b-versatile"
            return ChatGroq(
                model=model_name,
                groq_api_key=api_key,
                temperature=0.7
            )
        else:
            raise ValueError(f"Unsupported LLM provider: {provider}")

    def _get_active_key(self, provider: str) -> Optional[str]:
        keys = self.gemini_keys if provider == "gemini" else self.groq_keys
        if not keys:
            return None

        start_index = self.gemini_index if provider == "gemini" else self.groq_index
        n = len(keys)

        for i in range(n):
            idx = (start_index + i) % n
            key = keys[idx]
            if self._is_key_active(key):
                if provider == "gemini":
                    self.gemini_index = (idx + 1) % n
                else:
                    self.groq_index = (idx + 1) % n
                return key
        return None

    def _is_rate_limit_error(self, err: Exception) -> bool:
        err_str = str(err).lower()
        keywords = ["429", "503", "rate limit", "quota", "resource_exhausted", "too many requests", "high demand", "unavailable", "spikes in demand"]
        return any(k in err_str for k in keywords)

    async def ainvoke(self, prompt: Any) -> Any:
        primary_provider = settings.LLM_PROVIDER.lower() if settings.LLM_PROVIDER else "gemini"
        secondary_provider = "groq" if primary_provider == "gemini" else "gemini"

        providers_to_try = [primary_provider, secondary_provider]

        last_exception = None

        for provider in providers_to_try:
            keys = self.gemini_keys if provider == "gemini" else self.groq_keys
            attempts = len(keys)

            for _ in range(max(1, attempts)):
                key = self._get_active_key(provider)
                if not key:
                    break

                try:
                    client = self._create_client(provider, key)
                    response = await client.ainvoke(prompt)
                    return response
                except Exception as e:
                    last_exception = e
                    if self._is_rate_limit_error(e):
                        self._mark_cooldown(key)
                    else:
                        logger.error(f"Error executing prompt with provider {provider}: {e}")
                        break

        if last_exception:
            raise last_exception
        raise RuntimeError("No active API keys available across all configured providers")
