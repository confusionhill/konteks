-- Enable vector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Projects table (scraped tenants -> projects with safe soft-delete)
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '' NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_owner_deleted ON projects(owner_id, deleted_at);

-- Project Members table (future-proofing project user invitations & permissions)
CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);

-- Project Invitations table (future-proofing pending user invitations)
CREATE TABLE IF NOT EXISTS project_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member', 'viewer')),
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'revoked')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_project_invitations_token ON project_invitations(token);
CREATE INDEX IF NOT EXISTS idx_project_invitations_project_email ON project_invitations(project_id, email);

-- API Keys table (bound per project)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_project_id ON api_keys(project_id);

-- Rooms table (bound to project)
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255),
    state_schema JSONB,
    default_policy JSONB,
    status VARCHAR(20) DEFAULT 'active' NOT NULL CHECK (status IN ('active', 'archived', 'archiving', 'hydrating')),
    archive_s3_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rooms_project_id ON rooms(project_id);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    tokens_input INT DEFAULT NULL,
    tokens_output INT DEFAULT NULL,
    cost_total NUMERIC(12, 6) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_room_sequence ON messages(room_id, sequence);

-- Room State table
CREATE TABLE IF NOT EXISTS room_state (
    room_id UUID PRIMARY KEY REFERENCES rooms(id) ON DELETE CASCADE,
    summary TEXT NOT NULL DEFAULT '',
    active_topic VARCHAR(255) NOT NULL DEFAULT '',
    state JSONB DEFAULT '{}'::jsonb NOT NULL,
    "references" JSONB DEFAULT '{}'::jsonb,
    "entities" JSONB DEFAULT '[]'::jsonb,
    last_summarized_seq BIGINT DEFAULT 0 NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Memories table
CREATE TABLE IF NOT EXISTS memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255),
    content TEXT NOT NULL,
    embedding vector(1536) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memories_room_id ON memories(room_id);
CREATE INDEX IF NOT EXISTS idx_memories_project_user ON memories(project_id, external_user_id);

-- Room Events table
CREATE TABLE IF NOT EXISTS room_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_room_events_room_created ON room_events(room_id, created_at DESC);

-- Usage Records table
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    endpoint VARCHAR(100) NOT NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE SET NULL,
    tokens_used INT NOT NULL DEFAULT 0,
    tokens_input INT NOT NULL DEFAULT 0,
    tokens_output INT NOT NULL DEFAULT 0,
    cost_input NUMERIC(12, 6) NOT NULL DEFAULT 0,
    cost_output NUMERIC(12, 6) NOT NULL DEFAULT 0,
    cost_total NUMERIC(12, 6) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_usage_project_created ON usage_records(project_id, created_at DESC);

-- Context Evaluations table
CREATE TABLE IF NOT EXISTS context_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_context_evaluations_project_id ON context_evaluations(project_id);

-- Seed Initial Test Admin User (email: admin@admin.com | password: admin123)
-- Bcrypt hash for "admin123": $2a$10$DIk22SIkufr3xFaumPQtHu5diNVade5xIvsvMY5vmsRCeKKSFJ/Ja
INSERT INTO users (id, email, password_hash, name)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin@admin.com',
    '$2a$10$DIk22SIkufr3xFaumPQtHu5diNVade5xIvsvMY5vmsRCeKKSFJ/Ja',
    'Admin User'
) ON CONFLICT (email) DO NOTHING;

-- Seed Initial Demo Project for Admin User
INSERT INTO projects (id, name, description, owner_id)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'Demo AI Assistant Project',
    'Default project for local testing and SDK integration',
    'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (id) DO NOTHING;

-- Seed Project Member for Owner
INSERT INTO project_members (project_id, user_id, role)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'owner'
) ON CONFLICT (project_id, user_id) DO NOTHING;

-- Seed Initial API Key for Demo Project (raw key: ck_live_dev_test_key_123)
-- SHA-256 hash for "ck_live_dev_test_key_123": 2ec2b322a33f5d5e236561f7e025176c1dfb194d33458ef1df1a545300d8b4e7
INSERT INTO api_keys (id, project_id, key_hash, key_prefix, name)
VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    '2ec2b322a33f5d5e236561f7e025176c1dfb194d33458ef1df1a545300d8b4e7',
    'ck_live_dev...',
    'Development Test Key'
) ON CONFLICT (id) DO NOTHING;
