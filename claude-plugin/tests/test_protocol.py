import io
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from contextkit import protocol  # noqa: E402
from contextkit.mirror import mirror_tool, inject_room_id, tool_is_room_scoped  # noqa: E402


class ProtocolTests(unittest.TestCase):
    def test_read_message_round_trip(self):
        stream = io.StringIO('{"jsonrpc": "2.0", "id": 1, "method": "ping"}\n')
        msg = protocol.read_message(stream)
        self.assertEqual(msg["method"], "ping")
        self.assertIsNone(protocol.read_message(stream))  # EOF

    def test_read_message_tolerates_blank_lines(self):
        stream = io.StringIO('\n\n{"jsonrpc": "2.0", "id": 1, "method": "ping"}\n')
        self.assertEqual(protocol.read_message(stream)["method"], "ping")

    def test_read_message_rejects_bad_json(self):
        stream = io.StringIO("hello\n")
        with self.assertRaises(ValueError):
            protocol.read_message(stream)

    def test_write_message_is_single_line(self):
        out = io.StringIO()
        protocol.write_message(out, {"jsonrpc": "2.0", "id": 1, "result": {"ok": True}})
        line = out.getvalue()
        self.assertEqual(line.count("\n"), 1)
        self.assertNotIn(" ", line.replace("\n", ""))  # compact separators

    def test_parse_sse_single(self):
        body = (
            'event: message\n'
            'data: {"jsonrpc":"2.0","id":1,"result":{"x":1}}\n'
            '\n'
        )
        objects = protocol.parse_sse(body)
        self.assertEqual(len(objects), 1)
        self.assertEqual(objects[0]["result"]["x"], 1)

    def test_parse_sse_multiline_data(self):
        # SSE spec: multiple data: lines in one event join with a newline.
        body = (
            'event: message\n'
            'data: {"jsonrpc":"2.0","id":2,\n'
            'data:  "result":{"y":2}}\n'
            '\n'
        )
        parsed = protocol.parse_sse(body)[0]
        self.assertEqual(parsed["id"], 2)

    def test_parse_sse_ignores_comments_and_retry(self):
        body = (
            ': keep-alive\n'
            'retry: 100\n'
            'data: {"method":"logs/level"}\n'
            '\n'
            'data: {"jsonrpc":"2.0","id":3,"result":{}}\n'
            '\n'
        )
        objects = protocol.parse_sse(body)
        self.assertEqual(len(objects), 2)


ROOM_TOOL = {
    "name": "get_context_package",
    "inputSchema": {
        "type": "object",
        "properties": {"room_id": {"type": "string"}, "query": {"type": "string"}},
        "required": ["room_id"],
    },
}
NON_ROOM_TOOL = {
    "name": "list_rooms",
    "inputSchema": {
        "type": "object",
        "properties": {"limit": {"type": "integer"}},
    },
}


class MirrorTests(unittest.TestCase):
    def test_room_tool_detected(self):
        self.assertTrue(tool_is_room_scoped("get_context_package", [ROOM_TOOL]))
        self.assertFalse(tool_is_room_scoped("list_rooms", [NON_ROOM_TOOL]))

    def test_mirror_drops_room_id_from_required(self):
        cloned = mirror_tool(ROOM_TOOL)
        self.assertNotIn("room_id", cloned["inputSchema"]["required"])
        # the property itself is preserved so explicit overrides still work
        self.assertIn("room_id", cloned["inputSchema"]["properties"])

    def test_mirror_is_copy_not_mutation(self):
        import copy
        original = copy.deepcopy(ROOM_TOOL)
        mirror_tool(ROOM_TOOL)
        self.assertIn("room_id", ROOM_TOOL["inputSchema"]["required"])

    def test_inject_room_id(self):
        args = inject_room_id("get_context_package", {"query": "hi"}, [ROOM_TOOL], "room_1")
        self.assertEqual(args, {"query": "hi", "room_id": "room_1"})

    def test_inject_respects_explicit_room(self):
        args = inject_room_id(
            "get_context_package", {"query": "hi", "room_id": "room_2"}, [ROOM_TOOL], "room_1"
        )
        self.assertEqual(args["room_id"], "room_2")

    def test_inject_never_touches_non_room_tools(self):
        args = inject_room_id("list_rooms", {"limit": 5}, [NON_ROOM_TOOL], "room_1")
        self.assertEqual(args, {"limit": 5})

    def test_inject_skips_when_no_bound_room(self):
        args = inject_room_id("get_context_package", {"query": "hi"}, [ROOM_TOOL], None)
        self.assertNotIn("room_id", args)


if __name__ == "__main__":
    unittest.main()