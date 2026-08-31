"""
Unit Tests for QDD Antigravity Ping-Pong Protocol & Client
Strictly follows Zero-Else and Early Return.
"""

import unittest
import asyncio
import os
import sys

# Ensure parent directory is in sys.path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from python.protocol_types import PingRequest, PongResponse
from python.ping_pong_client import AntigravityPingPongClient
from python.tool_bridge import QDDToolBridge

class TestPingPongProtocol(unittest.TestCase):
    def test_ping_request_serialization(self):
        req = PingRequest(
            prompt="Test prompt",
            session_id="sess_123",
            turn_id=1,
            context={"rule": "zero-else"}
        )
        data = req.to_dict()

        self.assertEqual(data["protocol"], "qdd-ping-pong/v1")
        self.assertEqual(data["message_type"], "PING")
        self.assertEqual(data["session_id"], "sess_123")
        self.assertEqual(data["turn_id"], 1)
        self.assertEqual(data["prompt"], "Test prompt")
        self.assertEqual(data["context"]["rule"], "zero-else")

    def test_pong_response_validation(self):
        res = PongResponse(
            session_id="sess_123",
            turn_id=1,
            status="SUCCESS",
            content="Generated response",
            duration_ms=45.2
        )
        self.assertTrue(res.is_success())
        self.assertEqual(res.to_dict()["status"], "SUCCESS")

    def test_async_ping_pong_handshake(self):
        async def run_handshake():
            client = AntigravityPingPongClient(system_instructions="Test assistant")
            collected_tokens = []

            async with client.session("test_sess_abc") as session:
                pong = await session.ping(
                    prompt="Hello Antigravity",
                    on_token=lambda tok: collected_tokens.append(tok)
                )

            self.assertEqual(pong.session_id, "test_sess_abc")
            self.assertEqual(pong.turn_id, 1)
            self.assertTrue(pong.is_success())
            self.assertGreater(len(collected_tokens), 0)
            self.assertIn("QDD Ping-Pong Handshake Active", pong.content)

        asyncio.run(run_handshake())

    def test_tool_bridge_execution(self):
        async def run_tool_test():
            bridge = QDDToolBridge()
            bridge.register_tool(
                name="calculate_sum",
                description="Sums two integers",
                parameters_schema={"type": "object", "properties": {"a": {"type": "integer"}, "b": {"type": "integer"}}},
                handler=lambda a, b: a + b
            )

            tools = bridge.list_tools()
            self.assertEqual(len(tools), 1)
            self.assertEqual(tools[0]["name"], "calculate_sum")

            res = await bridge.execute_tool("calculate_sum", {"a": 10, "b": 25})
            self.assertEqual(res["status"], "SUCCESS")
            self.assertEqual(res["result"], 35)

            err_res = await bridge.execute_tool("non_existent_tool")
            self.assertEqual(err_res["status"], "ERROR")

        asyncio.run(run_tool_test())

if __name__ == "__main__":
    unittest.main()
