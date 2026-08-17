"""End-to-end tests for the adapter as a real stdio MCP process.

The adapter runs as a subprocess talking JSON-RPC over stdin/stdout against the
in-process fake remote MCP server. Covers the full onboarding lifecycle, tool
mirroring, room injection, not-configured gating, and room switching.
"""

import json
import os
import subprocess
import sys
import tempfile
import time
import unittest

_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
_PLUGIN_DIR = os.path.dirname(_THIS_DIR)
for _p in (_PLUGIN_DIR, _THIS_DIR):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from fake_remote_mcp import FakeRemoteServer  # noqa: E402


class AdapterProcess:
    def __init__(self, env):
        self._id = 0
        script = os.path.join(_PLUGIN_DIR, "contextkit_adapter.py")
        self.proc = subprocess.Popen(
            [sys.executable, script],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            env=env,
            text=True,
        )

    def request(self, method, params=None):
        self._id += 1
        req = {"jsonrpc": "2.0", "id": self._id, "method": method}
        if params is not None:
            req["params"] = params
        self.proc.stdin.write(json.dumps(req, separators=(",", ":")) + "\n")
        self.proc.stdin.flush()
        line = self.proc.stdout.readline()
        if not line:
            raise RuntimeError("adapter process exited unexpectedly")
        return json.loads(line)

    def call_tool(self, name, arguments):
        response = self.request("tools/call", {"name": name, "arguments": arguments})
        return response["result"]

    def close(self):
        try:
            self.proc.stdin.close()
            self.proc.wait(timeout=5)
        except (BrokenPipeError, subprocess.TimeoutExpired):
            self.proc.kill()
        finally:
            self.proc.stdout.close()


def _text(result):
    for item in result.get("content") or []:
        if item.get("type") == "text":
            return json.loads(item["text"])
    return {}


def _tool_names(result):
    ids = set()
    for tool in result.get("tools") or []:
        ids.add(tool["name"])
    return ids


class AdapterEndToEndTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fake = FakeRemoteServer().start()

    @classmethod
    def tearDownClass(cls):
        cls.fake.stop()

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        env = dict(os.environ)
        env.update({
            "CONTEXTKIT_STATE_DIR": self._tmp.name,
            "CONTEXTKIT_KEY_STORE": "file",
            "CONTEXTKIT_REMOTE_URL": self.fake.url,
            "CONTEXTKIT_LOG_LEVEL": "ERROR",
            "CLAUDE_PROJECT_DIR": self._tmp.name,
        })
        self.adapter = AdapterProcess(env)
        self.addCleanup(self.adapter.close)

    def test_initialize(self):
        response = self.adapter.request("initialize", {
            "protocolVersion": "2025-06-18",
            "capabilities": {},
            "clientInfo": {"name": "test", "version": "1"},
        })
        self.assertEqual(response.get("result", {}).get("serverInfo", {}).get("name"), "contextkit")

    def test_not_configured_before_setup(self):
        response = self.adapter.request("tools/list")
        names = _tool_names(response["result"])
        self.assertIn("contextkit_get_config", names)
        self.assertNotIn("get_context_package", names)  # remote tools hidden without a key

        result = self.adapter.call_tool("get_context_package", {"query": "hi"})
        self.assertTrue(result["isError"])
        self.assertIn("ContextKitNotConfigured", result["content"][0]["text"])

    def test_full_onboarding_lifecycle(self):
        # 1. store + validate the key
        result = self.adapter.call_tool("contextkit_set_key", {"api_key": "ck_fake_valid"})
        payload = _text(result)
        self.assertTrue(payload["stored"])
        self.assertEqual(payload["project_id"], "proj_fake")

        # 2. remote tools now visible, room_id not required
        names = _tool_names(self.adapter.request("tools/list")["result"])
        self.assertIn("get_context_package", names)
        tools = self.adapter.request("tools/list")["result"]["tools"]
        package = next(t for t in tools if t["name"] == "get_context_package")
        self.assertNotIn("room_id", package["inputSchema"]["required"])

        # 3. bind a room
        result = self.adapter.call_tool("contextkit_set_room", {"room_id": "room_a"})
        self.assertEqual(_text(result)["room_id"], "room_a")

        # 4. configured now
        cfg = _text(self.adapter.call_tool("contextkit_get_config", {}))
        self.assertTrue(cfg["configured"])
        self.assertEqual(cfg["room_id"], "room_a")

        # 5. room is injected automatically
        result = self.adapter.call_tool("get_context_package", {"query": "hello"})
        package_text = result["content"][0]["text"]
        self.assertIn("room_a", package_text)

        # 6. multi-arg tool injection
        result = self.adapter.call_tool("add_message", {"role": "user", "content": "hi"})
        self.assertIn("msg_1", result["content"][0]["text"])

        # 7. explicit room override is preserved and validated by the server
        result = self.adapter.call_tool(
            "get_context_package", {"query": "x", "room_id": "room_b"}
        )
        self.assertIn("room_b", result["content"][0]["text"])

    def test_room_switching_reports_previous(self):
        self.adapter.call_tool("contextkit_set_key", {"api_key": "ck_fake_valid"})
        self.adapter.call_tool("contextkit_set_room", {"room_id": "room_a"})
        result = self.adapter.call_tool("contextkit_set_room", {"room_id": "room_b"})
        payload = _text(result)
        self.assertEqual(payload["previous_room_id"], "room_a")
        cfg = _text(self.adapter.call_tool("contextkit_get_config", {}))
        self.assertEqual(cfg["room_id"], "room_b")

    def test_unset_room_gates_context_again(self):
        self.adapter.call_tool("contextkit_set_key", {"api_key": "ck_fake_valid"})
        self.adapter.call_tool("contextkit_set_room", {"room_id": "room_a"})
        self.adapter.call_tool("contextkit_unset_room", {})
        cfg = _text(self.adapter.call_tool("contextkit_get_config", {}))
        self.assertFalse(cfg["configured"])
        result = self.adapter.call_tool("get_context_package", {"query": "hi"})
        self.assertTrue(result["isError"])
        self.assertIn("ContextKitNotConfigured", result["content"][0]["text"])

    def test_clear_wipes_key_and_config(self):
        self.adapter.call_tool("contextkit_set_key", {"api_key": "ck_fake_valid"})
        self.adapter.call_tool("contextkit_set_room", {"room_id": "room_a"})
        result = self.adapter.call_tool("contextkit_clear", {})
        self.assertTrue(_text(result)["cleared"])
        cfg = _text(self.adapter.call_tool("contextkit_get_config", {}))
        self.assertFalse(cfg["has_api_key"])
        self.assertFalse(cfg["configured"])


if __name__ == "__main__":
    time.sleep(0)
    unittest.main()