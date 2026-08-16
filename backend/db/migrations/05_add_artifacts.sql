-- Artifact Groups table (hierarchical folders / tags)
CREATE TABLE IF NOT EXISTS artifact_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(512) DEFAULT '/' NOT NULL,
    description TEXT DEFAULT '' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (project_id, name)
);
CREATE INDEX IF NOT EXISTS idx_artifact_groups_project ON artifact_groups(project_id);

-- Artifacts table (Core Markdown Memory metadata & S3 pointer)
CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    group_id UUID REFERENCES artifact_groups(id) ON DELETE SET NULL,
    file_path VARCHAR(512) NOT NULL,
    title VARCHAR(255) NOT NULL,
    s3_key VARCHAR(1024) DEFAULT '' NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (project_id, file_path)
);
CREATE INDEX IF NOT EXISTS idx_artifacts_project_path ON artifacts(project_id, file_path);
CREATE INDEX IF NOT EXISTS idx_artifacts_group_id ON artifacts(group_id);

-- Artifact Chunks table (Header-aware chunking & pgvector index)
CREATE TABLE IF NOT EXISTS artifact_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    section_heading VARCHAR(255) DEFAULT '' NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_artifact_chunks_artifact ON artifact_chunks(artifact_id);
CREATE INDEX IF NOT EXISTS idx_artifact_chunks_project ON artifact_chunks(project_id);

-- Artifact References table (Knowledge Graph Edges)
CREATE TABLE IF NOT EXISTS artifact_references (
    source_artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    target_artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) DEFAULT 'refers_to' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (source_artifact_id, target_artifact_id)
);
CREATE INDEX IF NOT EXISTS idx_artifact_references_target ON artifact_references(target_artifact_id);

-- Room Artifact Blacklists table
CREATE TABLE IF NOT EXISTS room_artifact_blacklists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE,
    group_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT check_blacklist_target CHECK (artifact_id IS NOT NULL OR group_name IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_room_blacklists_room ON room_artifact_blacklists(room_id);
