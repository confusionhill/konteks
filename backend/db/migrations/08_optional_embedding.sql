-- Migration 08: Make embedding column nullable in memories and artifact_chunks
-- Required for production deployment offloading vector storage to Cloudflare Vectorize.

ALTER TABLE memories ALTER COLUMN embedding DROP NOT NULL;
ALTER TABLE artifact_chunks ALTER COLUMN embedding DROP NOT NULL;
