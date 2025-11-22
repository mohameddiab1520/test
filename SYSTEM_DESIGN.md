# Unity Collaboration Platform - System Design

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Service Design](#service-design)
3. [Data Flow](#data-flow)
4. [Communication Protocols](#communication-protocols)
5. [State Management](#state-management)
6. [Error Handling](#error-handling)
7. [Performance Requirements](#performance-requirements)

---

## System Requirements

### Functional Requirements

#### Core Features
1. **Real-Time Collaboration**
   - Multiple users editing the same Unity scene simultaneously
   - Sub-100ms latency for changes to propagate
   - Visual indicators of other users' cursors and selections
   - Lock mechanism for exclusive editing when needed

2. **Asset Management**
   - Upload/download Unity assets
   - Version control for assets
   - Asset preview and thumbnails
   - Asset search and filtering
   - Support for large files (up to 5GB)

3. **Session Management**
   - Create collaborative sessions
   - Invite users to sessions
   - Role-based permissions (Owner, Editor, Viewer)
   - Session recording and playback

4. **Communication**
   - Voice chat with spatial audio
   - Text chat with mentions and threads
   - Screen sharing
   - Presence indicators

5. **Conflict Resolution**
   - Automatic merge for non-conflicting changes
   - Manual resolution UI for conflicts
   - Change history and rollback

6. **Integration**
   - Git/Perforce integration
   - CI/CD pipeline integration
   - Third-party authentication (Google, GitHub, Unity ID)

### Non-Functional Requirements

#### Performance
- **Latency**: < 100ms for sync operations
- **Throughput**: 100,000+ messages/second
- **Concurrent Users**: 10,000+ per cluster
- **Active Sessions**: 1,000+ simultaneous
- **Asset Upload**: 1GB/minute per user
- **API Response**: < 200ms for 95th percentile

#### Scalability
- Horizontal scaling for all services
- Support for 1M+ registered users
- Handle 100,000+ concurrent connections
- Auto-scaling based on load

#### Availability
- **Uptime**: 99.9% SLA (< 43 minutes downtime/month)
- **Multi-region deployment**: Active-active
- **Disaster recovery**: RTO < 1 hour, RPO < 5 minutes

#### Security
- TLS 1.3 for all communications
- OAuth 2.0 authentication
- JWT-based authorization
- End-to-end encryption for sensitive data
- GDPR and SOC 2 compliance

#### Reliability
- Circuit breakers for fault tolerance
- Retry mechanisms with exponential backoff
- Graceful degradation
- Data replication across regions

---

## Service Design

### 1. Session Service

#### Responsibilities
- Manage collaboration session lifecycle
- Handle participant management
- Track session state
- Enforce permissions

#### API Design

**Create Session**
```http
POST /api/v1/sessions
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "name": "Main Scene Collaboration",
  "projectId": "proj_123456",
  "settings": {
    "maxParticipants": 10,
    "isPublic": false,
    "voiceEnabled": true,
    "recordSession": false
  }
}

Response 201:
{
  "sessionId": "sess_abc123",
  "name": "Main Scene Collaboration",
  "projectId": "proj_123456",
  "ownerId": "user_xyz",
  "participants": [],
  "settings": {...},
  "status": "active",
  "createdAt": "2025-11-22T10:00:00Z",
  "wsUrl": "wss://collab.example.com/ws/sess_abc123"
}
```

**Join Session**
```http
POST /api/v1/sessions/{sessionId}/join
Authorization: Bearer <jwt_token>

Response 200:
{
  "success": true,
  "sessionData": {
    "participants": [...],
    "sceneState": {...},
    "wsToken": "eyJhbGc..."
  }
}
```

#### State Machine

```
┌─────────┐
│ Created │
└────┬────┘
     │ start()
     ▼
┌─────────┐  pause()  ┌────────┐
│ Active  │ ────────> │ Paused │
└────┬────┘           └───┬────┘
     │                    │ resume()
     │ <──────────────────┘
     │ end()
     ▼
┌─────────┐
│  Ended  │
└─────────┘
```

#### Database Schema

```sql
CREATE TABLE sessions (
  session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(project_id),
  owner_id UUID NOT NULL REFERENCES users(user_id),
  name VARCHAR(255) NOT NULL,
  settings JSONB NOT NULL DEFAULT '{}',
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ,

  INDEX idx_project_id (project_id),
  INDEX idx_owner_id (owner_id),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at)
);

CREATE TABLE session_participants (
  session_id UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(user_id),
  role VARCHAR(20) NOT NULL DEFAULT 'viewer',
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at TIMESTAMPTZ,

  PRIMARY KEY (session_id, user_id),
  INDEX idx_user_id (user_id)
);
```

#### Implementation (Go)

```go
package session

type Service struct {
    db    *sql.DB
    cache *redis.Client
    pubsub MessageBus
}

type Session struct {
    SessionID    string          `json:"sessionId"`
    ProjectID    string          `json:"projectId"`
    OwnerID      string          `json:"ownerId"`
    Name         string          `json:"name"`
    Settings     SessionSettings `json:"settings"`
    Status       SessionStatus   `json:"status"`
    CreatedAt    time.Time       `json:"createdAt"`
    UpdatedAt    time.Time       `json:"updatedAt"`
}

func (s *Service) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    // Create session in database
    session := &Session{
        SessionID: uuid.New().String(),
        ProjectID: req.ProjectID,
        OwnerID:   req.UserID,
        Name:      req.Name,
        Settings:  req.Settings,
        Status:    StatusActive,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := s.db.CreateSession(ctx, session); err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }

    // Cache session data
    if err := s.cache.Set(ctx, s.cacheKey(session.SessionID), session, 1*time.Hour); err != nil {
        log.Warn("failed to cache session", "error", err)
    }

    // Publish session created event
    s.pubsub.Publish("sessions.created", session)

    return session, nil
}

func (s *Service) JoinSession(ctx context.Context, sessionID, userID string) error {
    // Get session from cache or DB
    session, err := s.GetSession(ctx, sessionID)
    if err != nil {
        return err
    }

    // Check permissions
    if !session.CanJoin(userID) {
        return ErrPermissionDenied
    }

    // Add participant
    participant := &SessionParticipant{
        SessionID: sessionID,
        UserID:    userID,
        Role:      RoleViewer,
        JoinedAt:  time.Now(),
    }

    if err := s.db.AddParticipant(ctx, participant); err != nil {
        return err
    }

    // Publish participant joined event
    s.pubsub.Publish("sessions.participant_joined", participant)

    return nil
}
```

---

### 2. Sync Service (Real-Time Synchronization)

#### Responsibilities
- Handle real-time scene synchronization
- Apply Operational Transform (OT) algorithm
- Broadcast changes to participants
- Maintain operation history

#### Operational Transform Algorithm

**Concept**: Transform concurrent operations so they can be applied in any order while maintaining consistency.

**Example**:
```
Initial state: "Hello"

User A: Insert "World" at position 5 → "HelloWorld"
User B: Delete characters 0-4 → "o"

Without OT: Conflicts!
With OT: Transform B's operation considering A's operation
Result: "WorldHello" or "oWorld" (depending on transformation)
```

#### Implementation Strategy

```javascript
// Operation types
const OpType = {
  CREATE: 'create',
  UPDATE: 'update',
  DELETE: 'delete',
  TRANSFORM: 'transform'
};

// Operation structure
class Operation {
  constructor(type, objectId, path, value, version) {
    this.id = uuidv4();
    this.type = type;
    this.objectId = objectId;
    this.path = path;        // e.g., "transform.position.x"
    this.value = value;
    this.version = version;  // Vector clock
    this.userId = null;
    this.timestamp = Date.now();
  }
}

// Transformation function
function transform(op1, op2) {
  // If operations are on different objects, no transformation needed
  if (op1.objectId !== op2.objectId) {
    return [op1, op2];
  }

  // If operations are on different paths, no transformation needed
  if (op1.path !== op2.path) {
    return [op1, op2];
  }

  // Same object, same path - need to resolve conflict
  if (op1.type === 'update' && op2.type === 'update') {
    // Last write wins based on timestamp
    if (op1.timestamp > op2.timestamp) {
      return [op1, null]; // op2 is dropped
    } else {
      return [null, op2]; // op1 is dropped
    }
  }

  // Handle create/delete conflicts
  if (op1.type === 'delete' && op2.type === 'update') {
    return [op1, null]; // Delete wins
  }

  if (op1.type === 'update' && op2.type === 'delete') {
    return [null, op2]; // Delete wins
  }

  return [op1, op2];
}

// Sync engine
class SyncEngine {
  constructor(sessionId) {
    this.sessionId = sessionId;
    this.operations = [];      // Operation history
    this.clients = new Map();  // Connected clients
    this.version = 0;          // Global version counter
  }

  // Apply operation from client
  async applyOperation(userId, operation) {
    // Validate operation
    if (operation.version !== this.version) {
      // Client is behind, need to send missing operations
      const missingOps = this.operations.slice(operation.version);
      return { success: false, missingOps };
    }

    // Transform against concurrent operations
    let transformedOp = operation;
    for (const pendingOp of this.getPendingOperations(operation.timestamp)) {
      const [op1, op2] = transform(transformedOp, pendingOp);
      transformedOp = op1;
      if (!transformedOp) {
        // Operation was dropped due to conflict
        return { success: false, reason: 'conflict_resolved' };
      }
    }

    // Apply operation
    transformedOp.userId = userId;
    transformedOp.version = ++this.version;
    this.operations.push(transformedOp);

    // Broadcast to all clients except sender
    this.broadcast(transformedOp, userId);

    return { success: true, version: this.version };
  }

  broadcast(operation, excludeUserId) {
    const message = {
      type: 'sync',
      operation: operation
    };

    for (const [userId, client] of this.clients) {
      if (userId !== excludeUserId) {
        client.send(JSON.stringify(message));
      }
    }
  }

  getPendingOperations(sinceTimestamp) {
    return this.operations.filter(op => op.timestamp >= sinceTimestamp);
  }
}
```

#### WebSocket Protocol

**Client → Server**
```json
{
  "type": "operation",
  "sessionId": "sess_abc123",
  "operation": {
    "type": "update",
    "objectId": "obj_player_001",
    "path": "transform.position.x",
    "value": 10.5,
    "version": 42,
    "timestamp": 1700650000000
  }
}
```

**Server → Client**
```json
{
  "type": "sync",
  "operation": {
    "id": "op_xyz789",
    "type": "update",
    "objectId": "obj_player_001",
    "path": "transform.position.x",
    "value": 10.5,
    "version": 43,
    "userId": "user_abc",
    "timestamp": 1700650000100
  }
}
```

**Server → Client (Acknowledgment)**
```json
{
  "type": "ack",
  "success": true,
  "version": 43
}
```

**Server → Client (Sync Required)**
```json
{
  "type": "sync_required",
  "currentVersion": 50,
  "yourVersion": 42,
  "missingOperations": [...]
}
```

#### Go Implementation

```go
package sync

type SyncService struct {
    sessions map[string]*SyncSession
    mu       sync.RWMutex
}

type SyncSession struct {
    sessionID  string
    clients    map[string]*Client
    operations []Operation
    version    int64
    mu         sync.RWMutex
}

type Client struct {
    userID string
    conn   *websocket.Conn
    send   chan []byte
}

func (s *SyncService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Error("WebSocket upgrade failed", "error", err)
        return
    }
    defer conn.Close()

    // Authenticate and get session
    sessionID, userID, err := s.authenticate(r)
    if err != nil {
        conn.WriteJSON(map[string]string{"error": "authentication failed"})
        return
    }

    // Get or create sync session
    session := s.getOrCreateSession(sessionID)

    // Register client
    client := &Client{
        userID: userID,
        conn:   conn,
        send:   make(chan []byte, 256),
    }
    session.registerClient(client)
    defer session.unregisterClient(client)

    // Start goroutines
    go client.writePump()
    client.readPump(session)
}

func (c *Client) readPump(session *SyncSession) {
    defer c.conn.Close()

    for {
        var msg Message
        if err := c.conn.ReadJSON(&msg); err != nil {
            log.Error("Read error", "error", err)
            break
        }

        switch msg.Type {
        case "operation":
            session.handleOperation(c.userID, msg.Operation)
        case "ping":
            c.send <- []byte(`{"type":"pong"}`)
        }
    }
}

func (s *SyncSession) handleOperation(userID string, op Operation) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Validate version
    if op.Version != s.version {
        // Client is behind
        s.sendSyncRequired(userID)
        return
    }

    // Apply transformation
    transformedOp := op
    for _, pendingOp := range s.getPendingOperations(op.Timestamp) {
        transformedOp = s.transform(transformedOp, pendingOp)
        if transformedOp == nil {
            // Conflict resolved by dropping operation
            s.sendAck(userID, false, "conflict")
            return
        }
    }

    // Update version and store
    s.version++
    transformedOp.Version = s.version
    transformedOp.UserID = userID
    s.operations = append(s.operations, transformedOp)

    // Broadcast to all except sender
    s.broadcast(transformedOp, userID)

    // Send acknowledgment
    s.sendAck(userID, true, "")
}

func (s *SyncSession) broadcast(op Operation, excludeUserID string) {
    message, _ := json.Marshal(map[string]interface{}{
        "type":      "sync",
        "operation": op,
    })

    for userID, client := range s.clients {
        if userID != excludeUserID {
            select {
            case client.send <- message:
            default:
                // Client send buffer full, close connection
                close(client.send)
                delete(s.clients, userID)
            }
        }
    }
}
```

---

### 3. Asset Service

#### Responsibilities
- Handle asset uploads and downloads
- Generate thumbnails and previews
- Manage asset metadata
- Integrate with CDN for distribution

#### Upload Flow

```
1. Client requests upload URL
   POST /api/v1/assets/upload-url
   → Server generates presigned S3 URL

2. Client uploads directly to S3
   PUT <presigned-url>
   → Upload completes

3. Client confirms upload
   POST /api/v1/assets/confirm
   → Server processes asset (thumbnail, metadata extraction)

4. Asset becomes available
   GET /api/v1/assets/:id
```

#### API Design

**Request Upload URL**
```http
POST /api/v1/assets/upload-url
Authorization: Bearer <jwt_token>

{
  "fileName": "PlayerModel.fbx",
  "fileSize": 52428800,
  "contentType": "application/octet-stream",
  "projectId": "proj_123"
}

Response 200:
{
  "assetId": "asset_abc123",
  "uploadUrl": "https://s3.amazonaws.com/bucket/asset_abc123?X-Amz-...",
  "expiresIn": 3600
}
```

**Confirm Upload**
```http
POST /api/v1/assets/:assetId/confirm
Authorization: Bearer <jwt_token>

{
  "md5": "5d41402abc4b2a76b9719d911017c592"
}

Response 200:
{
  "success": true,
  "asset": {
    "assetId": "asset_abc123",
    "fileName": "PlayerModel.fbx",
    "fileSize": 52428800,
    "status": "processing",
    "downloadUrl": null
  }
}
```

#### Database Schema

```sql
CREATE TABLE assets (
  asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(project_id),
  user_id UUID NOT NULL REFERENCES users(user_id),
  file_name VARCHAR(255) NOT NULL,
  file_size BIGINT NOT NULL,
  content_type VARCHAR(100),
  storage_path VARCHAR(500) NOT NULL,
  cdn_url VARCHAR(500),
  thumbnail_url VARCHAR(500),
  metadata JSONB DEFAULT '{}',
  status VARCHAR(20) NOT NULL DEFAULT 'uploading',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  INDEX idx_project_id (project_id),
  INDEX idx_user_id (user_id),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at)
);

CREATE TABLE asset_versions (
  version_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  asset_id UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
  version_number INT NOT NULL,
  storage_path VARCHAR(500) NOT NULL,
  file_size BIGINT NOT NULL,
  created_by UUID NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE(asset_id, version_number),
  INDEX idx_asset_id (asset_id)
);
```

#### Implementation (Go)

```go
package asset

type Service struct {
    db       *sql.DB
    s3       *s3.Client
    cdn      CDNProvider
    processor *AssetProcessor
}

func (s *Service) RequestUploadURL(ctx context.Context, req *UploadURLRequest) (*UploadURLResponse, error) {
    // Validate file size
    if req.FileSize > MaxFileSize {
        return nil, ErrFileTooLarge
    }

    // Create asset record
    asset := &Asset{
        AssetID:     uuid.New().String(),
        ProjectID:   req.ProjectID,
        UserID:      getUserID(ctx),
        FileName:    req.FileName,
        FileSize:    req.FileSize,
        ContentType: req.ContentType,
        StoragePath: s.generateStoragePath(req.ProjectID, req.FileName),
        Status:      StatusUploading,
    }

    if err := s.db.CreateAsset(ctx, asset); err != nil {
        return nil, err
    }

    // Generate presigned URL for S3 upload
    uploadURL, err := s.s3.GeneratePresignedPutURL(
        asset.StoragePath,
        req.ContentType,
        1*time.Hour,
    )
    if err != nil {
        return nil, err
    }

    return &UploadURLResponse{
        AssetID:   asset.AssetID,
        UploadURL: uploadURL,
        ExpiresIn: 3600,
    }, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, assetID string, md5 string) error {
    // Get asset
    asset, err := s.db.GetAsset(ctx, assetID)
    if err != nil {
        return err
    }

    // Verify file exists in S3
    exists, actualMD5, err := s.s3.VerifyFile(asset.StoragePath, md5)
    if err != nil || !exists {
        return ErrUploadFailed
    }

    // Update asset status
    asset.Status = StatusProcessing
    if err := s.db.UpdateAsset(ctx, asset); err != nil {
        return err
    }

    // Queue for processing (thumbnail generation, metadata extraction)
    s.processor.Queue(asset)

    return nil
}

func (s *Service) DownloadAsset(ctx context.Context, assetID string) (string, error) {
    asset, err := s.db.GetAsset(ctx, assetID)
    if err != nil {
        return "", err
    }

    // Check permissions
    if !canDownload(ctx, asset) {
        return "", ErrPermissionDenied
    }

    // Return CDN URL if available
    if asset.CDNURL != "" {
        return asset.CDNURL, nil
    }

    // Generate presigned download URL
    downloadURL, err := s.s3.GeneratePresignedGetURL(asset.StoragePath, 1*time.Hour)
    if err != nil {
        return "", err
    }

    return downloadURL, nil
}
```

---

### 4. Authentication Service

#### Responsibilities
- User registration and login
- OAuth 2.0 integration
- JWT token generation and validation
- Session management

#### Authentication Flow

```
┌────────┐                     ┌──────────────┐                  ┌───────────┐
│ Client │                     │  Auth Service │                  │  Database │
└───┬────┘                     └──────┬───────┘                  └─────┬─────┘
    │                                 │                                 │
    │ POST /auth/login                │                                 │
    ├────────────────────────────────>│                                 │
    │                                 │ Query user                      │
    │                                 ├────────────────────────────────>│
    │                                 │<────────────────────────────────┤
    │                                 │ User data                       │
    │                                 │                                 │
    │                                 │ Verify password                 │
    │                                 │ Generate JWT                    │
    │                                 │ Store session in Redis          │
    │                                 │                                 │
    │<────────────────────────────────┤                                 │
    │ JWT Token + Refresh Token       │                                 │
    │                                 │                                 │
```

#### JWT Structure

```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user_xyz123",
    "iat": 1700650000,
    "exp": 1700653600,
    "iss": "collab.example.com",
    "aud": "collab-api",
    "roles": ["developer"],
    "permissions": ["read:projects", "write:projects"]
  },
  "signature": "..."
}
```

#### Database Schema

```sql
CREATE TABLE users (
  user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255),
  name VARCHAR(255),
  avatar_url VARCHAR(500),
  oauth_provider VARCHAR(50),
  oauth_id VARCHAR(255),
  email_verified BOOLEAN DEFAULT FALSE,
  mfa_enabled BOOLEAN DEFAULT FALSE,
  mfa_secret VARCHAR(255),
  status VARCHAR(20) DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,

  INDEX idx_email (email),
  INDEX idx_oauth (oauth_provider, oauth_id)
);

CREATE TABLE refresh_tokens (
  token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ,

  INDEX idx_user_id (user_id),
  INDEX idx_token_hash (token_hash)
);
```

#### Implementation (Go)

```go
package auth

type Service struct {
    db          *sql.DB
    redis       *redis.Client
    jwtSecret   []byte
    oauthConfig map[string]*oauth2.Config
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
    // Get user from database
    user, err := s.db.GetUserByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    // Verify password
    if !bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) {
        return nil, ErrInvalidCredentials
    }

    // Generate access token (JWT)
    accessToken, err := s.generateJWT(user, 1*time.Hour)
    if err != nil {
        return nil, err
    }

    // Generate refresh token
    refreshToken, err := s.generateRefreshToken(user)
    if err != nil {
        return nil, err
    }

    // Update last login
    s.db.UpdateLastLogin(ctx, user.UserID)

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    3600,
        User:         user.ToPublic(),
    }, nil
}

func (s *Service) ValidateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        // Check if token is revoked (check Redis blacklist)
        if s.isTokenRevoked(claims.JTI) {
            return nil, ErrTokenRevoked
        }

        return claims, nil
    }

    return nil, ErrInvalidToken
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
    // Hash and lookup refresh token
    tokenHash := hashToken(refreshToken)
    token, err := s.db.GetRefreshToken(ctx, tokenHash)
    if err != nil {
        return nil, ErrInvalidToken
    }

    // Check expiration
    if token.ExpiresAt.Before(time.Now()) {
        return nil, ErrTokenExpired
    }

    // Check if revoked
    if token.RevokedAt != nil {
        return nil, ErrTokenRevoked
    }

    // Get user
    user, err := s.db.GetUser(ctx, token.UserID)
    if err != nil {
        return nil, err
    }

    // Generate new access token
    accessToken, err := s.generateJWT(user, 1*time.Hour)
    if err != nil {
        return nil, err
    }

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    3600,
    }, nil
}
```

---

## Data Flow

### Scene Synchronization Flow

```
┌──────────┐          ┌──────────┐          ┌──────────┐
│  User A  │          │  Server  │          │  User B  │
└────┬─────┘          └────┬─────┘          └────┬─────┘
     │                     │                     │
     │ 1. Move object      │                     │
     │ (local update)      │                     │
     │                     │                     │
     │ 2. Send operation   │                     │
     ├────────────────────>│                     │
     │                     │ 3. Transform &      │
     │                     │    apply OT         │
     │                     │                     │
     │                     │ 4. Broadcast        │
     │                     ├────────────────────>│
     │                     │                     │
     │ 5. Ack              │                     │ 6. Apply operation
     │<────────────────────┤                     │    (update UI)
     │                     │                     │
```

### Asset Upload Flow

```
┌────────┐     ┌────────────┐     ┌─────┐     ┌─────┐     ┌─────┐
│ Client │     │   API GW   │     │ S3  │     │Queue│     │Worker│
└───┬────┘     └─────┬──────┘     └──┬──┘     └──┬──┘     └──┬──┘
    │                │                │           │           │
    │ 1. Request URL │                │           │           │
    ├───────────────>│                │           │           │
    │                │ 2. Create      │           │           │
    │                │    asset       │           │           │
    │                │ 3. Presigned   │           │           │
    │                │    URL         │           │           │
    │<───────────────┤                │           │           │
    │                                 │           │           │
    │ 4. Upload file                  │           │           │
    ├────────────────────────────────>│           │           │
    │                                 │           │           │
    │ 5. Confirm upload               │           │           │
    ├───────────────>│                │           │           │
    │                │ 6. Queue job   │           │           │
    │                ├───────────────────────────>│           │
    │                │                │           │ 7. Process│
    │                │                │           ├──────────>│
    │                │                │           │           │ 8. Generate
    │                │                │           │           │    thumbnail
    │                │                │           │           │ 9. Extract
    │                │                │           │           │    metadata
    │                │                │<──────────────────────┤ 10. Upload
    │                │                │           │           │     to CDN
    │                │ 11. Update DB  │           │           │
    │                │<───────────────────────────────────────┤
    │                │                │           │           │
    │ 12. Webhook    │                │           │           │
    │<───────────────┤                │           │           │
```

---

## Communication Protocols

### REST API
- Used for CRUD operations
- Stateless
- JSON payloads
- Standard HTTP methods (GET, POST, PUT, DELETE)

### WebSocket
- Used for real-time synchronization
- Persistent connection
- Bidirectional communication
- JSON or Binary (Protocol Buffers)

### gRPC
- Used for service-to-service communication
- Binary protocol (Protocol Buffers)
- HTTP/2
- Streaming support

### Message Queue
- Used for async operations
- Decouples services
- Guaranteed delivery
- Pub/Sub pattern

---

## State Management

### Client State
- **Local State**: Immediate UI updates
- **Optimistic Updates**: Apply changes before server confirmation
- **Rollback**: Revert on server rejection

### Server State
- **Session State**: Stored in Redis (fast access)
- **Persistent State**: Stored in PostgreSQL
- **Cache State**: Multi-level caching

### Conflict Resolution
- **Operational Transform**: For concurrent edits
- **Last Write Wins**: For simple conflicts
- **Manual Resolution**: For complex conflicts

---

## Error Handling

### Error Categories
1. **Client Errors** (4xx): Invalid request, authentication failure
2. **Server Errors** (5xx): Internal error, service unavailable
3. **Network Errors**: Timeout, connection lost
4. **Business Logic Errors**: Validation failure, permission denied

### Retry Strategy
```javascript
async function retryWithBackoff(fn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (!isRetryable(error) || i === maxRetries - 1) {
        throw error;
      }
      await sleep(Math.pow(2, i) * 1000);
    }
  }
}

function isRetryable(error) {
  return error.status >= 500 || error.code === 'NETWORK_ERROR';
}
```

### Circuit Breaker
```go
type CircuitBreaker struct {
    maxFailures    int
    resetTimeout   time.Duration
    state          State
    failures       int
    lastFailureTime time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == StateOpen {
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.state = StateHalfOpen
        } else {
            return ErrCircuitOpen
        }
    }

    err := fn()
    if err != nil {
        cb.onFailure()
        return err
    }

    cb.onSuccess()
    return nil
}
```

---

## Performance Requirements

### Latency Targets
| Operation | Target | Max |
|-----------|--------|-----|
| API Response | < 100ms (p95) | < 500ms |
| Sync Propagation | < 50ms (p95) | < 200ms |
| Asset Upload Start | < 200ms | < 1s |
| Asset Download Start | < 100ms | < 500ms |
| WebSocket Connect | < 500ms | < 2s |

### Throughput Targets
| Metric | Target | Max |
|--------|--------|-----|
| API Requests | 10,000/s | 50,000/s |
| WebSocket Messages | 100,000/s | 500,000/s |
| Concurrent Connections | 10,000 | 50,000 |
| Asset Upload Bandwidth | 10 GB/s | 50 GB/s |

### Resource Limits
| Resource | Limit |
|----------|-------|
| Max File Size | 5 GB |
| Max Session Participants | 50 |
| Max API Rate | 1,000 req/min per user |
| Max WebSocket Message Size | 10 MB |

---

## Conclusion

This system design provides a comprehensive blueprint for building a scalable, real-time Unity collaboration platform. Each service is designed for independence, scalability, and reliability.

**Next Steps**:
1. Implement core services (Session, Sync, Asset)
2. Set up infrastructure (Kubernetes, databases)
3. Build Unity plugin
4. Integrate and test
5. Deploy to staging
6. Performance testing and optimization
7. Launch to production
