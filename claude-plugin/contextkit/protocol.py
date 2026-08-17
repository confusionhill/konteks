"""JSON-RPC framing over stdio and streamable-HTTP response parsing.

The MCP stdio transport is newline-delimited JSON: one JSON-RPC message per
line, read from stdin and written to stdout. Everything the adapter logs must
go to stderr, otherwise it corrupts the protocol stream.

The remote ContextKit MCP server speaks streamable HTTP. Depending on the
``Accept`` header it may answer with a plain JSON body or a
``text/event-stream`` body; both are handled here.
"""

from __future__ import annotations

import json

from typing import Any


def read_message(stream) -> dict | None:
    """Read one JSON-RPC message from a line-delimited stream.

    Returns ``None`` at EOF. Raises ``ValueError`` for malformed JSON.
    """
    while True:
        line = stream.readline()
        if not line:
            return None
        line = line.strip()
        if not line:
            continue
        try:
            return json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid JSON-RPC message: {exc}") from exc


def write_message(stream, message: dict) -> None:
    data = json.dumps(message, ensure_ascii=False, separators=(",", ":"))
    stream.write(data + "\n")
    stream.flush()


def make_response(req_id: Any, result: Any = None, error: dict | None = None) -> dict:
    message = {"jsonrpc": "2.0", "id": req_id}
    if error is not None:
        message["error"] = error
    else:
        message["result"] = result
    return message


def make_error(req_id: Any, code: int, message: str) -> dict:
    return make_response(req_id, error={"code": code, "message": message})


def parse_sse(body: str) -> list[dict]:
    """Parse a streamable-HTTP text/event-stream body into JSON-RPC objects.

    ``data:`` payload lines belonging to the same event are joined with ``\\n``
    (as the SSE spec requires) and parsed as JSON. Returns every object found;
    a request-level response typically contains one. Malformed payloads are
    skipped.
    """
    objects: list[dict] = []
    payload_lines: list[str] = []

    for raw in body.splitlines():
        line = raw.strip()
        if line == "":
            _flush(payload_lines, objects)
            continue
        if line.startswith("data:"):
            payload_lines.append(line[len("data:"):].strip())
        # event:, id:, retry:, and comment lines are ignored for our purposes.

    _flush(payload_lines, objects)
    return objects


def _flush(payload_lines: list[str], objects: list[dict]) -> None:
    if not payload_lines:
        return
    payload = "\n".join(payload_lines)
    payload_lines.clear()
    try:
        objects.append(json.loads(payload))
    except json.JSONDecodeError:
        pass