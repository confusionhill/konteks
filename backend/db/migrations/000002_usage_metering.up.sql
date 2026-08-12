-- Usage metering table
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE SET NULL,
    tokens_used INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_usage_tenant_created ON usage_records(tenant_id, created_at DESC);
