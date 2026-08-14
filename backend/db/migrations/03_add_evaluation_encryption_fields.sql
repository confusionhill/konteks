-- Add encryption columns to context_evaluations table
ALTER TABLE context_evaluations ADD COLUMN IF NOT EXISTS is_encrypted BOOLEAN DEFAULT FALSE;
ALTER TABLE context_evaluations ADD COLUMN IF NOT EXISTS encryption_metadata JSONB DEFAULT NULL;
