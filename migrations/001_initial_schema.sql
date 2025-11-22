-- Unity Collaboration Platform - Initial Database Schema
-- PostgreSQL 15+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For fuzzy text search

-- =================================================================
-- HELPER FUNCTIONS
-- =================================================================

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- =================================================================
-- USERS & AUTHENTICATION
-- =================================================================

-- Users table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),

    -- OAuth fields
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),

    -- Security fields
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMPTZ,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(255),

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),

    -- Preferences
    preferences JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'),
    CONSTRAINT oauth_or_password CHECK (
        (oauth_provider IS NOT NULL AND oauth_id IS NOT NULL) OR
        password_hash IS NOT NULL
    )
);

-- Indexes for users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Trigger for users updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at)
    WHERE revoked_at IS NULL;

-- API Keys table
CREATE TABLE api_keys (
    api_key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(project_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    scopes TEXT[],
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(user_id)
    WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());

-- =================================================================
-- PROJECTS
-- =================================================================

-- Projects table
CREATE TABLE projects (
    project_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Settings
    settings JSONB DEFAULT '{}',
    unity_version VARCHAR(50),

    -- VCS Integration
    vcs_provider VARCHAR(50),
    vcs_url VARCHAR(500),
    vcs_branch VARCHAR(255),

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),

    -- Storage
    storage_used BIGINT DEFAULT 0,
    storage_limit BIGINT DEFAULT 10737418240, -- 10GB default

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ
);

CREATE INDEX idx_projects_owner_id ON projects(owner_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_created_at ON projects(created_at DESC);
CREATE INDEX idx_projects_last_activity ON projects(last_activity_at DESC NULLS LAST);

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Project members table
CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('owner', 'admin', 'developer', 'viewer')),
    permissions JSONB DEFAULT '{}',
    invited_by UUID REFERENCES users(user_id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_project_members_user_id ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(project_id, role);

-- =================================================================
-- COLLABORATION SESSIONS
-- =================================================================

-- Sessions table
CREATE TABLE sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Settings
    settings JSONB DEFAULT '{}',
    max_participants INT DEFAULT 10 CHECK (max_participants > 0 AND max_participants <= 100),
    is_public BOOLEAN DEFAULT FALSE,
    voice_enabled BOOLEAN DEFAULT TRUE,
    record_session BOOLEAN DEFAULT FALSE,

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'ended')),

    -- Scene state (snapshot)
    scene_data JSONB,

    -- Recording
    recording_url VARCHAR(500),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,

    CONSTRAINT valid_end_time CHECK (ended_at IS NULL OR ended_at > started_at)
);

CREATE INDEX idx_sessions_project_id ON sessions(project_id);
CREATE INDEX idx_sessions_owner_id ON sessions(owner_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX idx_sessions_active ON sessions(project_id, status, created_at DESC)
    WHERE status = 'active';

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Session participants table
CREATE TABLE session_participants (
    session_id UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('owner', 'editor', 'viewer')),
    status VARCHAR(20) DEFAULT 'online' CHECK (status IN ('online', 'away', 'offline')),

    -- Presence
    current_scene VARCHAR(255),
    selected_object VARCHAR(255),
    cursor_position JSONB,
    camera_transform JSONB,

    -- Voice
    voice_muted BOOLEAN DEFAULT FALSE,
    voice_deafened BOOLEAN DEFAULT FALSE,

    -- Timestamps
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (session_id, user_id),
    CONSTRAINT valid_leave_time CHECK (left_at IS NULL OR left_at > joined_at)
);

CREATE INDEX idx_session_participants_user_id ON session_participants(user_id);
CREATE INDEX idx_session_participants_status ON session_participants(session_id, status);
CREATE INDEX idx_session_participants_active ON session_participants(session_id)
    WHERE left_at IS NULL;

-- =================================================================
-- ASSETS
-- =================================================================

-- Assets table
CREATE TABLE assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE SET NULL,

    -- File info
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    content_type VARCHAR(100),
    md5_hash VARCHAR(32),

    -- Storage
    storage_path VARCHAR(500) NOT NULL,
    cdn_url VARCHAR(500),
    thumbnail_url VARCHAR(500),

    -- Metadata
    metadata JSONB DEFAULT '{}',
    category VARCHAR(50),
    tags TEXT[],

    -- Status
    status VARCHAR(20) DEFAULT 'uploading' CHECK (status IN ('uploading', 'processing', 'ready', 'failed', 'deleted')),
    error_message TEXT,

    -- Versioning
    version_number INT DEFAULT 1,
    parent_asset_id UUID REFERENCES assets(asset_id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT valid_file_size CHECK (file_size <= 5368709120) -- 5GB max
);

CREATE INDEX idx_assets_project_id ON assets(project_id);
CREATE INDEX idx_assets_user_id ON assets(user_id);
CREATE INDEX idx_assets_status ON assets(status);
CREATE INDEX idx_assets_category ON assets(project_id, category);
CREATE INDEX idx_assets_tags ON assets USING GIN(tags);
CREATE INDEX idx_assets_created_at ON assets(created_at DESC);
CREATE INDEX idx_assets_parent ON assets(parent_asset_id) WHERE parent_asset_id IS NOT NULL;

-- Full-text search index
CREATE INDEX idx_assets_search ON assets USING GIN(
    to_tsvector('english', file_name || ' ' || COALESCE(metadata->>'description', ''))
);

CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =================================================================
-- INVITATIONS
-- =================================================================

-- Invitations table
CREATE TABLE invitations (
    invitation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(project_id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(session_id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    message TEXT,
    token VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT project_or_session CHECK (
        (project_id IS NOT NULL AND session_id IS NULL) OR
        (project_id IS NULL AND session_id IS NOT NULL)
    ),
    CONSTRAINT valid_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_project ON invitations(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_invitations_session ON invitations(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_invitations_status ON invitations(status, expires_at);

-- =================================================================
-- WEBHOOKS
-- =================================================================

-- Webhooks table
CREATE TABLE webhooks (
    webhook_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    url VARCHAR(500) NOT NULL,
    events TEXT[] NOT NULL,
    secret VARCHAR(255),
    active BOOLEAN DEFAULT TRUE,

    -- Stats
    last_triggered_at TIMESTAMPTZ,
    total_deliveries INT DEFAULT 0,
    failed_deliveries INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_url CHECK (url ~* '^https?://.*')
);

CREATE INDEX idx_webhooks_project_id ON webhooks(project_id);
CREATE INDEX idx_webhooks_active ON webhooks(project_id, active) WHERE active = TRUE;

CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Webhook deliveries table (partitioned by month)
CREATE TABLE webhook_deliveries (
    delivery_id UUID DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL,
    event VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status_code INT,
    response_body TEXT,
    error_message TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (delivery_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);

-- Create initial partitions (current month and next 3 months)
CREATE TABLE webhook_deliveries_y2025m11 PARTITION OF webhook_deliveries
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE webhook_deliveries_y2025m12 PARTITION OF webhook_deliveries
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE webhook_deliveries_y2026m01 PARTITION OF webhook_deliveries
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE webhook_deliveries_y2026m02 PARTITION OF webhook_deliveries
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- =================================================================
-- AUDIT LOGS
-- =================================================================

-- Audit logs table (partitioned by month)
CREATE TABLE audit_logs (
    log_id UUID DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(project_id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(session_id) ON DELETE CASCADE,

    -- Action details
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),

    -- Request details
    ip_address INET,
    user_agent TEXT,

    -- Changes
    old_values JSONB,
    new_values JSONB,

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (log_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_project_id ON audit_logs(project_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Create initial partitions
CREATE TABLE audit_logs_y2025m11 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE audit_logs_y2025m12 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE audit_logs_y2026m01 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_logs_y2026m02 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- =================================================================
-- STATISTICS & MAINTENANCE
-- =================================================================

-- View for active sessions with participant count
CREATE VIEW v_active_sessions AS
SELECT
    s.session_id,
    s.project_id,
    s.owner_id,
    s.name,
    s.status,
    COUNT(sp.user_id) FILTER (WHERE sp.left_at IS NULL) as active_participants,
    s.max_participants,
    s.created_at,
    s.started_at
FROM sessions s
LEFT JOIN session_participants sp ON s.session_id = sp.session_id
WHERE s.status = 'active'
GROUP BY s.session_id;

-- View for project storage statistics
CREATE VIEW v_project_storage AS
SELECT
    p.project_id,
    p.name,
    p.storage_limit,
    COALESCE(SUM(a.file_size), 0) as storage_used,
    COUNT(a.asset_id) as asset_count,
    p.storage_limit - COALESCE(SUM(a.file_size), 0) as storage_available
FROM projects p
LEFT JOIN assets a ON p.project_id = a.project_id AND a.status = 'ready'
GROUP BY p.project_id;

-- =================================================================
-- INITIAL DATA
-- =================================================================

-- Create system user for automated operations
INSERT INTO users (user_id, email, name, password_hash, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'system@collab.internal',
    'System',
    'N/A',
    'active'
);

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dev;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO dev;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Unity Collaboration Platform schema initialized successfully!';
    RAISE NOTICE 'Database: collab_dev';
    RAISE NOTICE 'Tables created: %', (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE');
    RAISE NOTICE 'Indexes created: %', (SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public');
END $$;
