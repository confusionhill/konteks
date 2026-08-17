#!/usr/bin/env python3
"""ContextKit Claude Code adapter — a thin stdio MCP server.

The adapter:

* mirrors the remote ContextKit MCP server's tools (``room_id`` made optional),
* injects the room bound to the current project directory into each call,
* gate-keeps context tools until the installation is configured,
* exposes ``contextkit_*`` admin tools that drive conversational setup.

It holds no business logic: every real operation is forwarded to the remote
server with the stored API key as an ``X-API-Key`` header. All local state
lives under ``${CLAUDE_PLUGIN_DATA}`` (or ``~/.contextkit``); the API key is
stored in the macOS Keychain or a 0600 file.

Critical protocol rule: stdout carries ONLY newline-delimited JSON-RPC. All
logging goes to stderr.
"""

from __future__ import annotations

import logging
import os
import sys
import uuid

from contextkit import protocol, state as state_mod, tools as tools_mod
from contextkit.mirror import inject_room_id, mirror_tool, tool_is_room_scoped
from contextkit.proxy import AuthError, RemoteClient, RemoteError

logging.basicConfig(
    level=os.environ.get("CONTEXTKIT_LOG_LEVEL", "WARN").upper(),
    stream=sys.stderr,
    format="[contextkit] %(levelname)s %(message)s",
)
log = logging.getLogger("contextkit.adapter")

ADAPTER_VERSION = "0.1.0"


class ContextKitAdapter:
    def __init__(self, stdin, stdout):
        self.stdin = stdin
        self.stdout = stdout
        self.cfg = state_mod.load_config()
        state_mod.ensure_installation_id(self.cfg)
        self.key_store = state_mod.default_key_store()
        self.remote: RemoteClient | None = None
        self._remote_tools: list | None = None

    # ------------------------------------------------------------------ #
    # helpers
    # ------------------------------------------------------------------ #

    def _api_key(self) -> str | None:
        return self.key_store.get()

    def _get_remote(self) -> RemoteClient:
        if self.remote is None:
            self.remote = RemoteClient(state_mod.remote_url(self.cfg), self._api_key() or "")
        return self.remote

    def _invalidate(self) -> None:
        self.remote = None
        self._remote_tools = None

    # ------------------------------------------------------------------ #
    # JSON-RPC loop
    # ------------------------------------------------------------------ #

    def serve(self) -> None:
        while True:
            try:
                message = protocol.read_message(self.stdin)
            except ValueError as exc:
                log.warning("dropping malformed message: %s", exc)
                continue
            if message is None:
                return
            self._handle(message)

    def _handle(self, message: dict) -> None:
        method = message.get("method")
        req_id = message.get("id")
        params = message.get("params") or {}

        if method and req_id is None:
            self._handle_notification(method, params)
            return

        response = None
        if method == "initialize":
            response = protocol.make_response(req_id, self._handle_initialize(params))
        elif method == "ping":
            response = protocol.make_response(req_id, {})
        elif method == "tools/list":
            response = protocol.make_response(req_id, {"tools": self.tools_list()})
        elif method == "tools/call":
            response = protocol.make_response(
                req_id,
                self.tools_call(params.get("name"), params.get("arguments") or {}),
            )
        elif method == "resources/list":
            response = protocol.make_response(req_id, {"resources": []})
        elif method == "prompts/list":
            response = protocol.make_response(req_id, {"prompts": []})
        elif method:
            response = protocol.make_error(req_id, -32601, f"method not found: {method}")

        if response is not None:
            protocol.write_message(self.stdout, response)

    @staticmethod
    def _handle_notification(method: str, params: dict) -> None:
        ignored = {
            "notifications/initialized",
            "notifications/cancelled",
            "notifications/roots/list_changed",
        }
        if method not in ignored:
            log.debug("ignoring notification %s", method)

    def _handle_initialize(self, params: dict) -> dict:
        version = params.get("protocolVersion") or "2025-06-18"
        return {
            "protocolVersion": version,
            "capabilities": {"tools": {"listChanged": False}},
            "serverInfo": {"name": "contextkit", "version": ADAPTER_VERSION},
        }

    # ------------------------------------------------------------------ #
    # tools/list and tools/call
    # ------------------------------------------------------------------ #

    def tools_list(self) -> list[dict]:
        admin = tools_mod.admin_tool_specs()
        mirrored = [mirror_tool(tool) for tool in self._remote_tool_specs()]
        return admin + mirrored

    def _remote_tool_specs(self) -> list[dict]:
        if self._remote_tools is not None:
            return self._remote_tools
        if not self._api_key():
            self._remote_tools = []
            return self._remote_tools
        try:
            client = self._get_remote()
            client.ensure_initialized()
            self._remote_tools = client.list_tools()
        except (AuthError, RemoteError) as exc:
            log.warning("remote tools unavailable: %s", exc)
            self._remote_tools = []
        return self._remote_tools

    def tools_call(self, name: str, arguments: dict) -> dict:
        if not name:
            return tools_mod.tool_error("tool name is required")
        if name.startswith("contextkit_"):
            return self._admin_call(name, arguments)
        return self._remote_call(name, arguments)

    def _admin_call(self, name: str, arguments: dict) -> dict:
        dispatcher = tools_mod.AdminDispatcher(self._get_remote(), self.cfg, self.key_store)
        result = dispatcher.dispatch(name, arguments)
        if name in _CONFIG_MUTATORS:
            self.cfg = state_mod.load_config()
            self._invalidate()
        return result

    def _remote_call(self, name: str, arguments: dict) -> dict:
        if not self._api_key():
            return tools_mod.tool_error(
                tools_mod.NOT_CONFIGURED
                + ": no API key is stored. Run contextkit_set_key during setup first."
            )

        binding = state_mod.get_binding(self.cfg)
        if binding is None and tool_is_room_scoped(name, self._remote_tool_specs()):
            return tools_mod.tool_error(
                tools_mod.NOT_CONFIGURED
                + ": no room is bound to this project directory. Run contextkit_set_room or "
                "switch rooms (contextkit_set_room) first."
            )

        client = self._get_remote()
        try:
            client.ensure_initialized()
            args = inject_room_id(
                name, arguments, self._remote_tool_specs(), binding["room_id"] if binding else None
            )
            return client.call_tool(name, args)
        except AuthError as exc:
            return tools_mod.tool_error(str(exc))
        except RemoteError as exc:
            return tools_mod.tool_error(f"ContextKit remote error: {exc}")


_CONFIG_MUTATORS = {
    "contextkit_set_key",
    "contextkit_set_room",
    "contextkit_unset_room",
    "contextkit_clear",
}


def main() -> None:
    try:
        ContextKitAdapter(sys.stdin, sys.stdout).serve()
    except (KeyboardInterrupt, BrokenPipeError):
        pass


if __name__ == "__main__":
    main()