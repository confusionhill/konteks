-- Add status and archive_s3_path columns to rooms table
ALTER TABLE rooms 
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' NOT NULL CHECK (status IN ('active', 'archived', 'archiving', 'hydrating')),
    ADD COLUMN IF NOT EXISTS archive_s3_path TEXT;

-- Add last_summarized_seq column to room_state table
ALTER TABLE room_state 
    ADD COLUMN IF NOT EXISTS last_summarized_seq BIGINT DEFAULT 0 NOT NULL;
