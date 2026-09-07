import unittest
from unittest.mock import AsyncMock, MagicMock, patch
from app.workflows.activities import EvaluationActivities
from app.workflows.contracts import EvaluationWorkflowInput

class TestTemporalEvaluationActivities(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.mock_repo = MagicMock()
        self.mock_llm = MagicMock()
        self.mock_knowledge = MagicMock()
        self.activities = EvaluationActivities(
            repo=self.mock_repo,
            llm=self.mock_llm,
            knowledge=self.mock_knowledge
        )

    async def test_fetch_interview_context(self):
        self.mock_repo.get_interview_context.return_value = {
            "title": "Design Uber",
            "difficulty": "Senior",
            "expected_topics": ["Geohash", "WebSockets"]
        }
        self.mock_repo.get_diagram_context.return_value = {
            "nodes": [{"id": "n1", "label": "Driver Service"}],
            "edges": []
        }

        with patch("app.workflows.activities.get_message_history") as mock_hist:
            mock_hist_obj = MagicMock()
            mock_hist_obj.messages = [
                MagicMock(type="human", content="We will use QuadTrees and Geohash."),
                MagicMock(type="ai", content="How do drivers update locations?")
            ]
            mock_hist.return_value = mock_hist_obj

            res = await self.activities.fetch_interview_context("session-uber-123")
            self.assertEqual(res["session_id"], "session-uber-123")
            self.assertEqual(res["title"], "Design Uber")
            self.assertEqual(res["human_message_count"], 1)
            self.assertTrue(res["has_diagram"])

    async def test_search_knowledge_rubrics(self):
        self.mock_knowledge.search.return_value = [
            {"content": "Geohashing divides spatial area into buckets.", "score": 0.92}
        ]
        results = await self.activities.search_knowledge_rubrics(["Geohash"])
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["topic"], "Geohash")

    async def test_execute_llm_evaluation_zero_participation(self):
        payload = {
            "context": {
                "title": "Design Twitter",
                "difficulty": "Senior",
                "expected_topics": ["Fanout", "Redis"],
                "transcript": "",
                "human_message_count": 0,
                "diagram_nodes": [],
                "diagram_edges": [],
                "has_diagram": False
            },
            "knowledge_rubrics": []
        }

        result = await self.activities.execute_llm_evaluation(payload)
        self.assertEqual(result["overall_score"], 0)
        self.assertEqual(len(result["rubrics"]), 4)

    async def test_execute_llm_evaluation_success(self):
        payload = {
            "context": {
                "title": "Design Netflix",
                "difficulty": "Senior",
                "expected_topics": ["CDN", "Microservices"],
                "transcript": "human: We use CDN for video distribution and Cassandra for metadata.\nai: Good choice.",
                "human_message_count": 1,
                "diagram_nodes": [{"id": "1", "label": "CDN"}],
                "diagram_edges": [],
                "has_diagram": True
            },
            "knowledge_rubrics": [
                {"topic": "CDN", "content": "Edge caching reduces video latency.", "score": 0.95}
            ]
        }

        mock_llm_response = MagicMock()
        mock_llm_response.content = """
        {
            "overall_score": 90,
            "interviewer_summary": "Strong architectural design with appropriate CDN caching.",
            "rubrics": [
                {
                    "name": "Requirements & Scope",
                    "score": 90,
                    "weight": 20,
                    "summary": "Clear requirements.",
                    "feedback_points": ["Addressed scale."]
                },
                {
                    "name": "Capacity Estimation",
                    "score": 85,
                    "weight": 20,
                    "summary": "Good estimation.",
                    "feedback_points": ["Calculated bandwidth."]
                },
                {
                    "name": "High-Level Architecture",
                    "score": 92,
                    "weight": 30,
                    "summary": "Excellent design.",
                    "feedback_points": ["Decoupled services."]
                },
                {
                    "name": "Scalability & Deep Dive",
                    "score": 90,
                    "weight": 30,
                    "summary": "Thorough scaling.",
                    "feedback_points": ["Handled failover."]
                }
            ],
            "strengths": ["Clear CDN caching approach"],
            "weaknesses": ["Could elaborate on DRM"],
            "recommendations": ["Review multi-CDN strategies"],
            "diagram_components": ["CDN"]
        }
        """
        self.mock_llm.ainvoke = AsyncMock(return_value=mock_llm_response)

        result = await self.activities.execute_llm_evaluation(payload)
        self.assertEqual(result["overall_score"], 90)
        self.assertEqual(len(result["rubrics"]), 4)

    async def test_persist_evaluation_report(self):
        payload = {
            "session_id": "session-100",
            "score": 88,
            "feedback": {"overall_score": 88}
        }
        success = await self.activities.persist_evaluation_report(payload)
        self.assertTrue(success)
        self.mock_repo.save_evaluation_report.assert_called_once_with("session-100", 88, {"overall_score": 88})

    async def test_compensate_evaluation_failure(self):
        payload = {
            "session_id": "session-100",
            "error_message": "LLM rate limit exceeded"
        }
        success = await self.activities.compensate_evaluation_failure(payload)
        self.assertTrue(success)
        self.mock_repo.save_evaluation_error.assert_called_once_with("session-100", "LLM rate limit exceeded")
