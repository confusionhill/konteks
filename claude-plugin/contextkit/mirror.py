"""Schema transform + per-call policy for reaching the remote MCP.

The remote ContextKit server keeps ``room_id`` a required argument on
room-scoped tools (create_room/list_rooms excluded), so it can validate against
the authenticated project on every call. This module makes room_id optional for
the model by:

* mirroring each remote tool's schema with ``room_id`` dropped from ``required``,
* injecting the room bound to the current project directory into the arguments
  of any room-scoped call that omits it.

The model can still pass an explicit ``room_id``; the server enforces that it
belongs to the project, so tool arguments can never cross tenant boundaries.
"""

from __future__ import annotations

import copy

from typing import Any

ROOM_PROPERTY = "room_id"


def is_room_scoped(tool: dict) -> bool:
    schema = tool.get("inputSchema") or {}
    properties = schema.get("properties") or {}
    return ROOM_PROPERTY in properties


def tool_is_room_scoped(name: str, tools: list[dict]) -> bool:
    for tool in tools:
        if tool.get("name") == name:
            return is_room_scoped(tool)
    return False


def mirror_tool(tool: dict) -> dict:
    """Return a copy of a remote tool whose room_id param is no longer required.

    Every other property (including the optional room_id property itself) is
    preserved so the model can still override the room explicitly.
    """
    clone = copy.deepcopy(tool)
    schema = clone.get("inputSchema")
    if not schema or ROOM_PROPERTY not in (schema.get("properties") or {}):
        return clone
    required = schema.get("required") or []
    if ROOM_PROPERTY in required:
        schema["required"] = [name for name in required if name != ROOM_PROPERTY]
    return clone


def inject_room_id(
    tool_name: str,
    arguments: dict[str, Any],
    tools: list[dict],
    room_id: str | None,
) -> dict[str, Any]:
    """Return a copy of arguments with the bound room injected when needed.

    Inject only when the tool is room-scoped, the call did not already provide
    a room_id, and a bound room exists. ``room_id`` is None when the vehicle is
    not configured; the adapter rejects earlier in that case.
    """
    args = dict(arguments or {})
    if ROOM_PROPERTY in args:
        return args
    if room_id and tool_is_room_scoped(tool_name, tools):
        args[ROOM_PROPERTY] = room_id
    return args