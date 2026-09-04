import unittest
from unittest.mock import AsyncMock, MagicMock
from app.services.evaluation_service import EvaluationService

class TestEvaluationService(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.mock_llm = MagicMock()
        self.mock_repo = MagicMock()
        self.service = EvaluationService(self.mock_llm, self.mock_repo)

    async def test_evaluate_session_zero_participation(self):
        self.mock_repo.get_interview_context.return_value = {
            "title": "Design TinyURL",
            "difficulty": "Senior",
            "expected_topics": ["Hashing", "Base62"]
        }
        mock_msg_hist = MagicMock()
        mock_msg_hist.messages = [
            MagicMock(type="ai", content="Welcome! Let's design TinyURL.")
        ]
        self.service.get_message_history = MagicMock(return_value=mock_msg_hist)
        self.mock_repo.get_diagram_context.return_value = {"nodes": [], "edges": []}

        await self.service.evaluate_session("empty-session-1")

        self.mock_llm.ainvoke.assert_not_called()
        self.mock_repo.save_evaluation_report.assert_called_once()
        args = self.mock_repo.save_evaluation_report.call_args[0]
        self.assertEqual(args[0], "empty-session-1")
        self.assertEqual(args[1], 0)
        self.assertEqual(args[2]["overall_score"], 0)
        self.assertEqual(len(args[2]["rubrics"]), 4)

    async def test_evaluate_session_success(self):
        self.mock_repo.get_interview_context.return_value = {
            "title": "Design TikTok",
            "difficulty": "Senior",
            "expected_topics": ["CDN", "NoSQL"]
        }
        
        mock_msg_hist = MagicMock()
        mock_msg_hist.messages = [
            MagicMock(type="human", content="How to scale it? We should use CDN and NoSQL."),
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
            "interviewer_summary": "Great coverage of CDNs and distributed storage.",
            "rubrics": [
                {
                    "name": "Requirements & Scope",
                    "score": 85,
                    "weight": 20,
                    "summary": "Clear requirements.",
                    "feedback_points": ["Good scoping."]
                },
                {
                    "name": "Capacity Estimation",
                    "score": 80,
                    "weight": 20,
                    "summary": "Estimated bandwidth.",
                    "feedback_points": ["Solid calculations."]
                },
                {
                    "name": "High-Level Architecture",
                    "score": 90,
                    "weight": 30,
                    "summary": "Clean CDN edge tier.",
                    "feedback_points": ["Utilized CDN node."]
                },
                {
                    "name": "Scalability & Deep Dive",
                    "score": 85,
                    "weight": 30,
                    "summary": "Discussed partition keys.",
                    "feedback_points": ["Detailed scaling."]
                }
            ],
            "strengths": ["CDNs used properly."],
            "weaknesses": ["Could explore live streaming transcoding."],
            "recommendations": ["Review HLS/DASH video delivery."],
            "diagram_components": ["CDN"]
        }
        """
        self.mock_llm.ainvoke = AsyncMock(return_value=mock_llm_response)

        await self.service.evaluate_session("test-session-123")

        self.mock_repo.get_interview_context.assert_called_once_with("test-session-123")
        self.mock_repo.get_diagram_context.assert_called_once_with("test-session-123")
        self.mock_llm.ainvoke.assert_called_once()
        self.mock_repo.save_evaluation_report.assert_called_once()
        args = self.mock_repo.save_evaluation_report.call_args[0]
        self.assertEqual(args[0], "test-session-123")
        self.assertEqual(args[1], 85)
        self.assertEqual(args[2]["interviewer_summary"], "Great coverage of CDNs and distributed storage.")

    async def test_evaluate_session_failure(self):
        self.mock_repo.get_interview_context.return_value = {
            "title": "Design Cache",
            "difficulty": "Senior",
            "expected_topics": ["LRU", "Redis"]
        }
        
        mock_msg_hist = MagicMock()
        mock_msg_hist.messages = [
            MagicMock(type="human", content="How to scale it? We should add Redis cache.")
        ]
        self.service.get_message_history = MagicMock(return_value=mock_msg_hist)
        self.mock_repo.get_diagram_context.return_value = None

        self.mock_llm.ainvoke = AsyncMock(side_effect=Exception("LLM Timeout"))

        await self.service.evaluate_session("test-session-456")

        self.mock_repo.save_evaluation_error.assert_called_once_with("test-session-456", "LLM Timeout")

if __name__ == "__main__":
    unittest.main()
