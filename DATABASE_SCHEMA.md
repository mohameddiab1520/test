# Unity Collaboration Platform - Database Schema

## Table of Contents

1. [Overview](#overview)
2. [PostgreSQL Schema (Main Database)](#postgresql-schema-main-database)
3. [Redis Schema (Cache & Sessions)](#redis-schema-cache--sessions)
4. [MongoDB Schema (Logs & Analytics)](#mongodb-schema-logs--analytics)
5. [TimescaleDB Schema (Metrics)](#timescaledb-schema-metrics)
6. [Migrations](#migrations)
7. [Indexes & Performance](#indexes--performance)

---

## Overview

### Database Strategy

| Database | Purpose | Data Types |
|----------|---------|------------|
| **PostgreSQL** | Primary relational data | Users, Projects, Sessions, Assets metadata |
| **Redis** | Cache & real-time sessions | Session cache, WebSocket connections, rate limiting |
| **MongoDB** | Document storage | Operation logs, analytics, audit trails |
| **TimescaleDB** | Time-series metrics | Performance metrics, usage analytics |
| **S3/MinIO** | Object storage | Asset files, backups |

---

## PostgreSQL Schema (Main Database)

### Users Table

```sql
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

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Refresh Tokens Table

```sql
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
```

### Projects Table

```sql
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
```

### Project Members Table

```sql
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
```

### Sessions Table

```sql
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
```

### Session Participants Table

```sql
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
```

### Assets Table

```sql
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

-- Full-text search
CREATE INDEX idx_assets_search ON assets USING GIN(
    to_tsvector('english', file_name || ' ' || COALESCE(metadata->>'description', ''))
);

CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Invitations Table

```sql
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
```

### Webhooks Table

```sql
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
```

### Webhook Deliveries Table

```sql
CREATE TABLE webhook_deliveries (
    delivery_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(webhook_id) ON DELETE CASCADE,
    event VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status_code INT,
    response_body TEXT,
    error_message TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);

-- Partition by month for efficient querying and archival
CREATE TABLE webhook_deliveries_y2025m11 PARTITION OF webhook_deliveries
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
```

### API Keys Table

```sql
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
```

### Audit Logs Table

```sql
CREATE TABLE audit_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_project_id ON audit_logs(project_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Partition by month
CREATE TABLE audit_logs_y2025m11 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
```

---

## Redis Schema (Cache & Sessions)

### Key Naming Convention

```
{namespace}:{entity}:{id}:{field}
```

### User Session Cache

```
Key: session:user:{user_id}
Type: String (JSON)
TTL: 3600 seconds (1 hour)

Value:
{
  "userId": "user_abc123",
  "email": "user@example.com",
  "name": "John Doe",
  "permissions": ["read:projects", "write:projects"]
}
```

### Collaboration Session Cache

```
Key: session:collab:{session_id}
Type: String (JSON)
TTL: 7200 seconds (2 hours)

Value:
{
  "sessionId": "sess_abc123",
  "projectId": "proj_123",
  "participants": ["user_1", "user_2"],
  "sceneData": {...},
  "version": 42
}
```

### WebSocket Connection Tracking

```
Key: ws:connections:{session_id}
Type: Hash
TTL: No expiry (deleted on disconnect)

Fields:
  user_1 -> connection_id_1
  user_2 -> connection_id_2
```

### Operation Queue (Per Session)

```
Key: ops:queue:{session_id}
Type: List
TTL: 3600 seconds

Values:
[
  "{operation_json_1}",
  "{operation_json_2}",
  ...
]
```

### Rate Limiting

```
Key: ratelimit:{user_id}:{endpoint}
Type: String (counter)
TTL: 60 seconds (sliding window)

Value: request_count
```

### Presence Tracking

```
Key: presence:{session_id}:{user_id}
Type: String (JSON)
TTL: 300 seconds (5 minutes, refreshed on heartbeat)

Value:
{
  "status": "online",
  "currentScene": "MainScene",
  "selectedObject": "Player",
  "lastActivity": 1700650000
}
```

### Cache Invalidation Patterns

```
Key: cache:invalidate:{resource_type}:{resource_id}
Type: Pub/Sub Channel

Message:
{
  "type": "invalidate",
  "resource": "session",
  "id": "sess_abc123",
  "timestamp": 1700650000
}
```

---

## MongoDB Schema (Logs & Analytics)

### Operations Collection

```javascript
// Collection: operations
{
  _id: ObjectId("..."),
  sessionId: "sess_abc123",
  userId: "user_xyz",
  operation: {
    type: "update",
    objectId: "obj_player_001",
    path: "transform.position.x",
    oldValue: 5.0,
    newValue: 10.5,
    version: 42
  },
  metadata: {
    userAgent: "Unity/2022.3.15f1",
    platform: "Windows",
    ipAddress: "192.168.1.1"
  },
  timestamp: ISODate("2025-11-22T10:00:00Z"),
  processingTime: 15 // milliseconds
}

// Indexes
db.operations.createIndex({ sessionId: 1, timestamp: -1 });
db.operations.createIndex({ userId: 1, timestamp: -1 });
db.operations.createIndex({ timestamp: -1 });
db.operations.createIndex({ "operation.objectId": 1 });

// TTL Index (auto-delete after 30 days)
db.operations.createIndex({ timestamp: 1 }, { expireAfterSeconds: 2592000 });
```

### Session Analytics Collection

```javascript
// Collection: session_analytics
{
  _id: ObjectId("..."),
  sessionId: "sess_abc123",
  projectId: "proj_123",

  // Summary
  startTime: ISODate("2025-11-22T10:00:00Z"),
  endTime: ISODate("2025-11-22T12:00:00Z"),
  duration: 7200, // seconds

  // Participants
  participants: [
    {
      userId: "user_1",
      role: "owner",
      joinedAt: ISODate("2025-11-22T10:00:00Z"),
      leftAt: ISODate("2025-11-22T12:00:00Z"),
      operationCount: 650,
      activeTime: 7000 // seconds
    }
  ],

  // Operations
  totalOperations: 1523,
  operationsByType: {
    create: 23,
    update: 1450,
    delete: 50
  },

  // Objects
  objectsCreated: 23,
  objectsModified: 145,
  objectsDeleted: 50,

  // Performance
  avgLatency: 45, // milliseconds
  p95Latency: 120,
  p99Latency: 250,

  // Metadata
  createdAt: ISODate("2025-11-22T12:00:00Z")
}

// Indexes
db.session_analytics.createIndex({ sessionId: 1 });
db.session_analytics.createIndex({ projectId: 1, startTime: -1 });
db.session_analytics.createIndex({ startTime: -1 });
```

### Asset Analytics Collection

```javascript
// Collection: asset_analytics
{
  _id: ObjectId("..."),
  assetId: "asset_abc123",
  projectId: "proj_123",

  // Usage
  downloadCount: 45,
  viewCount: 120,

  // Downloads by user
  downloadsByUser: {
    "user_1": 10,
    "user_2": 5
  },

  // Time series data
  dailyStats: [
    {
      date: ISODate("2025-11-22T00:00:00Z"),
      downloads: 15,
      views: 40
    }
  ],

  // Metadata
  lastDownloadedAt: ISODate("2025-11-22T10:00:00Z"),
  updatedAt: ISODate("2025-11-22T10:00:00Z")
}

// Indexes
db.asset_analytics.createIndex({ assetId: 1 });
db.asset_analytics.createIndex({ projectId: 1 });
```

---

## TimescaleDB Schema (Metrics)

### System Metrics Hypertable

```sql
CREATE TABLE system_metrics (
    time TIMESTAMPTZ NOT NULL,
    service VARCHAR(50) NOT NULL,
    instance VARCHAR(100) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    tags JSONB DEFAULT '{}',

    PRIMARY KEY (time, service, instance, metric_name)
);

-- Convert to hypertable
SELECT create_hypertable('system_metrics', 'time');

-- Create indexes
CREATE INDEX idx_system_metrics_service ON system_metrics(service, time DESC);
CREATE INDEX idx_system_metrics_metric ON system_metrics(metric_name, time DESC);

-- Compression policy (compress data older than 7 days)
ALTER TABLE system_metrics SET (
  timescaledb.compress,
  timescaledb.compress_segmentby = 'service,instance,metric_name'
);

SELECT add_compression_policy('system_metrics', INTERVAL '7 days');

-- Retention policy (delete data older than 90 days)
SELECT add_retention_policy('system_metrics', INTERVAL '90 days');
```

### API Metrics Hypertable

```sql
CREATE TABLE api_metrics (
    time TIMESTAMPTZ NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INT NOT NULL,
    response_time_ms INT NOT NULL,
    user_id UUID,
    ip_address INET,

    PRIMARY KEY (time, endpoint, method)
);

SELECT create_hypertable('api_metrics', 'time');

CREATE INDEX idx_api_metrics_endpoint ON api_metrics(endpoint, time DESC);
CREATE INDEX idx_api_metrics_user ON api_metrics(user_id, time DESC) WHERE user_id IS NOT NULL;
CREATE INDEX idx_api_metrics_status ON api_metrics(status_code, time DESC);
```

### Continuous Aggregates (Pre-computed Views)

```sql
-- Hourly aggregates
CREATE MATERIALIZED VIEW api_metrics_hourly
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 hour', time) AS bucket,
  endpoint,
  method,
  COUNT(*) AS request_count,
  AVG(response_time_ms) AS avg_response_time,
  percentile_cont(0.95) WITHIN GROUP (ORDER BY response_time_ms) AS p95_response_time,
  percentile_cont(0.99) WITHIN GROUP (ORDER BY response_time_ms) AS p99_response_time,
  COUNT(*) FILTER (WHERE status_code >= 500) AS error_count
FROM api_metrics
GROUP BY bucket, endpoint, method;

-- Refresh policy
SELECT add_continuous_aggregate_policy('api_metrics_hourly',
  start_offset => INTERVAL '3 hours',
  end_offset => INTERVAL '1 hour',
  schedule_interval => INTERVAL '1 hour');
```

---

## Migrations

### Migration Tool: golang-migrate

```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create migration
migrate create -ext sql -dir migrations -seq create_users_table

# Run migrations
migrate -database postgres://user:pass@localhost:5432/dbname?sslmode=disable \
        -path migrations up

# Rollback
migrate -database postgres://user:pass@localhost:5432/dbname?sslmode=disable \
        -path migrations down 1
```

### Migration Example

**migrations/000001_create_users_table.up.sql**
```sql
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

**migrations/000001_create_users_table.down.sql**
```sql
DROP TABLE IF EXISTS users;
```

---

## Indexes & Performance

### Index Strategy

1. **Primary Keys**: Always UUID with B-tree index
2. **Foreign Keys**: Index all foreign key columns
3. **Query Patterns**: Index columns used in WHERE, JOIN, ORDER BY
4. **Composite Indexes**: For multi-column queries
5. **Partial Indexes**: For filtered queries (e.g., WHERE status = 'active')
6. **GIN Indexes**: For JSONB and array columns
7. **Full-Text Search**: GIN indexes on tsvector

### Query Optimization Examples

**Before (Slow)**
```sql
SELECT * FROM sessions WHERE project_id = 'proj_123' AND status = 'active';
-- Sequential scan on 1M rows
```

**After (Fast)**
```sql
CREATE INDEX idx_sessions_active ON sessions(project_id, status, created_at DESC)
WHERE status = 'active';

SELECT * FROM sessions WHERE project_id = 'proj_123' AND status = 'active';
-- Index scan on 100 rows
```

### Connection Pooling

```yaml
# PgBouncer Configuration
[databases]
collab_db = host=localhost port=5432 dbname=collab

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
min_pool_size = 10
reserve_pool_size = 5
reserve_pool_timeout = 3
max_db_connections = 100
max_user_connections = 100
```

### Partitioning Strategy

**Time-based Partitioning**
```sql
-- Partition audit_logs by month
CREATE TABLE audit_logs (
    log_id UUID DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL,
    ...
) PARTITION BY RANGE (created_at);

CREATE TABLE audit_logs_y2025m11 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

CREATE TABLE audit_logs_y2025m12 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
```

### Vacuum & Maintenance

```sql
-- Auto-vacuum settings
ALTER TABLE sessions SET (
  autovacuum_vacuum_scale_factor = 0.1,
  autovacuum_analyze_scale_factor = 0.05
);

-- Manual vacuum (run during low traffic)
VACUUM ANALYZE sessions;

-- Reindex (if needed)
REINDEX TABLE sessions;
```

---

## Backup & Recovery

### Backup Strategy

**Daily Full Backup**
```bash
pg_dump -Fc -Z9 -h localhost -U postgres collab_db > backup_$(date +%Y%m%d).dump
```

**Continuous Archiving (WAL)**
```
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'
```

**Point-in-Time Recovery**
```bash
# Restore base backup
pg_restore -d collab_db backup_20251122.dump

# Replay WAL files up to specific time
recovery_target_time = '2025-11-22 12:00:00'
```

---

This database schema provides a comprehensive, scalable foundation for the Unity Collaboration Platform with proper indexing, partitioning, and performance optimization strategies.
