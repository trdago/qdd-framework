"""
Unit Tests for Autonomous Tool Loop and Robustness Helpers
Strictly follows Zero-Else and Early Return.
"""

import unittest
import asyncio
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from python.ping_pong_client import extract_json_block, get_agentapi_path, AntigravityPingPongClient
from python.tool_bridge import QDDToolBridge

class TestAutonomousLoopAndHelpers(unittest.TestCase):
    def test_extract_json_block_delimited(self):
        raw = """Pensamiento preliminar...
[VERDICT_START]
{
  "status": "APPROVED",
  "score": 100,
  "zero_else": true
}
[VERDICT_END]
Comentarios finales."""
        parsed = extract_json_block(raw)
        self.assertEqual(parsed.get("status"), "APPROVED")
        self.assertEqual(parsed.get("score"), 100)
        self.assertTrue(parsed.get("zero_else"))

    def test_extract_json_block_markdown(self):
        raw = """Aquí está el resultado:
```json
{
  "valid": true,
  "reason": "Todo en orden"
}
```"""
        parsed = extract_json_block(raw)
        self.assertTrue(parsed.get("valid"))
        self.assertEqual(parsed.get("reason"), "Todo en orden")

    def test_extract_json_block_raw_or_empty(self):
        empty_res = extract_json_block("")
        self.assertEqual(empty_res, {})

        invalid_res = extract_json_block("Texto no JSON")
        self.assertEqual(invalid_res.get("raw"), "Texto no JSON")

    def test_get_agentapi_path_resolution(self):
        path = get_agentapi_path()
        # Path may be str or None depending on OS environment, but should not crash
        if path is not None:
            self.assertTrue(isinstance(path, str))

    def test_autonomous_session_with_tool_bridge(self):
        async def run_autonomous_test():
            bridge = QDDToolBridge()
            bridge.register_tool(
                name="health_check",
                description="Returns system status",
                parameters_schema={"type": "object"},
                handler=lambda: {"alive": True, "code": 200}
            )

            client = AntigravityPingPongClient(tool_bridge=bridge)
            async with client.session("test_autonomous_sess") as session:
                pong = await session.ping(
                    prompt="Verifica el estado del sistema",
                    autonomous_tools=True
                )

                self.assertTrue(pong.is_success())
                self.assertEqual(pong.session_id, "test_autonomous_sess")
                self.assertEqual(pong.turn_id, 1)

        asyncio.run(run_autonomous_test())

if __name__ == "__main__":
    unittest.main()
