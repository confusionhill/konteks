"""Multi-turn example agent that drives the ContextKit MCP server over HTTP.

Shows the full agent loop: create a room, exchange a few messages, persist a
memory, assemble a context package to ground a response, then search memories.

Requirements:
  - ContextKit MCP server running with --transport http (see backend/docs/mcp.md)
  - An API key valid for the project (e.g. the seeded ck_live_dev_test_key_123)

Run:
  python example/mcp_agent.py
"""

import json
import urllib.request

BASE_URL = "http://localhost:8099/mcp"
API_KEY = "ck_live_dev_test_key_123"

def post(payload, session=None, accept="application/json, text/event-stream"):
    headers = {
        "Content-Type": "application/json",
        "Accept": accept,
        "X-API-Key": API_KEY,
    }
    if session:
        headers["Mcp-Session-Id"] = session
    req = urllib.request.Request(
        BASE_URL, data=json.dumps(payload).encode(), headers=headers, method="POST"
    )
    with urllib.request.urlopen(req) as resp:
        body = resp.read().decode()
        new_session = resp.headers.get("Mcp-Session-Id") or session
        result = json.loads(body) if body.strip().startswith("{") else body
    return new_session, result

def call(session, method, params):
    session, msg = post({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}, session)
    if "error" in msg:
        raise RuntimeError(f"{method}: {msg['error']}")
    return session, msg.get("result")

def tool(session, name, arguments):
    session, result = call(session, "tools/call", {"name": name, "arguments": arguments})
    content = result.get("content") or []
    text = content[0].get("text", "") if content else ""
    if result.get("isError"):
        raise RuntimeError(f"{name} failed: {text}")
    return session, json.loads(text)

def main():
    # 1. initialize handshake (captures the MCP session id)
    session, result = post({
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2025-06-18",
            "capabilities": {},
            "clientInfo": {"name": "example-agent", "version": "1.0"},
        },
    })
    if not session:
        raise RuntimeError("no Mcp-Session-Id returned by initialize")
    post({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}}, session)

    # 2. create a room for an external (agent) user
    session, room = tool(session, "create_room", {
        "name": "Travel Assistant",
        "external_user_id": "customer-42",
        "description": "Helps the customer plan business trips.",
        "initial_state": {"currency": "USD", "trips": 0},
    })
    room_id = room["id"]
    print(f"[room] {room_id}")

    # 3. multi-turn conversation
    turns = [
        ("user", "I need to fly to Berlin next Monday for a conference."),
        ("assistant", "I can help with that. Do you prefer a morning or evening flight?"),
        ("user", "Morning. And I'd like to avoid red-eye flights."),
        ("assistant", "Noted. I'll look at Monday morning departures."),
    ]
    for role, content in turns:
        tool(session, "add_message", {"room_id": room_id, "role": role, "content": content})
    print("[messages] 4 appended")

    # 4. persist a long-term memory (embedding fills in asynchronously)
    session, mem = tool(session, "add_memory", {
        "room_id": room_id,
        "content": "customer-42 prefers morning flights and avoids red-eyes.",
        "metadata": {"type": "preference"},
    })
    print(f"[memory] {mem['memory_id']} status={mem['status']}")

    # 5. assemble context package to ground the next assistant turn
    _, pkg = tool(session, "get_context_package", {
        "room_id": room_id,
        "query": "what flight should I book for Berlin?",
    })
    print(f"[context] strategy={pkg['strategy_used']} "
          f"estimated_tokens={pkg['tokens_estimated']} "
          f"relevant_messages={len(pkg['relevant_messages'])} "
          f"memories={len(pkg['memories'])}")

    # 6. semantic search for the preference we just stored
    _, search = tool(session, "search_memories", {
        "room_id": room_id,
        "query": "flight preferences",
    })
    print(f"[search] {[m['content'][:50] for m in search['memories']]}")

    # 7. patch room state to keep the extracted state in sync
    _, state = tool(session, "update_room_state", {
        "room_id": room_id,
        "state_updates": {"trips": 1},
    })
    print(f"[state] trips={state['state'].get('trips')}")

if __name__ == "__main__":
    main()