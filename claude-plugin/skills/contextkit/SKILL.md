---
name: contextkit
description: Persistent shared context. Use whenever the user wants to connect, configure, set up, or use ContextKit, manage rooms or shared memory, or switch or create a context room. Also use when context tools return "not configured" errors.
---

# ContextKit — persistent shared context for this Claude Code installation

ContextKit is a remote service that gives different agents, chat sessions, and
Claude Code installations access to the same persistent context through a
**room**. This plugin is configured entirely through conversation — no dashboard
or CLI needed.

Whenever the user asks to set up, connect, use, or switch ContextKit context, or
when a Context * tool reports `ContextKitNotConfigured`, run the **Setup** flow
below.

## Quick reference

- `contextkit_get_config` — is this installation configured? Which room?
- `contextkit_set_key` — securely store the project API key (validated first).
- `contextkit_set_room` — bind the current project directory to a room.
- `contextkit_unset_room` — remove the room binding.
- `contextkit_clear` — wipe the key and all bindings.

The ContextKit MCP server (tools like `create_room`, `list_rooms`, `get_room`,
`get_context_package`, `add_message`, `add_memory`, ...) belongs to this plugin.
Room-scoped tools automatically act on the room bound to the current project
directory — **do not ask the user for a room id on every call**.

## Setup flow (run in this order)

1. **Check config.** Call `contextkit_get_config`. If `configured` is `true`,
   report the current room and stop — nothing to do.
2. **Ask for the API key.** When the user provides it, call `contextkit_set_key`
   right away. After the call, **never repeat, echo, or log the key** in any
   subsequent message. Only report what came back (`stored: true`, project id).
   Do not guess or fabricate a key.
3. **Offer rooms.** Call `list_rooms` (project-scoped automatically).
   - If rooms exist, present them as numbered options and ask the user which to use.
   - Always offer "create a new room" as an option.
4. **Use an existing room.** Once the user picks one, call `contextkit_set_room`
   with that `room_id`. done.
5. **Create a new room** (when chosen). Collect without overwhelming the user:
   - `description` — short purpose (recommended).
   - `external_user_id` — optional agent/customer identifier.
   - `state_schema` — optional empty object `{}` unless the user has a schema.
   - `default_policy` — optional `{"strategy": "auto"}`.
   - `initial_state` — optional empty object `{}`.
   Call `create_room` with what the user actually cares about, then
   `contextkit_set_room` with the returned `room.id`.
6. **Confirm.** Call `contextkit_get_config` and tell the user the connected room
   (and that it is shared with other agents on the same room).

If `contextkit_set_room` fails, the server returned an error — re-check the room
id and that it belongs to the project of the stored key.

## Switching rooms

When the user says things like "put my context in room 2", "switch to the other
room", or wants the context to be a different room:

1. Call `list_rooms` to enumerate rooms.
2. Pick/create the target room (same create flow as above if needed).
3. Call `contextkit_set_room`.
4. Confirm with `contextkit_get_config`; if it reports a previous room, that is
   expected and fine — it means the project directory switched rooms.

## Sharing context with other agents

Context is shared per room, not per installation. Any other agent that stores
and reads the same room can see this installation's context and vice versa.
Make this explicit to the user when they ask, e.g. "Which room should this agent
use?" or "Can I chat with context from my WhatsApp agent?".

## Troubleshooting errors

- `ContextKitNotConfigured: no API key...` → run the setup flow, step 2.
- `ContextKitNotConfigured: no room is bound...` → run the setup flow, steps 3–6.
- `authentication failed: API key invalid, revoked, or expired` → the stored key
  is stale. Ask for a new key and call `contextkit_set_key` again. Rooms do not
  need to be re-bound afterwards.
- `room not found or unauthorized` → the room id does not belong to the project
  of the stored key. Re-check with `list_rooms`.

## Security rules

- Never repeat the API key after `contextkit_set_key` returns.
- Never write the key to a file, message, or log yourself — the adapter handles
  storage in the macOS Keychain (or a 0600 file on other platforms).
- If the user wants to use a different project, they need a different API key.