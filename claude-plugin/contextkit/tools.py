"""Adapter-local admin tools used during conversational setup.

These are the only tools that touch local state directly. Keeping them as MCP
tools means Claude drives setup through the protocol (no guessed file paths),
and persistence is deterministic: the dispatcher owns every write.

The API key is validated against the remote server *before* it is stored, using
the existing ``list_rooms`` tool (a successful response means the key is valid,
and the first room round-trips a ``get_room`` that exposes the project id).
"""

from __future__ import annotations

import json

from typing import Any

from . import state as state_mod
from .proxy import AuthError, RemoteClient, RemoteError

NOT_CONFIGURED = "ContextKitNotConfigured"


def tool_result(data: Any) -> dict[str, Any]:
    return {"content": [{"type": "text", "text": json.dumps(data, indent=2, ensure_ascii=False)}]}


def tool_error(message: str) -> dict[str, Any]:
    return {"isError": True, "content": [{"type": "text", "text": str(message)}]}


def admin_tool_specs() -> list[dict]:
    return [
        {
            "name": "contextkit_get_config",
            "description": (
                "Report whether this ContextKit installation is configured: whether an API key "
                "is stored and which room is bound to the current project directory. Call this "
                "first during setup and after any room switch."
            ),
            "inputSchema": {"type": "object", "properties": {}},
        },
        {
            "name": "contextkit_set_key",
            "description": (
                "Validate and securely store the ContextKit project API key (macOS Keychain or a "
                "0600 file). The key is validated against the remote server before storage. Call "
                "once during setup and then NEVER repeat or echo the key in conversation."
            ),
            "inputSchema": {
                "type": "object",
                "properties": {
                    "api_key": {
                        "type": "string",
                        "description": "ContextKit project API key (e.g. ck_live_...)",
                    }
                },
                "required": ["api_key"],
            },
        },
        {
            "name": "contextkit_set_room",
            "description": (
                "Validate and bind the current project directory to an existing room. The room "
                "must belong to the project of the stored API key; the server validates this. "
                "Every subsequent room-scoped context tool call will target this room."
            ),
            "inputSchema": {
                "type": "object",
                "properties": {
                    "room_id": {"type": "string", "description": "Existing room id (e.g. room_123)"},
                    "room_name": {
                        "type": "string",
                        "description": "Optional human-readable room name for display",
                    },
                },
                "required": ["room_id"],
            },
        },
        {
            "name": "contextkit_unset_room",
            "description": (
                "Remove the room binding for the current project directory. After this, "
                "room-scoped context tools return ContextKitNotConfigured until a room is bound."
            ),
            "inputSchema": {"type": "object", "properties": {}},
        },
        {
            "name": "contextkit_clear",
            "description": (
                "Delete the stored API key and every room binding for this installation. "
                "Run contextkit_set_key and contextkit_set_room again to re-configure."
            ),
            "inputSchema": {"type": "object", "properties": {}},
        },
    ]


def _extract_json(result: dict) -> Any:
    """Pull the first text content out of an MCP tool result and parse it."""
    content = result.get("content") or []
    for item in content:
        if item.get("type") != "text":
            continue
        text = item.get("text", "")
        try:
            return json.loads(text)
        except (ValueError, TypeError):
            return text
    return {}


class AdminDispatcher:
    """Runs the contextkit_* tools against local state and the remote proxy."""

    def __init__(self, proxy: RemoteClient | None, cfg: dict, key_store: state_mod.KeyStore):
        self.proxy = proxy
        self.cfg = cfg
        self.key_store = key_store

    def has_key(self) -> bool:
        return self.key_store.has()

    def dispatch(self, name: str, arguments: dict) -> dict[str, Any]:
        handler = getattr(self, name.replace("contextkit_", "handle_", 1), None)
        if handler is None:
            return tool_error(f"unknown admin tool: {name}")
        try:
            result = handler(arguments or {})
        except AuthError as exc:
            return tool_error(str(exc))
        except RemoteError as exc:
            return tool_error(f"remote error: {exc}")
        return result

    # ------------------------------------------------------------------ #

    def handle_get_config(self, arguments: dict) -> dict[str, Any]:
        binding = state_mod.get_binding(self.cfg)
        return tool_result(
            {
                "configured": self.has_key() and binding is not None,
                "has_api_key": self.has_key(),
                "project_id": self.cfg.get("project_id") or None,
                "project_name": self.cfg.get("project_name") or None,
                "room_id": binding.get("room_id") if binding else None,
                "room_name": binding.get("room_name") if binding else None,
                "bindings": self.cfg.get("bindings", []),
                "remote_url": state_mod.remote_url(self.cfg),
            }
        )

    def handle_set_key(self, arguments: dict) -> dict[str, Any]:
        raw = (arguments.get("api_key") or "").strip()
        if not raw:
            return tool_error("api_key is required")

        client = RemoteClient(state_mod.remote_url(self.cfg), raw)
        client.ensure_initialized()

        # Validate the key: any well-formed list_rooms response means it is
        # accepted. If rooms exist, read one to learn the project id.
        rooms = _extract_json(client.call_tool("list_rooms", {"limit": 1})) or {}
        project_id = self.cfg.get("project_id") or ""
        room_id = ""
        room_list = rooms.get("rooms") or []
        if room_list:
            room_id = room_list[0].get("id") or ""
        if room_id:
            room_result = client.call_tool("get_room", {"room_id": room_id})
            room = _extract_json(room_result).get("room") or {}
            project_id = room.get("project_id") or project_id

        self.key_store.set(raw)
        self.cfg["project_id"] = project_id
        state_mod.save_config(self.cfg)

        return tool_result(
            {
                "stored": True,
                "project_id": project_id or None,
                "project_name": None,
                # Do NOT return the key here; it must stay out of the transcript.
            }
        )

    def handle_set_room(self, arguments: dict) -> dict[str, Any]:
        room_id = (arguments.get("room_id") or "").strip()
        if not room_id:
            return tool_error("room_id is required")
        if not self.has_key():
            return tool_error(
                "no API key is stored; run contextkit_set_key before choosing a room"
            )

        if self.proxy is None:
            return tool_error("remote MCP is not configured")
        self.proxy.ensure_initialized()

        # Server-authoritative validation: get_room returns "room not found or
        # unauthorized" unless the room belongs to the authenticated project.
        room_result = self.proxy.call_tool("get_room", {"room_id": room_id})
        room = _extract_json(room_result).get("room") or {}

        room_name = (arguments.get("room_name") or "").strip()
        if not room_name:
            room_name = room.get("description") or room_id
        if not self.cfg.get("project_id"):
            self.cfg["project_id"] = room.get("project_id") or ""
            state_mod.save_config(self.cfg)

        previous = state_mod.get_binding(self.cfg)
        binding = state_mod.set_binding(self.cfg, room_id, room_name)
        return tool_result(
            {
                "binding_project": binding["project_dir_hash"],
                "room_id": room_id,
                "room_name": room_name,
                "previous_room_id": previous.get("room_id") if previous else None,
                "previous_room_name": previous.get("room_name") if previous else None,
            }
        )

    def handle_unset_room(self, arguments: dict) -> dict[str, Any]:
        state_mod.unset_binding(self.cfg)
        return tool_result({"unset": True})

    def handle_clear(self, arguments: dict) -> dict[str, Any]:
        self.key_store.delete()
        state_mod.clear_config()
        self.cfg = state_mod.default_config()
        if self.proxy is not None:
            self.proxy.api_key = ""
            self.proxy.reset()
        return tool_result({"cleared": True})