-- Migration 000004: Add references and entities JSONB columns to room_state table
ALTER TABLE room_state 
    ADD COLUMN IF NOT EXISTS references JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS entities JSONB DEFAULT '[]'::jsonb;
