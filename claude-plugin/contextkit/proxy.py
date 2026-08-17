"""Minimal remote MCP client used to reach the ContextKit server.

Only what the adapter needs: ``initialize`` handshake (capturing the remote
``Mcp-Session-Id`` header), ``tools/list``, ``tools/call``, and unsolicited
notifications. The API key travels as an ``X-API-Key`` header on every request,
matching ``backend/cmd/mcp_server/main.go``.

Uses only the standard library (``urllib``); no third-party MCP SDK.
"""

from __future__ import annotations

import json
import urllib.error
import urllib.request

from .protocol import parse_sse

_MCP_PROTOCOL_VERSION = "2025-06-18"


class RemoteError(Exception):
    def __init__(self, message: str, code: int | None = None):
        super().__init__(message)
        self.code = code


class AuthError(RemoteError):
    """The remote rejected our API key (401/403 or a JSON-RPC auth error)."""


class RemoteUnavailable(RemoteError):
    """Network-level failure: could not reach the remote MCP endpoint."""


class RemoteClient:
    def __init__(self, url: str, api_key: str | None):
        self.url = url
        self.api_key = api_key or ""
        self.session_id: str | None = None
        self.initialized = False
        self._next_id = 0

    def reset(self) -> None:
        self.session_id = None
        self.initialized = False

    # ---- transport -------------------------------------------------------- #

    def _post(self, payload: dict, timeout: float = 60.0) -> dict | None:
        """POST one JSON-RPC payload and return the parsed response message.

        Returns ``None`` for notifications, which the server answers with an
        empty 202 body.
        """
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        if self.session_id:
            headers["Mcp-Session-Id"] = self.session_id

        body = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        request = urllib.request.Request(self.url, data=body, headers=headers, method="POST")

        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                content_type = response.headers.get("Content-Type", "")
                session = response.headers.get("Mcp-Session-Id")
                if session:
                    self.session_id = session
                raw = response.read().decode("utf-8", "replace")
        except urllib.error.HTTPError as exc:
            code = exc.code
            detail = exc.read().decode("utf-8", "replace")[:400]
            if code in (401, 403):
                raise AuthError(
                    "authentication failed: API key invalid, revoked, or expired"
                ) from exc
            raise RemoteError(f"remote MCP returned HTTP {code}: {detail}") from exc
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            raise RemoteUnavailable(
                f"cannot reach the remote MCP server at {self.url}: {exc}"
            ) from exc

        if not raw.strip():
            return None

        if "text/event-stream" in content_type:
            objects = parse_sse(raw)
            if not objects:
                raise RemoteError("remote MCP returned an empty event-stream response")
            return objects[-1]
        try:
            return json.loads(raw)
        except json.JSONDecodeError as exc:
            raise RemoteError(
                f"remote MCP returned an invalid JSON response: {exc}"
            ) from exc

    def _rpc(self, method: str, params: dict | None = None, timeout: float = 60.0):
        """Send a request and return its ``result`` (or raise RemoteError)."""
        self._next_id += 1
        payload: dict = {"jsonrpc": "2.0", "id": self._next_id, "method": method}
        if params is not None:
            payload["params"] = params
        message = self._post(payload, timeout=timeout)
        if message is None:
            raise RemoteError(f"empty response for {method}")
        error = message.get("error")
        if error:
            text = str(error.get("message", "remote MCP error"))
            code = error.get("code")
            if code in (-32001, -32002) or "auth" in text.lower():
                raise AuthError(f"authentication failed: {text}") from None
            raise RemoteError(text, code=code)
        return message.get("result")

    # ---- MCP protocol ------------------------------------------------------ #

    def initialize(self) -> dict:
        result = self._rpc(
            "initialize",
            {
                "protocolVersion": _MCP_PROTOCOL_VERSION,
                "capabilities": {},
                "clientInfo": {"name": "contextkit-adapter", "version": "0.1.0"},
            },
            timeout=15.0,
        )
        self.initialized = True
        self._notify("notifications/initialized")
        return result or {}

    def ensure_initialized(self) -> None:
        if not self.initialized:
            self.initialize()

    def _notify(self, method: str, params: dict | None = None) -> None:
        payload: dict = {"jsonrpc": "2.0", "method": method}
        if params is not None:
            payload["params"] = params
        try:
            self._post(payload, timeout=10.0)
        except RemoteError:
            pass  # notifications are fire-and-forget

    def list_tools(self) -> list[dict]:
        result = self._rpc("tools/list")
        return (result or {}).get("tools") or []

    def call_tool(self, name: str, arguments: dict | None = None) -> dict:
        result = self._rpc("tools/call", {"name": name, "arguments": arguments or {}})
        return result or {}