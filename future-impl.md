# Future Production Durability Implementation Notes

To guarantee 100% task execution durability (preventing any loss of background embedding generation or state updates if the server crashes or restarts), the following options are planned for a production deployment:

## Option A (Redis Queue)
- **Concept**: Offload background tasks from memory channels to a persistent Redis-backed task queue.
- **Tools**: Use a Go task scheduling library like [hibiken/asynq](https://github.com/hibiken/asynq) or [adjust/rmq](https://github.com/adjust/rmq) backed by Redis.
- **Workflow**:
  1. `CreateMessage` writes message synchronously to PostgreSQL.
  2. `CreateMessage` enqueues a background job payload to Redis.
  3. Bounded workers running on separate consumer instances pull jobs from Redis and execute state extraction/embeddings.
  4. Redis guarantees task delivery even if a worker node crashes mid-execution.

## Option B (Database Outbox Pattern)
- **Concept**: Commit the task logs to a relational table in the same database transaction as the message write.
- **Workflow**:
  1. Create a `context_tasks` table in PostgreSQL:
     ```sql
     CREATE TABLE context_tasks (
         id UUID PRIMARY KEY,
         room_id UUID NOT NULL,
         task_type TEXT NOT NULL,
         payload JSONB NOT NULL,
         status TEXT NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
         attempts INT DEFAULT 0,
         created_at TIMESTAMP DEFAULT NOW(),
         processed_at TIMESTAMP
     );
     ```
  2. Inside a database transaction:
     - Insert message into `messages` table.
     - Insert extraction task into `context_tasks` table.
  3. A background process (ticker/sweeper) polls the `context_tasks` table for `'pending'` tasks, updates status to `'processing'`, runs LLM extraction, and marks them `'completed'`.
  4. If a server crashes, pending tasks remain in the DB and are automatically picked up by the next active scheduler instance upon restart.

---

## Production Memory Optimization & Cold Storage Design

### 1. The Direct Message Embedding Problem in Production
In our current prototype, **every message** is embedded and stored in the vector database. In production, this is suboptimal for several reasons:
- **High Cost & Latency:** Generating embeddings via LLM APIs on every conversational turn is expensive and introduces overhead.
- **Noise Pollution:** Most conversational logs contain filler messages (e.g., *"Okay"*, *"Thanks"*, *"Hello"*). Embedding these creates semantic vectors of low information density.
- **Query Dilution:** Searching for relevant facts across thousands of raw conversation snippets causes vector search dilution (retrieving noisy matching messages instead of actual factual context).

#### Production Best Practice:
Instead of embedding raw messages, production memory systems (like Zep, Honcho, or Zilliz) use an asynchronous pipeline to:
1. **Classify Relevance:** Filter out filler conversational turns.
2. **Extract Facts:** Use LLM summary models to parse the message history and extract structured facts (e.g., *"User prefers aisle seats"*, *"User is traveling to Seattle on Aug 15"*).
3. **Embed Facts:** Only embed and index these *extracted facts* rather than raw chat turns.

---

### 2. Cold Storage Archival in Markdown format (`.md`)
To optimize database hosting costs, vector search performance, and PG index sizes, we can offload older, inactive rooms into a structured, human-readable **Markdown cold storage format** on AWS S3 or Google Cloud Storage.

#### The Archival Workflow:
1. **Inactivity Sweep:** A cron process flags rooms that have been inactive for more than `N` days (e.g., 30 days).
2. **Markdown Serialization:** The archiver serializes the room's entire context history into a single Markdown file (`room_<room_id>_archive.md`):

```markdown
# ContextKit Room Archive: {room_id}
- **Tenant ID**: {tenant_id}
- **External User**: {external_user_id}
- **Archived At**: {timestamp}

## Final Conversation Summary
{summary}

## Extracted Active Topic
{active_topic}

## Final Room State
```json
{
  "state_variables": { ... }
}
```

## Chronological Message Log
### Sequence 1 [User] (2026-08-13T01:00:00Z)
"Hello, I need to book a flight."

### Sequence 2 [Assistant] (2026-08-13T01:00:15Z)
"Sure! I can help you search for flights."
```

3. **Database Purge:** 
   - Write the Markdown file to AWS S3 Glacier or GCS Coldline.
   - Delete the room's raw message logs and embedding memory vectors from PostgreSQL/pgvector.
   - Retain only the room meta-row with status set to `'archived'` and a link to the S3 bucket path.
4. **Hydration on Revival:**
   - If the user/agent sends a new message to an `'archived'` room, the system retrieves the Markdown file from S3, parses the final summary and state variables, repopulates the active database context, and sets room status to `'active'`.
   - This "lazy loading" maintains sub-100ms response times for active users while keeping the database size minimal.
