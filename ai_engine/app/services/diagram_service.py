import json
from typing import Any
from app.core.config import settings

class DiagramService:
    def __init__(self, llm: Any):
        self.llm = llm

    def parse_diagram_to_text(self, diagram_ctx: dict) -> str:
        if not diagram_ctx or (not diagram_ctx.get("nodes") and not diagram_ctx.get("edges")):
            return "The whiteboard canvas is currently empty."

        nodes = diagram_ctx.get("nodes", [])
        edges = diagram_ctx.get("edges", [])

        node_map = {n["id"]: f"{n['label']} ({n['type']})" for n in nodes}

        markdown_desc = "Whiteboard Components:\n"
        for n in nodes:
            markdown_desc += f"- {n['label']} (Type: {n['type']})\n"

        if edges:
            markdown_desc += "\nComponent Connections (Data Flow/Dependencies):\n"
            for e in edges:
                src_label = node_map.get(e["source"], e["source"])
                tgt_label = node_map.get(e["target"], e["target"])
                edge_type = e.get("type", "connects to")
                markdown_desc += f"- {src_label} --[{edge_type}]--> {tgt_label}\n"

        return markdown_desc

    async def generate_diagram_critique(self, session_id: str, parsed_diagram: str, history_messages: list) -> dict:
        context = "\n".join([f"{msg.type}: {msg.content}" for msg in history_messages[-6:]])

        system_prompt = (
            "You are a Senior System Design Interviewer.\n"
            "The candidate has just modified the architectural whiteboard diagram.\n"
            "Your task is to analyze the diagram structure and contextually decide if you need to comment or ask a follow-up question.\n\n"
            f"Here is the Candidate's Current Whiteboard Diagram:\n"
            f"\"\"\"\n{parsed_diagram}\n\"\"\"\n\n"
            f"Here is the recent conversation history:\n"
            f"\"\"\"\n{context}\n\"\"\"\n\n"
            "Guidelines:\n"
            "1. Only respond if there is a significant change (e.g. added a database, queue, load balancer, cache) that has issues, lacks detail, or warrants immediate discussion.\n"
            "2. If the diagram is incomplete, or the changes do not warrant an immediate follow-up, remain silent.\n"
            "3. Keep any critique or question very brief (1-2 sentences), keeping it professional and demanding.\n\n"
            "You MUST output your response as a valid JSON object matching this structure:\n"
            "{\n"
            '  "should_respond": true | false,\n'
            '  "response": "Your question or feedback here (only if should_respond is true)"\n'
            "}\n"
            "Do not wrap the JSON output in markdown formatting or code blocks."
        )

        try:
            response = await self.llm.ainvoke(system_prompt)
            content = response.content.strip()
            if content.startswith("```"):
                import re
                match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
                if match:
                    content = match.group(1).strip()
            return json.loads(content)
        except Exception as e:
            print(f"Error generating diagram critique: {e}")
            return {"should_respond": False, "response": ""}
