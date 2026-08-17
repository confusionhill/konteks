"""A minimal streamable-HTTP MCP server used to exercise the adapter offline.

Implements just enough of the protocol for the adapter to work against it:
``initialize`` (returning a session id header), ``tools/list``, ``tools/call``
for a fixed set of fake rooms/tools, and notification handling. Plain JSON
responses, matching what ``Accept: application/json`` should yield.
"""

from __future__ import annotations

import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

FAKE_ROOMS = {
    "room_a": {
        "id": "room_a",
        "project_id": "proj_fake",
        "external_user_id": "user-1",
        "description": "Customer Support",
        "status": "active",
    },
    "room_b": {
        "id": "room_b",
        "project_id": "proj_fake",
        "external_user_id": "user-2",
        "description": "Coding",
        "status": "active",
    },
}

FAKE_TOOLS = [
    {
        "name": "list_rooms",
        "description": "List rooms in the project.",
        "inputSchema": {
            "type": "object",
            "properties": {"limit": {"type": "integer", "default": 20}},
        },
    },
    {
        "name": "get_room",
        "description": "Fetch a room.",
        "inputSchema": {
            "type": "object",
            "properties": {"room_id": {"type": "string"}},
            "required": ["room_id"],
        },
    },
    {
        "name": "create_room",
        "description": "Create a room.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "description": {"type": "string"},
                "external_user_id": {"type": "string"},
            },
        },
    },
    {
        "name": "get_context_package",
        "description": "Assemble a context package for a room.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "room_id": {"type": "string"},
                "query": {"type": "string"},
            },
            "required": ["room_id"],
        },
    },
    {
        "name": "add_message",
        "description": "Append a message to a room.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "room_id": {"type": "string"},
                "role": {"type": "string"},
                "content": {"type": "string"},
            },
            "required": ["room_id", "role", "content"],
        },
    },
]


class _Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    server_version = "FakeContextKitMCP/0.1"

    def log_message(self, *args):  # silence request logging
        pass

    def do_POST(self):  # noqa: N802
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length).decode("utf-8", "replace")
        try:
            message = json.loads(raw)
        except json.JSONDecodeError:
            self._json_response(400, {"jsonrpc": "2.0", "error": {"code": -32700, "message": "parse error"}})
            return

        method = message.get("method")
        if not method:
            return self._json_response(400, {"jsonrpc": "2.0", "error": {"code": -32600, "message": "invalid request"}})

        if method in ("notifications/initialized", "notifications/cancelled"):
            self.send_response(202)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return

        req_id = message.get("id")
        if method == "initialize":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Mcp-Session-Id", "mcp-session-fake-1")
            body = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "protocolVersion": "2025-06-18",
                    "capabilities": {"tools": {}},
                    "serverInfo": {"name": "fake-remote", "version": "1.0"},
                },
            }
            self.end_headers()
            self.wfile.write(json.dumps(body).encode("utf-8"))
            return

        if method == "tools/list":
            result = {"tools": FAKE_TOOLS}
        elif method == "tools/call":
            result = self._call_tool(message.get("params") or {})
        else:
            return self._json_response(200, {
                "jsonrpc": "2.0", "id": req_id,
                "error": {"code": -32601, "message": f"method not found: {method}"},
            })

        return self._json_response(200, {"jsonrpc": "2.0", "id": req_id, "result": result})

    def _call_tool(self, params):
        name = params.get("name")
        args = params.get("arguments") or {}
        if name == "list_rooms":
            return wrap({
                "total": len(FAKE_ROOMS),
                "limit": args.get("limit", 20),
                "offset": 0,
                "rooms": list(FAKE_ROOMS.values()),
            })
        if name == "get_room":
            room = FAKE_ROOMS.get(args.get("room_id"))
            if room is None:
                return tool_error_result("room not found or unauthorized")
            return wrap({"room": room, "state": {}})
        if name == "create_room":
            return wrap({
                "id": "room_new",
                "project_id": "proj_fake",
                "description": args.get("description", ""),
                "external_user_id": args.get("external_user_id", ""),
                "status": "active",
            })
        if name == "get_context_package":
            room = FAKE_ROOMS.get(args.get("room_id"))
            if room is None:
                return tool_error_result("room not found or unauthorized")
            return wrap({
                "room_id": args.get("room_id"),
                "query": args.get("query", ""),
                "strategy_used": "auto",
                "summary": room["description"],
            })
        if name == "add_message":
            room = FAKE_ROOMS.get(args.get("room_id"))
            if room is None:
                return tool_error_result("room not found or unauthorized")
            return wrap({"message_id": "msg_1", "sequence": 1})
        return tool_error_result(f"unknown tool: {name}")

    def _json_response(self, status, body):
        data = json.dumps(body).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def wrap(obj):
    """Wrap a payload in the standard MCP tool-result content format."""
    return {"content": [{"type": "text", "text": json.dumps(obj, ensure_ascii=False)}]}


def tool_error_result(text):
    return {"isError": True, "content": [{"type": "text", "text": text}]}


class FakeRemoteServer:
    """Thin wrapper that runs the fake MCP server on an ephemeral port."""

    def __init__(self):
        self._server = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)

    @property
    def url(self) -> str:
        host, port = self._server.server_address[:2]
        return f"http://{host}:{port}/mcp"

    def start(self) -> "FakeRemoteServer":
        self._thread.start()
        return self

    def stop(self) -> None:
        self._server.shutdown()
        self._thread.join(timeout=5)
        self._server.server_close()