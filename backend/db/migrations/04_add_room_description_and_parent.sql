-- Migration 04: Add description and parent_room_id to rooms table for parent-child sub-agent orchestration
ALTER TABLE rooms
ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '' NOT NULL,
ADD COLUMN IF NOT EXISTS parent_room_id UUID REFERENCES rooms(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_rooms_parent_room_id ON rooms(parent_room_id);
