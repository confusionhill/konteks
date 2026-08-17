# ContextKit MCP Server

The ContextKit MCP server exposes ContextKit's room, context, and memory
capabilities to AI agents through the [Model Context Protocol](https://modelcontextprotocol.io).
It is a peer service to the main REST backend: it reads from the same PostgreSQL
+ pgvector database directly and uses its own slim repository/service layers
under `backend/internal/`.

- Language: Go 1.25+, built on `mark3labs/mcp-go` v0.57.0
- Entry point: `backend/cmd/mcp_server/main.go`
- Transports: stdio (default) and streamable HTTP (`/mcp`)

## Quick start

```bash
cd backend

# build
go build -o mcp_server ./cmd/mcp_server

# run over stdio (the API key is validated at startup)
CONTEXTKIT_API_KEY=ck_live_dev_test_key_123 \
DATABASE_URL="postgres://postgres:postgres@localhost:5435/konteks?sslmode=disable" \
./mcp_server --transport stdio

# run over HTTP
MCP_PORT=8099 ./mcp_server --transport http
```

### Prerequisites

- PostgreSQL 15 with the `vector` extension (migrations live in `db/migrations/`).
  Migrations are idempotent SQL files; apply them in order (01_init, 02, 03, 04,
  05, 07, 08) as done by `cmd/server/main.go`.
- An API key in the `api_keys` table. The repo seed command
  (`go run ./cmd/seed`) creates the dev key `ck_live_dev_test_key_123`.

## Configuration

| Setting              | Flag          | Env var            | Default |
| -------------------- | ------------- | ------------------ | ------- |
| Transport            | `--transport` | `MCP_TRANSPORT`    | `stdio` |
| HTTP port            | `--port`      | `MCP_PORT`         | `8099`  |
| Project API key      | `--api-key`   | `CONTEXTKIT_API_KEY` | —  (required) |
| Database URL         | —             | `DATABASE_URL`     | from `secrets/config.json` |

In stdio mode the API key is resolved at startup and applied to every request.
In HTTP mode each request must send the key via the `X-API-Key` header,
`Authorization: Bearer <key>`, **or** the `api_key` (aliased `token`) query
parameter on the request URL: `https://mcp.contextkit.app/mcp?api_key=ck_...`.
The query-param form exists for MCP clients that cannot set headers, such as
Claude Desktop / claude.ai *remote* MCP servers, which reach the server from
Anthropic's cloud and therefore require a publicly reachable HTTPS URL. Keep
tokens out of logs; rotate the key if it leaks via URL or proxy logs.

## Authentication model

The API key identifies the **project** only — there is no internal user binding
(no JWT, no user id). Rooms belong to a project; user scope is expressed with the
`external_user_id` parameter on room operations. Keys are validated against the
`api_keys` table by their SHA-256 hash, with revoked/expired checks and a 5-minute
in-memory cache. Unauthenticated HTTP tool calls are rejected with
`authentication failed`.

## Tools

| Tool                 | Purpose |
| -------------------- | ------- |
| `create_room`        | Create a persistent conversation room |
| `update_room`        | Update room metadata / parent / schema / default policy |
| `get_room`           | Fetch room + full state package |
| `list_rooms`         | List project rooms, optionally by external user |
| `get_room_state`     | Fetch extracted state (summary, active_topic, state, entities, references) |
| `update_room_state`  | Patch the room state object (set/remove keys) |
| `archive_room`       | Archive a room |
| `add_message`        | Append a message (optionally client-encrypted) |
| `get_messages`       | List message history with pagination |
| `get_context_package`| Assemble a token-budgeted context package (use before answering) |
| `add_memory`         | Persist a memory; embedding generated asynchronously |
| `search_memories`    | Semantic (cosine) search of a room's memories |
| `list_memories`      | List memories with pagination |
| `remove_memory`      | Delete a memory by ID |

### `get_context_package` details

Policy is optional and inline; defaults mirror the main backend:

```json
{
  "strategy": "auto",             // auto|recent_heavy|memory_heavy|hybrid
  "max_tokens": 4000,
  "max_history_depth": 50,
  "freshness_weight": 0.5
}
```

`fast_mode: true` skips LLM state re-extraction. The response includes
`tokens_estimated` and `tokens_processed` for budget accounting, plus a
`strategy_used` field. Child agent room summaries are included when
`include_children: true`.

### Memory lifecycle

`add_memory` returns immediately with `status: "embedding_pending"`; an
in-process worker pool (5 workers, 5000 buffer) generates the embedding via
OpenRouter and updates the row in the background. `search_memories` embeds the
query synchronously so results may take a few seconds on first use.

## Resources

Dynamic resource templates for read-only access:

| URI template | Contents |
| ------------ | -------- |
| `/contextkit/room/{room_id}`                | Room + state |
| `/contextkit/room/{room_id}/context`        | Assembled context package |
| `/contextkit/room/{room_id}/messages`       | Message history (up to 200) |
| `/contextkit/room/{room_id}/memories`       | Memory index (up to 200) |

## Connecting from an MCP client

### Claude Desktop

Add a stdio server to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "contextkit": {
      "command": "/absolute/path/to/mcp_server",
      "args": ["--transport", "stdio"],
      "env": {
        "CONTEXTKIT_API_KEY": "ck_live_dev_test_key_123",
        "DATABASE_URL": "postgres://postgres:postgres@localhost:5435/konteks?sslmode=disable"
      }
    }
  }
}
```

### HTTP (any MCP-capable agent)

```bash
# initialize (capture the Mcp-Session-Id header from the response)
curl -X POST http://localhost:8099/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "X-API-Key: ck_live_dev_test_key_123" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"1"}}}'

# then, with the returned Mcp-Session-Id:
curl -X POST http://localhost:8099/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: mcp-session-<id>" \
  -H "X-API-Key: ck_live_dev_test_key_123" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_room","arguments":{"name":"My Room"}}}'
```

## Testing

```bash
cd backend
go vet ./...
go test ./internal/mcp/ ./internal/service/ ./internal/worker/
```

Unit tests cover the room service, context assembly, memory lifecycle, the
async embedding worker, and the auth provider resolution/caching.