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
