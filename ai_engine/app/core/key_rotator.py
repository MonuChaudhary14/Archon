import time
import json
import socket
import asyncio
import logging
import requests
import urllib3.util.connection as urllib3_cn
from types import SimpleNamespace
from typing import List, Dict, Any, Optional
from langchain_groq import ChatGroq
from app.core.config import settings

def _allowed_gai_family():
    return socket.AF_INET

urllib3_cn.allowed_gai_family = _allowed_gai_family

logger = logging.getLogger(__name__)

class GeminiClient:
    def __init__(self, api_key: str, model: Optional[str] = None):
        self.api_key = api_key
        preferred_model = model or settings.GEMINI_MODEL or "gemini-2.5-flash"
        self.fallback_models = [
            preferred_model,
            "gemini-2.5-flash",
            "gemini-2.0-flash",
            "gemini-1.5-flash",
            "gemini-2.0-flash-lite",
        ]
        seen = set()
        self.models = [m for m in self.fallback_models if not (m in seen or seen.add(m))]

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
        last_error = None
        for m in self.models:
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{m}:generateContent?key={self.api_key}"
            try:
                res = await loop.run_in_executor(
                    None,
                    lambda u=url: requests.post(u, json=payload, timeout=20)
                )
                if res.status_code == 200:
                    data = res.json()
                    text = data["candidates"][0]["content"]["parts"][0]["text"]
                    return SimpleNamespace(content=text)
                elif res.status_code in (400, 401, 403, 404, 429, 500, 503):
                    last_error = RuntimeError(f"Gemini model {m} returned {res.status_code}: {res.text}")
                    continue
                else:
                    raise RuntimeError(f"Gemini API HTTP {res.status_code}: {res.text}")
            except Exception as e:
                last_error = e
                continue
        if last_error:
            raise last_error
        raise RuntimeError("All Gemini models exhausted for this key")

    async def astream(self, prompt: Any):
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
        last_error = None
        for m in self.models:
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{m}:streamGenerateContent?alt=sse&key={self.api_key}"
            def _stream_request(u=url):
                return requests.post(u, json=payload, stream=True, timeout=8)

            try:
                res = await loop.run_in_executor(None, _stream_request)
                if res.status_code == 200:
                    yielded_any = False
                    q = asyncio.Queue()
                    sentinel = object()

                    def _reader():
                        try:
                            for line in res.iter_lines():
                                if line:
                                    loop.call_soon_threadsafe(q.put_nowait, line)
                        except Exception as read_err:
                            loop.call_soon_threadsafe(q.put_nowait, read_err)
                        finally:
                            loop.call_soon_threadsafe(q.put_nowait, sentinel)

                    loop.run_in_executor(None, _reader)

                    while True:
                        item = await q.get()
                        if item is sentinel:
                            break
                        if isinstance(item, Exception):
                            break
                        decoded = item.decode("utf-8").strip()
                        if decoded.startswith("data: "):
                            data_str = decoded[6:].strip()
                            try:
                                chunk_json = json.loads(data_str)
                                candidates = chunk_json.get("candidates", [])
                                if candidates:
                                    parts = candidates[0].get("content", {}).get("parts", [])
                                    for part in parts:
                                        delta_text = part.get("text", "")
                                        if delta_text:
                                            yielded_any = True
                                            yield delta_text
                            except Exception:
                                pass

                    if yielded_any:
                        return
                elif res.status_code in (400, 401, 403, 404, 429, 500, 503):
                    last_error = RuntimeError(f"Gemini model {m} stream returned {res.status_code}: {res.text}")
                    continue
                else:
                    raise RuntimeError(f"Gemini API HTTP {res.status_code}: {res.text}")
            except Exception as e:
                last_error = e
                continue
        if last_error:
            raise last_error
        raise RuntimeError("All Gemini models exhausted for this key during stream")

class NvidiaClient:
    def __init__(self, api_key: str, model: str = "meta/llama-3.2-11b-vision-instruct", base_url: str = "https://integrate.api.nvidia.com/v1"):
        self.api_key = api_key
        self.model = model
        self.url = f"{base_url.rstrip('/')}/chat/completions"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
            "Accept": "application/json"
        }

    async def ainvoke(self, prompt: Any) -> Any:
        prompt_text = prompt if isinstance(prompt, str) else str(prompt)
        payload = {
            "model": self.model,
            "messages": [{"role": "user", "content": prompt_text}],
            "temperature": 0.7,
            "max_tokens": 2048,
        }
        loop = asyncio.get_running_loop()
        res = await loop.run_in_executor(
            None,
            lambda: requests.post(self.url, headers=self.headers, json=payload, timeout=45)
        )
        if res.status_code == 200:
            data = res.json()
            try:
                text = data["choices"][0]["message"]["content"]
                return SimpleNamespace(content=text)
            except (KeyError, IndexError):
                raise ValueError(f"Unexpected NVIDIA API response structure: {data}")
        else:
            raise RuntimeError(f"NVIDIA API HTTP {res.status_code}: {res.text}")

    async def astream(self, prompt: Any):
        prompt_text = prompt if isinstance(prompt, str) else str(prompt)
        payload = {
            "model": self.model,
            "messages": [{"role": "user", "content": prompt_text}],
            "temperature": 0.7,
            "max_tokens": 2048,
            "stream": True,
        }
        loop = asyncio.get_running_loop()
        def _stream_request():
            headers = {**self.headers, "Accept": "text/event-stream"}
            return requests.post(self.url, headers=headers, json=payload, stream=True, timeout=45)

        res = await loop.run_in_executor(None, _stream_request)
        if res.status_code != 200:
            raise RuntimeError(f"NVIDIA API HTTP {res.status_code}: {res.text}")

        q = asyncio.Queue()
        sentinel = object()

        def _reader():
            try:
                for line in res.iter_lines():
                    if line:
                        loop.call_soon_threadsafe(q.put_nowait, line)
            except Exception as read_err:
                loop.call_soon_threadsafe(q.put_nowait, read_err)
            finally:
                loop.call_soon_threadsafe(q.put_nowait, sentinel)

        loop.run_in_executor(None, _reader)

        while True:
            item = await q.get()
            if item is sentinel:
                break
            if isinstance(item, Exception):
                break
            decoded = item.decode("utf-8").strip()
            if decoded.startswith("data: "):
                data_str = decoded[6:].strip()
                if data_str == "[DONE]":
                    break
                try:
                    chunk_json = json.loads(data_str)
                    delta = chunk_json["choices"][0]["delta"].get("content", "")
                    if delta:
                        yield delta
                except Exception:
                    pass

class LLMKeyRotator:
    def __init__(self):
        self.cooldown_duration = 30.0
        self.cooldowns: Dict[str, float] = {}

        self.gemini_keys: List[str] = self._parse_keys(settings.GEMINI_API_KEYS, settings.GEMINI_API_KEY, provider="gemini")
        self.groq_keys: List[str] = self._parse_keys(settings.GROQ_API_KEYS, settings.GROQ_API_KEY, provider="groq")
        self.nvidia_keys: List[str] = self._parse_keys(settings.NVIDIA_API_KEYS, settings.NVIDIA_API_KEY, provider="nvidia")

        self.gemini_index = 0
        self.groq_index = 0
        self.nvidia_index = 0

    def _parse_keys(self, keys_str: str, single_key: str, provider: str = "") -> List[str]:
        keys = []
        raw_keys = []
        if keys_str:
            raw_keys.extend([k.strip() for k in keys_str.split(",") if k.strip()])
        if single_key and single_key.strip():
            raw_keys.append(single_key.strip())

        for k_clean in raw_keys:
            if k_clean not in keys:
                keys.append(k_clean)
        return keys

    def _is_key_active(self, key: str) -> bool:
        cooldown_until = self.cooldowns.get(key, 0.0)
        return time.time() >= cooldown_until

    def _mark_cooldown(self, key: str):
        self.cooldowns[key] = time.time() + self.cooldown_duration
        logger.warning(f"Key {key[:6]}... placed on cooldown for {self.cooldown_duration}s")

    def _create_client(self, provider: str, api_key: str):
        if provider == "gemini":
            model_name = settings.GEMINI_MODEL or "gemini-2.5-flash"
            return GeminiClient(api_key=api_key, model=model_name)
        elif provider == "groq":
            model_name = settings.GROQ_MODEL or "llama-3.3-70b-versatile"
            return ChatGroq(
                model=model_name,
                groq_api_key=api_key,
                temperature=0.7
            )
        elif provider == "nvidia":
            model_name = settings.NVIDIA_MODEL or "meta/llama-3.2-11b-vision-instruct"
            base_url = settings.NVIDIA_BASE_URL or "https://integrate.api.nvidia.com/v1"
            return NvidiaClient(api_key=api_key, model=model_name, base_url=base_url)
        else:
            raise ValueError(f"Unsupported LLM provider: {provider}")

    def _get_active_key(self, provider: str) -> Optional[str]:
        if provider == "gemini":
            keys = self.gemini_keys
            start_index = self.gemini_index
        elif provider == "nvidia":
            keys = self.nvidia_keys
            start_index = self.nvidia_index
        else:
            keys = self.groq_keys
            start_index = self.groq_index

        if not keys:
            return None

        n = len(keys)
        for i in range(n):
            idx = (start_index + i) % n
            key = keys[idx]
            if self._is_key_active(key):
                if provider == "gemini":
                    self.gemini_index = (idx + 1) % n
                elif provider == "nvidia":
                    self.nvidia_index = (idx + 1) % n
                else:
                    self.groq_index = (idx + 1) % n
                return key

        # If all keys are in cooldown, pick the one with the lowest cooldown expiry
        best_key = min(keys, key=lambda k: self.cooldowns.get(k, 0.0))
        return best_key

    def _is_rate_limit_error(self, err: Exception) -> bool:
        err_str = str(err).lower()
        keywords = ["429", "503", "rate limit", "quota", "resource_exhausted", "too many requests", "high demand", "unavailable", "spikes in demand"]
        return any(k in err_str for k in keywords)

    def _get_providers_order(self) -> List[str]:
        primary = settings.LLM_PROVIDER.lower() if settings.LLM_PROVIDER else "groq"
        all_providers = ["groq", "gemini", "nvidia"]
        if primary in all_providers:
            ordered = [primary] + [p for p in all_providers if p != primary]
        else:
            ordered = all_providers
        return ordered

    async def ainvoke(self, prompt: Any) -> Any:
        providers_to_try = self._get_providers_order()
        last_exception = None

        for provider in providers_to_try:
            if provider == "gemini":
                keys = self.gemini_keys
            elif provider == "nvidia":
                keys = self.nvidia_keys
            else:
                keys = self.groq_keys

            if not keys:
                continue

            attempts = len(keys)
            for _ in range(attempts):
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
                    logger.warning(f"Failed prompt with provider {provider} (key {key[:6]}...): {e}. Trying next.")
                    continue

        if last_exception:
            raise last_exception
        raise RuntimeError("No active API keys available across all configured providers")

    async def astream(self, prompt: Any):
        providers_to_try = self._get_providers_order()
        last_exception = None

        for provider in providers_to_try:
            if provider == "gemini":
                keys = self.gemini_keys
            elif provider == "nvidia":
                keys = self.nvidia_keys
            else:
                keys = self.groq_keys

            if not keys:
                continue

            attempts = len(keys)
            for _ in range(attempts):
                key = self._get_active_key(provider)
                if not key:
                    break

                try:
                    client = self._create_client(provider, key)
                    if hasattr(client, "astream"):
                        yielded_any = False
                        async for chunk in client.astream(prompt):
                            content = chunk.content if hasattr(chunk, "content") else str(chunk)
                            if content:
                                yielded_any = True
                                yield content
                        if yielded_any:
                            return
                    else:
                        response = await client.ainvoke(prompt)
                        yield response.content
                        return
                except Exception as e:
                    last_exception = e
                    if self._is_rate_limit_error(e):
                        self._mark_cooldown(key)
                    logger.warning(f"Failed streaming with provider {provider} (key {key[:6]}...): {e}. Trying next.")
                    continue

        if last_exception:
            raise last_exception
        raise RuntimeError("No active API keys available across all configured providers")
