CREATE TABLE IF NOT EXISTS context_evaluations (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL,
    query TEXT NOT NULL,
    context_snapshot JSONB NOT NULL,
    relevance_score DOUBLE PRECISION NOT NULL,
    completeness_score DOUBLE PRECISION NOT NULL,
    token_efficiency_score DOUBLE PRECISION NOT NULL,
    reasoning TEXT NOT NULL,
    is_new_topic BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_context_evaluations_room_id ON context_evaluations(room_id);
