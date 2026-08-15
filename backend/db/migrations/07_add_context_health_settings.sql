-- Migration 07: Add Context Health Policy & Stale Memory Lifecycle Settings

-- 1. Add context_health_policy settings JSONB column to projects
ALTER TABLE projects 
ADD COLUMN IF NOT EXISTS context_health_policy JSONB DEFAULT '{
  "enabled": true,
  "mode": "balanced",
  "decay_half_life_days": 30,
  "max_entropy_threshold": 4.8,
  "turn_eval_interval": 50,
  "deduplication_threshold": 0.95,
  "e2ee_eval_location": "server_distance_only"
}'::jsonb NOT NULL;

-- 2. Extend memories table for status tracking, supersedence DAG, and entropy scores
ALTER TABLE memories
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' NOT NULL,
ADD COLUMN IF NOT EXISTS superseded_by_id UUID REFERENCES memories(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS shannon_entropy DOUBLE PRECISION DEFAULT 0.0 NOT NULL,
ADD COLUMN IF NOT EXISTS is_whitelisted BOOLEAN DEFAULT false NOT NULL;

-- Indexes for memory status filtering and DAG traversal
CREATE INDEX IF NOT EXISTS idx_memories_room_status ON memories(room_id, status);
CREATE INDEX IF NOT EXISTS idx_memories_superseded_by ON memories(superseded_by_id);

-- 3. Create memory_revisions audit table for supersedence DAG tracking and undo/restore operations
CREATE TABLE IF NOT EXISTS memory_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL, -- 'superseded', 'restored', 'syntax_flagged'
    previous_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memory_revisions_room ON memory_revisions(room_id);
CREATE INDEX IF NOT EXISTS idx_memory_revisions_memory ON memory_revisions(memory_id);
