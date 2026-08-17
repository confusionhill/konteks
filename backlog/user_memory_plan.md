# Project-Bound User Memory Feature Plan

Design and plan a multi-room, project-bound User Memory system in ContextKit. This feature collects and synthesizes behavioral traits, tone, speaking style, personality characteristics, and explicit user preferences for an end-user (`external_user_id`) across all rooms accessed under a given project.

---

## 1. Key Architectural Requirements

1. **Incremental Cumulative Learning**: User Memory profile grows and refines continuously across conversation turns in any room owned by the user under the project.
2. **API Opt-In / Opt-Out Control**: End users or integration clients can explicitly control whether a conversation turn is ingested into user memory using a `learn_user_memory` boolean flag in the API (`POST /v1/rooms/:room_id/messages`).
3. **Asynchronous Non-Blocking Execution**: Ingestion into user memory occurs asynchronously in background worker routines, ensuring chat message creation returns immediately with zero added latency.
4. **Dedicated User Memory API**: Dedicated `/v1/projects/:project_id/users/:external_user_id/memory` endpoints for fetching, updating, listing, and resetting user memory profiles.

---

## 2. API Specifications & Controls

### A. Message Ingestion API with Memory Learning Control

**`POST /v1/rooms/:room_id/messages`**
- **Request Body**:
  ```json
  {
    "role": "user",
    "content": "I prefer short Go code examples using standard library HTTP handlers.",
    "learn_user_memory": true
  }
  ```
- **Behavior**:
  - `learn_user_memory: true` (default): Queue task to extract user traits and update project user memory asynchronously.
  - `learn_user_memory: false`: Message is processed for room state and message history, but **skipped** for project user memory profiling.

---

### B. Dedicated User Memory API Endpoints

#### 1. Fetch User Profile
- **`GET /v1/projects/:project_id/users/:external_user_id/memory`**
- **Response**:
  ```json
  {
    "id": "uuid",
    "project_id": "b0000000-0000-0000-0000-000000000001",
    "external_user_id": "alice_123",
    "profile_summary": "Senior Go developer building cloud backend services. Prefers concise code snippets.",
    "speaking_style": "Direct, technical, prefers bullet points and standard library solutions",
    "personality_traits": ["analytical", "solution-oriented", "detail-focused"],
    "preferences": {
      "language": "Go",
      "tone": "concise",
      "library_preference": "stdlib"
    },
    "room_count": 4,
    "last_interaction_at": "2026-08-15T02:20:00Z",
    "created_at": "2026-08-10T10:00:00Z",
    "updated_at": "2026-08-15T02:20:00Z"
  }
  ```

#### 2. Manual Update Profile
- **`PUT /v1/projects/:project_id/users/:external_user_id/memory`**

#### 3. List Project User Profiles (Dashboard)
- **`GET /v1/projects/:project_id/user-memories?search=...&limit=20&offset=0`**

#### 4. Delete / Reset User Profile
- **`DELETE /v1/projects/:project_id/users/:external_user_id/memory`**

#### 5. Trigger Synthesis On-Demand
- **`POST /v1/projects/:project_id/users/:external_user_id/memory/synthesize`**

---

## 3. Asynchronous Sequence Diagram

```
actor Client as Client SDK / End User
participant API as Message API Handler
participant DB as Postgres DB
participant Queue as Async Task Queue
participant Worker as Async Worker Pool
participant LLM as OpenRouter LLM Service
participant UserMem as User Memory Repo

Client -> API: POST /v1/rooms/:room_id/messages (content, learn_user_memory: true)
API -> DB: Save Message
API -> Queue: Push task (ProjectID, RoomID, ExtUserID, LearnUserMemory)
API --> Client: 201 Created (Instant Response)

Queue -> Worker: Dequeue messageTask (Async background execution)
Worker -> DB: Fetch Room State & Conversation History
Worker -> LLM: Extract Room State & Summary

alt LearnUserMemory is true AND room has external_user_id
    Worker -> UserMem: Fetch existing user profile for (project_id, external_user_id)
    Worker -> LLM: Extract & accumulate user traits (speaking style, traits, preferences)
    Worker -> UserMem: Upsert updated project UserMemoryProfile
else LearnUserMemory is false
    Worker -> Worker: Skip User Memory profiling for this turn
end
```

---

## 4. Database Schema

### Migration: `backend/db/migrations/06_add_user_memories.sql`
```sql
CREATE TABLE IF NOT EXISTS user_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255) NOT NULL,
    profile_summary TEXT NOT NULL DEFAULT '',
    speaking_style TEXT NOT NULL DEFAULT '',
    personality_traits JSONB DEFAULT '[]'::jsonb NOT NULL,
    preferences JSONB DEFAULT '{}'::jsonb NOT NULL,
    room_count INT DEFAULT 0 NOT NULL,
    last_interaction_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (project_id, external_user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_memories_project_user ON user_memories(project_id, external_user_id);
```

---

## 5. Summary of System Components & Files

| Component | File Path | Description |
| :--- | :--- | :--- |
| **Migration** | `backend/db/migrations/06_add_user_memories.sql` | Create `user_memories` table |
| **Domain** | `backend/internal/domain/user_memory.go` | `UserMemoryProfile` & interfaces |
| **Message Handler** | `backend/internal/handler/message_handler.go` | Parse `learn_user_memory` flag in `POST /rooms/:room_id/messages` |
| **Message Usecase** | `backend/internal/usecase/message_usecase.go` | Pass `LearnUserMemory` in task queue & perform conditional async learning |
| **Repository** | `backend/internal/repository/user_memory_repo.go` | PostgreSQL profile persistence |
| **LLM Profiler** | `backend/internal/service/openrouter.go` | Extract & refine profile summary, speaking style, traits, preferences |
| **User Memory Handler** | `backend/internal/handler/user_memory_handler.go` | Dedicated User Memory REST API endpoints |
| **Frontend UI** | `frontend/src/pages/UserMemoriesPage.tsx` | Dashboard page to view, search, and edit user memories |

---

## 6. Verification & Test Plan

1. **Unit Tests**:
   - Write `backend/internal/usecase/user_memory_usecase_test.go` testing conditional learning (`learn_user_memory: true` vs `false`), profile updates, and deletion.
   - Run `go test ./...` in `backend/`.
2. **API Verification**:
   - Send `POST /rooms/:room_id/messages` with `"learn_user_memory": true` and verify profile growth via `GET /v1/projects/:project_id/users/:external_user_id/memory`.
   - Send message with `"learn_user_memory": false` and verify user profile remains unchanged.
