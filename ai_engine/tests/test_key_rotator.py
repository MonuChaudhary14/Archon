import unittest
from unittest.mock import patch, MagicMock, AsyncMock
from types import SimpleNamespace
from app.core.key_rotator import LLMKeyRotator

class TestLLMKeyRotator(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        with patch("app.core.key_rotator.settings") as mock_settings:
            mock_settings.GEMINI_API_KEYS = "gemini_key_1,gemini_key_2"
            mock_settings.GEMINI_API_KEY = ""
            mock_settings.GROQ_API_KEYS = "groq_key_1"
            mock_settings.GROQ_API_KEY = ""
            mock_settings.LLM_PROVIDER = "gemini"
            mock_settings.GEMINI_MODEL = "gemini-flash-latest"
            mock_settings.GROQ_MODEL = "openai/gpt-oss-20b"
            self.rotator = LLMKeyRotator()

    def test_key_parsing(self):
        self.assertEqual(len(self.rotator.gemini_keys), 2)
        self.assertEqual(self.rotator.gemini_keys[0], "gemini_key_1")
        self.assertEqual(self.rotator.gemini_keys[1], "gemini_key_2")
        self.assertEqual(len(self.rotator.groq_keys), 1)
        self.assertEqual(self.rotator.groq_keys[0], "groq_key_1")

    def test_round_robin_selection(self):
        k1 = self.rotator._get_active_key("gemini")
        k2 = self.rotator._get_active_key("gemini")
        k3 = self.rotator._get_active_key("gemini")
        self.assertEqual(k1, "gemini_key_1")
        self.assertEqual(k2, "gemini_key_2")
        self.assertEqual(k3, "gemini_key_1")

    @patch("app.core.key_rotator.GeminiClient")
    async def test_ainvoke_success(self, mock_gemini_cls):
        mock_client = MagicMock()
        mock_client.ainvoke = AsyncMock(return_value=SimpleNamespace(content="Gemini response"))
        mock_gemini_cls.return_value = mock_client

        res = await self.rotator.ainvoke("Hello")
        self.assertEqual(res.content, "Gemini response")

    @patch("app.core.key_rotator.ChatGroq")
    @patch("app.core.key_rotator.GeminiClient")
    async def test_fallback_on_429(self, mock_gemini_cls, mock_groq_cls):
        mock_gemini_client = MagicMock()
        mock_gemini_client.ainvoke = AsyncMock(side_effect=Exception("429 Too Many Requests"))
        mock_gemini_cls.return_value = mock_gemini_client

        mock_groq_client = MagicMock()
        mock_groq_response = SimpleNamespace(content="Groq fallback response")
        mock_groq_client.ainvoke = AsyncMock(return_value=mock_groq_response)
        mock_groq_cls.return_value = mock_groq_client

        res = await self.rotator.ainvoke("Hello")
        self.assertEqual(res.content, "Groq fallback response")

if __name__ == "__main__":
    unittest.main()
