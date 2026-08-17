"""Integration test against the REAL local ContextKit MCP server.

Gated behind ``CONTEXTKIT_IT=1`` and a valid key in ``CONTEXTKIT_IT_KEY``.
Skip by default so the suite stays hermetic.

    cd backend && MCP_PORT=8099 ./mcp_server --transport http
    CONTEXTKIT_IT=1 CONTEXTKIT_IT_KEY=ck_live_dev_test_key_123 \
        python3 -m unittest tests/test_integration_local.py -v
"""

import os
import sys
import tempfile
import unittest

_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
_PLUGIN_DIR = os.path.dirname(_THIS_DIR)
for _p in (_PLUGIN_DIR, _THIS_DIR):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from test_adapter import AdapterProcess  # noqa: E402

KEY = os.environ.get("CONTEXTKIT_IT_KEY")
URL = os.environ.get("CONTEXTKIT_IT_URL", "http://localhost:8099/mcp")


def _text(result):
    for item in result.get("content") or []:
        if item.get("type") == "text":
            return item["text"]
    return ""


@unittest.skipUnless(
    os.environ.get("CONTEXTKIT_IT") == "1" and KEY,
    "integration requires CONTEXTKIT_IT=1 and CONTEXTKIT_IT_KEY",
)
class RealServerIntegrationTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        env = dict(os.environ)
        env.update({
            "CONTEXTKIT_STATE_DIR": self._tmp.name,
            "CONTEXTKIT_KEY_STORE": "file",
            "CONTEXTKIT_REMOTE_URL": URL,
            "CONTEXTKIT_LOG_LEVEL": "ERROR",
            "CLAUDE_PROJECT_DIR": self._tmp.name,
        })
        self.adapter = AdapterProcess(env)
        self.addCleanup(self.adapter.close)

    def test_real_remote_tools_mirrored(self):
        result = self.adapter.call_tool("contextkit_set_key", {"api_key": KEY})
        import json
        payload = json.loads(_text(result))
        self.assertTrue(payload["stored"])

        tools = self.adapter.request("tools/list")["result"]["tools"]
        names = {t["name"] for t in tools}
        for expected in ("list_rooms", "get_room", "add_message", "get_context_package"):
            self.assertIn(expected, names)
        package = next(t for t in tools if t["name"] == "get_context_package")
        self.assertNotIn("room_id", package["inputSchema"].get("required", []))

        rooms_raw = _text(self.adapter.call_tool("list_rooms", {"limit": 5}))
        rooms = (json.loads(rooms_raw) or {}).get("rooms") or []
        if rooms:
            first_id = rooms[0]["id"]
            result = self.adapter.call_tool("contextkit_set_room", {"room_id": first_id})
            cfg = json.loads(_text(self.adapter.call_tool("contextkit_get_config", {})))
            self.assertTrue(cfg["configured"])
            # room injection against the real server
            package_raw = _text(
                self.adapter.call_tool("get_context_package", {"query": "status"})
            )
            self.assertIn("room_id", json.loads(package_raw))


if __name__ == "__main__":
    unittest.main()