-- Cloudflare Vectorize is the single source of truth for embeddings.
-- Postgres no longer stores vector columns.
ALTER TABLE memories DROP COLUMN IF EXISTS embedding;
ALTER TABLE artifact_chunks DROP COLUMN IF EXISTS embedding;