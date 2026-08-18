-- Migration 10: Track artifact chunk embedding status.
-- Chunks whose embedding generation failed are flagged 'pending' so a
-- background worker can retry embedding them. Chunks successfully pushed
-- to the vector store (Cloudflare Vectorize) are marked 'indexed'.
ALTER TABLE artifact_chunks ADD COLUMN IF NOT EXISTS vector_status VARCHAR(16) NOT NULL DEFAULT 'indexed';
CREATE INDEX IF NOT EXISTS idx_artifact_chunks_vector_status ON artifact_chunks(vector_status);