# Plan: MCP Server for ContextKit

## TL;DR
Build a **fresh Go MCP server** as a new command in `/backend/cmd/mcp_server/` that exposes ContextKit's core capabilities (room management, context retrieval, memory/artifact operations, observability) as tools and resources for AI agents. The server directly accesses the PostgreSQL database—no dependency on or reuse of existing handler/usecase layers.

**Technology:** Go with Go MCP SDK, direct PostgreSQL + pgvector connection.

**Architecture:** Peer service alongside the main server—fresh repository/service layers built specifically for MCP's tool-oriented interface.

**Key Patterns from app.py:**
- **external_user_id** = agent_id or customer's user identifier (NOT internal user.id; passed on room creation)
- **Context Policy** = flexible schema (strategy, max_tokens, max_history_depth, freshness_weight, dynamic_schema)
- **Encryption** = optional/mixed per-message and per-context call (is_encrypted flag + encryption_metadata)
- **API Key** = project_id only (no internal user binding); external_user_id provides user scope
- **include_children** = support for hierarchical rooms (parent/child agent orchestration)

---

## Discovery Findings

### ContextKit Architecture
- **Core abstractions:** Rooms (persistent conversations), Messages, RoomState, Memories, ContextPackages, Artifacts, Events
- **Backend capabilities:** Room CRUD, semantic search (pgvector), async compaction/extraction, event streaming (SSE), encryption support
- **Data model:** Rooms contain messages, state, memories, and artifacts; context packages are intelligently assembled outputs

### MCP Suitability
- Strong fit: Agents need programmatic access to ContextKit's room, message, memory, and context operations
- Agents can manage their own conversations, retrieve context, persist memories, and upload knowledge artifacts
- SSE events can be exposed as MCP resources for observability

### Existing Patterns
- Backend uses REST API (likely Echo framework based on file references)
- Python SDK exists for client integration
- OpenRouter for embeddings/LLM operations
- Authentication appears token/project-based

---

## Steps

### Phase 1: Core MCP Server Structure
1. Create `/backend/cmd/mcp_server/main.go` and project layout
2. Add Go MCP SDK dependency to go.mod
3. Initialize PostgreSQL connection pool (reuse config pattern from main server)
4. Implement MCP server lifecycle (start, listen, shutdown)
5. **Set up API key authentication middleware:**
   - Accept API key from MCP client connection (via environment or initialization)
   - Query `api_keys` table to validate key and extract `project_id`, check revoked/expired
   - Build auth context struct: `{ProjectID, APIKeyID, IsEncrypted}`
   - Attach context to all tool calls; verify project_id on every DB query
   - Return error if key invalid, revoked, or expired
6. Create error handling and response mapping layer (map DB errors to MCP tool errors)
7. Define auth context types to thread through service → repository layers

**Dependencies:** None (phase 0)

### Phase 2: Fresh Repository Layer
7. Build `/backend/internal/repository/mcp_*.go` layer (separate from existing handlers):
   - `room_repository.go` — direct room queries (all scoped to project_id from auth context)
   - `message_repository.go` — message operations (insert, retrieve history)
   - `memory_repository.go` — memory CRUD + pgvector cosine similarity search
   - `artifact_repository.go` — artifact storage and semantic search
   - `state_repository.go` — room state extraction

8. Build `/backend/internal/service/mcp_*.go` service layer (receives auth context):
   - `room_service.go` — room lifecycle, validate project_id ownership
   - `context_service.go` — assemble context packages (relevance ranking, token budgeting)
   - `memory_service.go` — semantic search, memory validation
   - `artifact_service.go` — artifact operations

**Dependencies:** Phase 1 complete

### Phase 3: Tools Implementation
9. Implement **Room Management Tools** (API key provides project_id; external_user_id passed per call):
   - `create_room(external_user_id, description?, state_schema?, default_policy?, parent_room_id?)` → room
   - `get_room(room_id)` → full room + state
   - `list_rooms(external_user_id?, limit=20, offset=0)` → paginated rooms (project-scoped)
   - `update_room_state(room_id, state_updates)` → updated state
   - `archive_room(room_id)` → room marked archived

10. Implement **Message & Context Tools:**
    - `add_message(room_id, role, content, is_encrypted=false, encryption_metadata?)` → message_id
    - `get_messages(room_id, limit=50, offset=0)` → message history
    - `get_context_package(room_id, query, policy={strategy, max_tokens, max_history_depth, freshness_weight}, dynamic_schema?, is_encrypted=false, include_children=false)` → assembled context
    - `get_room_state(room_id)` → summary, active_topic, state, entities

11. Implement **Memory & Search Tools:**
    - `add_memory(room_id, content, metadata?, external_user_id?)` → memory with pgvector embedding
    - `search_memories(room_id, query, limit=10)` → cosine similarity via pgvector
    - `list_memories(room_id, limit=20, offset=0)` → all memories with scores
    - `remove_memory(memory_id)` → deleted

12. Implement **Artifact Tools:**
    - `upload_artifact(room_id, name, content, metadata?)` → artifact (queue embedding async)
    - `search_artifacts(room_id, query, limit=10)` → semantic search
    - `list_artifacts(room_id, limit=20, offset=0)` → all artifacts



**Dependencies:** Phase 2 complete

### Phase 4: Resources & Configuration
14. Define **MCP Resources:**
    - `/contextkit/room/{room_id}` → room state (readable, dynamic)
    - `/contextkit/room/{room_id}/context` → current assembled context package
    - `/contextkit/room/{room_id}/messages` → message history
    - `/contextkit/room/{room_id}/memories` → memory index

15. Implement **Server Configuration:**
    - Load PostgreSQL connection, auth credentials from env vars
    - Reuse config pattern from [backend/internal/config/config.go](backend/internal/config/config.go)
    - Support graceful shutdown

16. Implement **Error Handling:**
    - Map database errors to MCP tool errors
    - Validation of required fields before DB operations
    - Transaction handling for multi-step operations

**Dependencies:** Phase 3 complete; *parallel with Phase 3*

### Phase 5: Testing & Documentation
17. Create **unit tests** for repository and service layers (mocking database)
18. Create **integration tests** with test database (docker postgres + migrations)
19. Write **MCP schema documentation** (tool descriptions, parameter schemas)
20. Add **example agent scripts** showing multi-turn usage
21. Create **deployment guide** (building, running, Docker)

**Dependencies:** Phases 1-4 complete; *parallel with Phase 4*

---

## Relevant Files

**Existing patterns to follow (don't reuse, but reference):**
- [backend/cmd/server/main.go](backend/cmd/server/main.go) — server startup pattern
- [backend/cmd/archiver/main.go](backend/cmd/archiver/main.go) — cmd app structure
- [backend/internal/config/config.go](backend/internal/config/config.go) — environment config pattern
- [backend/db/migrations/](backend/db/migrations/) — existing schema (queries will run against this)
- [backend/internal/domain/](backend/internal/domain/) — domain models (Room, Message, Memory, ContextPackage)

**MCP Server files to create (fresh implementation):**
- `backend/cmd/mcp_server/main.go` — MCP server entry point and CLI
- `backend/internal/repository/mcp_room.go` — direct room queries
- `backend/internal/repository/mcp_message.go` — message queries
- `backend/internal/repository/mcp_memory.go` — memory + pgvector search
- `backend/internal/repository/mcp_artifact.go` — artifact queries
- `backend/internal/repository/mcp_event.go` — event retrieval
- `backend/internal/service/mcp_room.go` — room service logic
- `backend/internal/service/mcp_context.go` — context package assembly
- `backend/internal/service/mcp_memory.go` — memory operations
- `backend/internal/service/mcp_artifact.go` — artifact operations
- `backend/internal/service/mcp_event.go` — observability service
- `backend/internal/mcp/tools.go` — MCP tool definitions
- `backend/internal/mcp/resources.go` — MCP resource definitions
- `tests/mcp_*.go` — integration tests

---

## Verification

1. **Unit Tests:** Tools correctly serialize/deserialize ContextKit responses
   - Test each tool with mocked backend responses
   - Verify error handling for 400, 401, 404, 500 responses

2. **Integration Tests:** Full workflow with mock ContextKit backend
   - Create room → add message → get context package → search memories
   - Verify tool responses match MCP schema
   - Test pagination, filtering, and edge cases

3. **Manual Testing:** Using Claude or other MCP client
   - Connect MCP server to Claude desktop
   - Create a room and carry a multi-turn conversation
   - Retrieve context package and verify relevance
   - Search and add memories

4. **Deployment Test:**
   - Run Docker container locally
   - Connect to live ContextKit backend (dev environment)
   - Execute agent workflow end-to-end

5. **Documentation Review:**
   - Verify tool descriptions are clear and match implementation
   - Test example scripts run without errors
   - Confirm setup guide is complete

---

## Decisions & Assumptions

- **Language:** Go (same stack as backend, direct database access, no HTTP wrapper needed)
- **Architecture:** Peer service alongside main server and archiver; fresh repository/service layers
- **Database Access:** Direct PostgreSQL + pgvector connection (no dependency on handler layer)
- **Shared Code:** Domain models only ([backend/internal/domain/](backend/internal/domain/)); no reuse of handler or usecase layers
- **MCP SDK:** Go MCP SDK
- **Authentication:** 
  - API key from env var or MCP client context → validate against `api_keys` table
  - Extract `project_id` (scoped to all queries)
  - **No internal user_id binding**; external_user_id is passed per-tool call
  - Auth context: `{ProjectID, APIKeyID, IsEncrypted}`
- **Tool Parameters:** 
  - API key implicit (project_id) + external_user_id explicit per call (for agent/user tracking)
  - room_id for room-scoped operations
  - Context Policy for flexible budgeting (not just max_tokens)
  - Encryption optional/mixed per-message
- **Embeddings:** Follow existing async worker pattern (queue to worker pool, not sync in MCP)
- **Scope Inclusions:**
  - All core ContextKit operations (room, message, context, memory, artifact)
  - Context policy-driven retrieval (strategy, max_tokens, freshness_weight, etc.)
  - Hierarchical rooms (parent/child with include_children)
  - Optional encryption per-message and per-context
  - **Observability (metrics, events, evaluations) via web dashboard only—NOT agentic**
- **Scope Exclusions:**
  - Admin/user/project management (handled by main backend)
  - Real-time streaming via SSE (MCP resources only, on-demand retrieval)
  - Zero-knowledge encryption key management (client responsibility)
  - LLM model selection (use existing OpenRouter integration if needed)
  - **Observability (metrics, events, evaluations)** — deferred to web dashboard for now
  - **Observability (metrics, events, evaluations)** — defer to web dashboard for now

---

## Implementation Details

### **Response Format & Pagination**
Follow REST API pattern from handlers:
```go
// List responses
map[string]any{
    "total": total,
    "limit": limit,
    "offset": offset,
    "{resource_plural}": resources,  // "artifacts", "messages", "rooms", etc.
}

// Single resource responses
c.JSON(http.StatusOK, resource)

// Error responses
map[string]string{"error": err.Error()}
```

### **Context Query Parameter**
- **OPTIONAL** (already handled by backend usecase layer)
- If not provided, backend uses room context implicitly
- Follow RestAPI handler pattern in context_handler.go

### **Async Embedding Architecture (Option A)**
MCP server implements its own **buffered channel worker pool**:
- **5 concurrent workers** (matches backend)
- **5000 buffer depth** task queue
- Workers call OpenRouter API directly for embeddings
- Both `add_memory` and `upload_artifact` return immediately after DB persist
- Embeddings generated asynchronously in background
- Pattern matches backend message_usecase + artifact_usecase

**Implementation:**
1. Create `/backend/internal/worker/mcp_worker.go` — task queue + worker pool
2. Define `EmbeddingTask` struct (project_id, object_id, content, type)
3. Start 5 workers on MCP server startup
4. `add_memory` → DB persist → queue embedding task → return
5. `upload_artifact` → DB persist + chunk → queue embedding tasks → return

