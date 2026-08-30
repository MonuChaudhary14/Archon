import unittest
from unittest.mock import AsyncMock, MagicMock
from app.services.diagram_service import DiagramService

class TestDiagramService(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.mock_llm = MagicMock()
        self.service = DiagramService(self.mock_llm)

    def test_parse_diagram_to_text_empty(self):
        res = self.service.parse_diagram_to_text({})
        self.assertEqual(res, "The whiteboard canvas is currently empty.")

        res2 = self.service.parse_diagram_to_text({"nodes": [], "edges": []})
        self.assertEqual(res2, "The whiteboard canvas is currently empty.")

    def test_parse_diagram_to_text_with_data(self):
        diagram_ctx = {
            "nodes": [
                {"id": "n1", "label": "Client Gateway", "type": "gateway"},
                {"id": "n2", "label": "Auth DB", "type": "postgres"}
            ],
            "edges": [
                {"source": "n1", "target": "n2", "type": "connects to"}
            ]
        }
        res = self.service.parse_diagram_to_text(diagram_ctx)
        self.assertIn("Client Gateway (Type: gateway)", res)
        self.assertIn("Auth DB (Type: postgres)", res)
        self.assertIn("Client Gateway (gateway) --[connects to]--> Auth DB (postgres)", res)

    async def test_generate_diagram_critique_should_respond(self):
        mock_response = MagicMock()
        mock_response.content = '{"should_respond": true, "response": "Add a load balancer."}'
        self.mock_llm.ainvoke = AsyncMock(return_value=mock_response)

        mock_msg1 = MagicMock()
        mock_msg1.type = "human"
        mock_msg1.content = "hello"

        res = await self.service.generate_diagram_critique(
            "session-abc",
            "Whiteboard Components:\n- Client Gateway (Type: gateway)",
            [mock_msg1]
        )
        self.assertTrue(res.get("should_respond"))
        self.assertEqual(res.get("response"), "Add a load balancer.")

    async def test_generate_diagram_critique_should_silent(self):
        mock_response = MagicMock()
        mock_response.content = '{"should_respond": false, "response": ""}'
        self.mock_llm.ainvoke = AsyncMock(return_value=mock_response)

        res = await self.service.generate_diagram_critique(
            "session-abc",
            "The whiteboard canvas is currently empty.",
            []
        )
        self.assertFalse(res.get("should_respond"))

if __name__ == "__main__":
    unittest.main()
