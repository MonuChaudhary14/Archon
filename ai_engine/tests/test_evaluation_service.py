import unittest
from unittest.mock import AsyncMock, MagicMock
from app.services.evaluation_service import EvaluationService

class TestEvaluationService(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.mock_llm = MagicMock()
        self.mock_repo = MagicMock()
        self.service = EvaluationService(self.mock_llm, self.mock_repo)

    async def test_evaluate_session_success(self):
        self.mock_repo.get_interview_context.return_value = {
            "title": "Design TikTok",
            "difficulty": "Senior",
            "expected_topics": ["CDN", "NoSQL"]
        }
        
        mock_msg_hist = MagicMock()
        mock_msg_hist.messages = [
            MagicMock(type="human", content="How to scale it?"),
            MagicMock(type="ai", content="We can use CDN.")
        ]
        self.service.get_message_history = MagicMock(return_value=mock_msg_hist)
        
        self.mock_repo.get_diagram_context.return_value = {
            "nodes": [{"id": "1", "type": "cdn", "label": "CDN"}],
            "edges": []
        }

        mock_llm_response = MagicMock()
        mock_llm_response.content = """
        {
            "overall_score": 85,
            "requirements_score": 4,
            "requirements_feedback": "Good job.",
            "estimation_score": 4,
            "estimation_feedback": "Accurate.",
            "high_level_score": 4,
            "high_level_feedback": "Excellent.",
            "deep_dive_score": 4,
            "deep_dive_feedback": "Detailed.",
            "summary_strengths": "CDNs used.",
            "summary_weaknesses": "None.",
            "personalized_learning_path": []
        }
        """
        self.mock_llm.ainvoke = AsyncMock(return_value=mock_llm_response)

        await self.service.evaluate_session("test-session-123")

        self.mock_repo.get_interview_context.assert_called_once_with("test-session-123")
        self.mock_repo.get_diagram_context.assert_called_once_with("test-session-123")
        self.mock_llm.ainvoke.assert_called_once()
        self.mock_repo.save_evaluation_report.assert_called_once_with(
            "test-session-123",
            85,
            {
                "overall_score": 85,
                "requirements_score": 4,
                "requirements_feedback": "Good job.",
                "estimation_score": 4,
                "estimation_feedback": "Accurate.",
                "high_level_score": 4,
                "high_level_feedback": "Excellent.",
                "deep_dive_score": 4,
                "deep_dive_feedback": "Detailed.",
                "summary_strengths": "CDNs used.",
                "summary_weaknesses": "None.",
                "personalized_learning_path": []
            }
        )

    async def test_evaluate_session_failure(self):
        self.mock_repo.get_interview_context.return_value = None
        
        mock_msg_hist = MagicMock()
        mock_msg_hist.messages = [
            MagicMock(type="human", content="How to scale it?")
        ]
        self.service.get_message_history = MagicMock(return_value=mock_msg_hist)
        self.mock_repo.get_diagram_context.return_value = None

        self.mock_llm.ainvoke = AsyncMock(side_effect=Exception("LLM Timeout"))

        await self.service.evaluate_session("test-session-456")

        self.mock_repo.save_evaluation_error.assert_called_once_with("test-session-456", "LLM Timeout")

if __name__ == "__main__":
    unittest.main()
