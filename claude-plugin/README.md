# ContextKit — Claude Code plugin

Persistent shared context for Claude Code, backed by [ContextKit](https://contextkit.app)
(Context-as-a-Service over a remote MCP server). Different Claude Code
installations, chat sessions, and other agents can share the same **room** and
therefore the same persistent context.

```
Claude Code
   │  (stdio MCP)
   ▼
contextkit-adapter (thin local proxy, this repo)
   │  room binding  → ${CLAUDE_PLUGIN_DATA}/state.json
   │  API key       → macOS Keychain (or 0600 file)
   │  forwards      → X-API-Key header
   ▼
Remote ContextKit MCP  →  Project  →  Room  →  persistent context
```

This plugin holds **no business logic**. Every real operation is forwarded to
the remote ContextKit MCP server with the stored API key. The adapter only
manages local onboarding state, injects the configured room into each call, and
gate-keeps context tools until the installation is configured.

## Requirements

- Claude Code (recent version with plugin support)
- Python 3.9+ on `PATH` as `python3`
- A ContextKit project API key

## Install

From within Claude Code:

```
/plugin install contextkit@<your-marketplace>
```

Or develop locally:

```
claude --plugin-dir ./claude-plugin
```

Enable the plugin, then just say:

> **Connect my context.**

Claude walks you through it conversationally:

1. asks for your project API key (stored in the macOS Keychain),
2. shows your rooms (or creates a new one),
3. binds the current project directory to the chosen room.

No dashboard, no CLI, no config file editing.

## What is stored and where

| Piece | Location |
| --- | --- |
| API key | macOS Keychain (`contextkit` / `contextkit-claude-plugin`), or `api_key` (0600) in the state dir on other platforms |
| Room binding | `${CLAUDE_PLUGIN_DATA}/state.json` (falls back to `~/.contextkit/state.json`) |

Bindings are keyed by the **project directory** (`CLAUDE_PROJECT_DIR`), so two
different Claude Code projects on the same machine use different rooms — even
with the same API key. `CLAUDE_PLUGIN_DATA` survives plugin updates and is
removed on uninstall.

## Tools

Plugin MCP server `contextkit-adapter` provides:

**Admin (setup / local):**

- `contextkit_get_config` — is this installation configured? Which room?
- `contextkit_set_key` — validate + securely store the API key.
- `contextkit_set_room` — bind the current project directory to a room.
- `contextkit_unset_room` — remove the room binding.
- `contextkit_clear` — wipe key and bindings.

**Context (mirrored from the remote server):** `create_room`, `list_rooms`,
`get_room`, `get_room_state`, `update_room_state`, `archive_room`,
`add_message`, `get_messages`, `get_context_package`, `add_memory`,
`search_memories`, `list_memories`, `remove_memory`. `room_id` is optional in
their schemas and is injected automatically from the bound room.

## Switching rooms

> **Switch my context to Room 2.**

Claude lists the rooms, you pick one, and it re-binds the current project
directory. Takes effect on the next call — no restart.

## Configuration

The remote endpoint is set with the `CONTEXTKIT_REMOTE_URL` environment
variable in `.mcp.json` (default `https://mcp.contextkit.app/mcp`). For local
development against the Go server:

```json
{ "CONTEXTKIT_REMOTE_URL": "http://localhost:8099/mcp" }
```

Environment variables respected by the adapter:

| Variable | Purpose |
| --- | --- |
| `CONTEXTKIT_REMOTE_URL` | Remote MCP endpoint (default `https://mcp.contextkit.app/mcp`) |
| `CLAUDE_PLUGIN_DATA` | State directory (set by Claude Code) |
| `CLAUDE_PROJECT_DIR` | Project directory used to key room bindings |
| `CONTEXTKIT_KEY_STORE=file` | Force the 0600-file key store (tests, no keychain) |
| `CONTEXTKIT_LOG_LEVEL` | Adapter logging level (`DEBUG`/`INFO`/`WARN`/`ERROR`) |

## Development & tests

```bash
python3 -m unittest discover -s claude-plugin/tests
```

Tests run against an in-process fake remote MCP server, so no database or
network is required. To test against the real local Go MCP server after running
it (`cd backend && MCP_PORT=8099 ./mcp_server --transport http`), set
`CONTEXTKIT_REMOTE_URL=http://localhost:8099/mcp` and use a valid key.

## Security notes

- The API key is validated against the remote server **before** it is stored.
- The key is never written to `state.json` or to stdout/logs; the adapter only
  sends it as the `X-API-Key` header.
- `room_id` is validated server-side on every call — a call can never reach a
  room outside the authenticated project, even if the model passes a room id
  explicitly.
- macOS note: the `security(1)` CLI receives the key as an argument the moment
  it is stored (briefly visible in the process list). Tokens should be treated
  as machine-level secrets anyway; rotate them if in doubt.