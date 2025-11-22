# Unity Collaboration Platform - Technical Specifications

## Table of Contents

1. [Development Environment](#development-environment)
2. [Technology Stack Details](#technology-stack-details)
3. [Code Standards](#code-standards)
4. [Testing Strategy](#testing-strategy)
5. [Performance Optimization](#performance-optimization)
6. [Security Implementation](#security-implementation)
7. [Monitoring & Logging](#monitoring--logging)

---

## Development Environment

### Required Tools

#### Backend Development
```bash
# Go (1.21+)
go version

# Node.js (20.x LTS)
node --version
npm --version

# Python (3.11+)
python --version

# Docker & Docker Compose
docker --version
docker-compose --version

# Kubernetes CLI
kubectl version

# Helm
helm version

# gRPC tools
protoc --version
```

#### Unity Development
```
Unity 2022.3 LTS or higher
Unity Hub
Visual Studio 2022 / Rider
.NET 6.0 SDK
```

#### Database Tools
```bash
# PostgreSQL client
psql --version

# Redis CLI
redis-cli --version

# MongoDB shell
mongosh --version
```

### Local Development Setup

#### 1. Clone Repository
```bash
git clone https://github.com/yourorg/unity-collab-platform.git
cd unity-collab-platform
```

#### 2. Start Infrastructure with Docker Compose
```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: collab_dev
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: devpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: dev
      MONGO_INITDB_ROOT_PASSWORD: devpass
    volumes:
      - mongo_data:/data/db

  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: dev
      RABBITMQ_DEFAULT_PASS: devpass

  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  redis_data:
  mongo_data:
  minio_data:
```

```bash
docker-compose -f docker-compose.dev.yml up -d
```

#### 3. Initialize Databases
```bash
# Run migrations
make migrate-up

# Seed development data
make seed-dev
```

#### 4. Start Services
```bash
# Terminal 1 - Session Service
cd services/session
go run cmd/server/main.go

# Terminal 2 - Sync Service
cd services/sync
npm run dev

# Terminal 3 - Asset Service
cd services/asset
go run cmd/server/main.go

# Terminal 4 - API Gateway
cd gateway
npm run dev
```

---

## Technology Stack Details

### Backend Services

#### Go Services (Session, Asset, Auth)

**Project Structure**
```
services/session/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes.go
│   ├── service/             # Business logic
│   │   └── session.go
│   ├── repository/          # Data access
│   │   └── session_repo.go
│   ├── models/              # Domain models
│   │   └── session.go
│   └── config/              # Configuration
│       └── config.go
├── pkg/                     # Shared packages
│   ├── logger/
│   ├── errors/
│   └── validation/
├── proto/                   # Protocol Buffers
│   └── session.proto
├── migrations/              # Database migrations
│   └── 001_initial.sql
├── tests/                   # Tests
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

**Dependencies (go.mod)**
```go
module github.com/yourorg/collab/services/session

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    github.com/go-redis/redis/v8 v8.11.5
    github.com/google/uuid v1.5.0
    github.com/golang-jwt/jwt/v5 v5.2.0
    google.golang.org/grpc v1.60.1
    google.golang.org/protobuf v1.32.0
    github.com/stretchr/testify v1.8.4
    go.uber.org/zap v1.26.0
    github.com/spf13/viper v1.18.2
)
```

**Main Application**
```go
// cmd/server/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/yourorg/collab/services/session/internal/api"
    "github.com/yourorg/collab/services/session/internal/config"
    "github.com/yourorg/collab/services/session/internal/repository"
    "github.com/yourorg/collab/services/session/internal/service"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // Initialize logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Initialize database
    db, err := repository.NewPostgresDB(cfg.Database.URL)
    if err != nil {
        logger.Fatal("Failed to connect to database", zap.Error(err))
    }
    defer db.Close()

    // Initialize Redis
    redisClient := repository.NewRedisClient(cfg.Redis.URL)
    defer redisClient.Close()

    // Initialize repository
    repo := repository.NewSessionRepository(db, redisClient)

    // Initialize service
    svc := service.NewSessionService(repo, logger)

    // Initialize HTTP server
    router := gin.Default()
    api.RegisterRoutes(router, svc)

    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
        Handler: router,
    }

    // Start server
    go func() {
        logger.Info("Starting server", zap.Int("port", cfg.Server.Port))
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Server failed", zap.Error(err))
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}
```

**Configuration**
```go
// internal/config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    JWT      JWTConfig
    Logger   LoggerConfig
}

type ServerConfig struct {
    Port int
    Mode string
}

type DatabaseConfig struct {
    URL             string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime int
}

type RedisConfig struct {
    URL string
    DB  int
}

type JWTConfig struct {
    Secret     string
    Expiration int
}

type LoggerConfig struct {
    Level  string
    Format string
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./config")
    viper.AddConfigPath(".")

    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

**Service Layer**
```go
// internal/service/session.go
package service

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/yourorg/collab/services/session/internal/models"
    "github.com/yourorg/collab/services/session/internal/repository"
)

type SessionService struct {
    repo   repository.SessionRepository
    logger *zap.Logger
}

func NewSessionService(repo repository.SessionRepository, logger *zap.Logger) *SessionService {
    return &SessionService{
        repo:   repo,
        logger: logger,
    }
}

func (s *SessionService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*models.Session, error) {
    // Validate input
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Create session model
    session := &models.Session{
        ID:        uuid.New().String(),
        ProjectID: req.ProjectID,
        OwnerID:   req.UserID,
        Name:      req.Name,
        Settings:  req.Settings,
        Status:    models.SessionStatusActive,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // Persist to database
    if err := s.repo.Create(ctx, session); err != nil {
        s.logger.Error("Failed to create session", zap.Error(err))
        return nil, fmt.Errorf("failed to create session: %w", err)
    }

    s.logger.Info("Session created", zap.String("sessionId", session.ID))

    return session, nil
}

func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
    // Try cache first
    session, err := s.repo.GetFromCache(ctx, sessionID)
    if err == nil {
        return session, nil
    }

    // Fallback to database
    session, err = s.repo.GetByID(ctx, sessionID)
    if err != nil {
        return nil, fmt.Errorf("session not found: %w", err)
    }

    // Update cache
    go s.repo.SetCache(context.Background(), session)

    return session, nil
}

func (s *SessionService) JoinSession(ctx context.Context, sessionID, userID string) error {
    session, err := s.GetSession(ctx, sessionID)
    if err != nil {
        return err
    }

    // Check permissions
    if !session.CanJoin(userID) {
        return ErrPermissionDenied
    }

    // Check max participants
    if len(session.Participants) >= session.Settings.MaxParticipants {
        return ErrSessionFull
    }

    // Add participant
    participant := &models.Participant{
        SessionID: sessionID,
        UserID:    userID,
        Role:      models.RoleViewer,
        JoinedAt:  time.Now(),
    }

    if err := s.repo.AddParticipant(ctx, participant); err != nil {
        return fmt.Errorf("failed to add participant: %w", err)
    }

    // Publish event
    s.publishEvent("session.participant_joined", participant)

    s.logger.Info("User joined session",
        zap.String("sessionId", sessionID),
        zap.String("userId", userID))

    return nil
}

type CreateSessionRequest struct {
    ProjectID string
    UserID    string
    Name      string
    Settings  models.SessionSettings
}

func (r *CreateSessionRequest) Validate() error {
    if r.ProjectID == "" {
        return fmt.Errorf("projectId is required")
    }
    if r.UserID == "" {
        return fmt.Errorf("userId is required")
    }
    if r.Name == "" {
        return fmt.Errorf("name is required")
    }
    return nil
}
```

**Repository Layer**
```go
// internal/repository/session_repo.go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    _ "github.com/lib/pq"

    "github.com/yourorg/collab/services/session/internal/models"
)

type SessionRepository interface {
    Create(ctx context.Context, session *models.Session) error
    GetByID(ctx context.Context, id string) (*models.Session, error)
    Update(ctx context.Context, session *models.Session) error
    Delete(ctx context.Context, id string) error
    GetFromCache(ctx context.Context, id string) (*models.Session, error)
    SetCache(ctx context.Context, session *models.Session) error
    AddParticipant(ctx context.Context, participant *models.Participant) error
}

type sessionRepository struct {
    db    *sql.DB
    redis *redis.Client
}

func NewSessionRepository(db *sql.DB, redis *redis.Client) SessionRepository {
    return &sessionRepository{
        db:    db,
        redis: redis,
    }
}

func (r *sessionRepository) Create(ctx context.Context, session *models.Session) error {
    query := `
        INSERT INTO sessions (session_id, project_id, owner_id, name, settings, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

    settingsJSON, _ := json.Marshal(session.Settings)

    _, err := r.db.ExecContext(ctx, query,
        session.ID,
        session.ProjectID,
        session.OwnerID,
        session.Name,
        settingsJSON,
        session.Status,
        session.CreatedAt,
        session.UpdatedAt,
    )

    return err
}

func (r *sessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
    query := `
        SELECT session_id, project_id, owner_id, name, settings, status, created_at, updated_at
        FROM sessions
        WHERE session_id = $1
    `

    var session models.Session
    var settingsJSON []byte

    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &session.ID,
        &session.ProjectID,
        &session.OwnerID,
        &session.Name,
        &settingsJSON,
        &session.Status,
        &session.CreatedAt,
        &session.UpdatedAt,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrSessionNotFound
        }
        return nil, err
    }

    json.Unmarshal(settingsJSON, &session.Settings)

    return &session, nil
}

func (r *sessionRepository) GetFromCache(ctx context.Context, id string) (*models.Session, error) {
    key := fmt.Sprintf("session:%s", id)
    data, err := r.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }

    var session models.Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }

    return &session, nil
}

func (r *sessionRepository) SetCache(ctx context.Context, session *models.Session) error {
    key := fmt.Sprintf("session:%s", session.ID)
    data, _ := json.Marshal(session)
    return r.redis.Set(ctx, key, data, 1*time.Hour).Err()
}
```

#### Node.js Service (Sync Service - Real-time)

**Project Structure**
```
services/sync/
├── src/
│   ├── index.ts             # Entry point
│   ├── server.ts            # WebSocket server
│   ├── controllers/
│   │   └── sync.controller.ts
│   ├── services/
│   │   ├── sync.service.ts
│   │   └── ot.service.ts    # Operational Transform
│   ├── models/
│   │   ├── operation.model.ts
│   │   └── session.model.ts
│   ├── utils/
│   │   ├── logger.ts
│   │   └── validator.ts
│   └── config/
│       └── config.ts
├── tests/
│   ├── unit/
│   └── integration/
├── package.json
├── tsconfig.json
├── Dockerfile
└── .env.example
```

**Dependencies (package.json)**
```json
{
  "name": "@collab/sync-service",
  "version": "1.0.0",
  "scripts": {
    "dev": "tsx watch src/index.ts",
    "build": "tsc",
    "start": "node dist/index.js",
    "test": "jest",
    "lint": "eslint src/**/*.ts"
  },
  "dependencies": {
    "ws": "^8.16.0",
    "express": "^4.18.2",
    "redis": "^4.6.12",
    "ioredis": "^5.3.2",
    "uuid": "^9.0.1",
    "zod": "^3.22.4",
    "winston": "^3.11.0",
    "dotenv": "^16.3.1"
  },
  "devDependencies": {
    "@types/node": "^20.11.0",
    "@types/ws": "^8.5.10",
    "@types/express": "^4.17.21",
    "typescript": "^5.3.3",
    "tsx": "^4.7.0",
    "jest": "^29.7.0",
    "eslint": "^8.56.0"
  }
}
```

**WebSocket Server**
```typescript
// src/server.ts
import { WebSocketServer, WebSocket } from 'ws';
import { IncomingMessage } from 'http';
import { parse } from 'url';
import { validateToken } from './auth';
import { SyncService } from './services/sync.service';
import { logger } from './utils/logger';

interface Client {
  userId: string;
  sessionId: string;
  ws: WebSocket;
}

export class SyncServer {
  private wss: WebSocketServer;
  private clients: Map<string, Client> = new Map();
  private syncService: SyncService;

  constructor(port: number) {
    this.wss = new WebSocketServer({ port });
    this.syncService = new SyncService();
    this.setupHandlers();
  }

  private setupHandlers() {
    this.wss.on('connection', (ws: WebSocket, req: IncomingMessage) => {
      this.handleConnection(ws, req);
    });
  }

  private async handleConnection(ws: WebSocket, req: IncomingMessage) {
    try {
      // Parse URL and extract token
      const { query } = parse(req.url || '', true);
      const token = query.token as string;

      // Validate token
      const { userId, sessionId } = await validateToken(token);

      // Register client
      const clientId = `${sessionId}:${userId}`;
      const client: Client = { userId, sessionId, ws };
      this.clients.set(clientId, client);

      logger.info('Client connected', { userId, sessionId });

      // Send current session state
      const sessionState = await this.syncService.getSessionState(sessionId);
      this.send(ws, { type: 'init', data: sessionState });

      // Handle messages
      ws.on('message', (data: Buffer) => {
        this.handleMessage(client, data);
      });

      // Handle disconnect
      ws.on('close', () => {
        this.handleDisconnect(client);
      });

      ws.on('error', (error) => {
        logger.error('WebSocket error', { error, userId, sessionId });
      });

    } catch (error) {
      logger.error('Connection failed', { error });
      ws.close(1008, 'Authentication failed');
    }
  }

  private async handleMessage(client: Client, data: Buffer) {
    try {
      const message = JSON.parse(data.toString());

      switch (message.type) {
        case 'operation':
          await this.handleOperation(client, message.operation);
          break;

        case 'ping':
          this.send(client.ws, { type: 'pong' });
          break;

        default:
          logger.warn('Unknown message type', { type: message.type });
      }
    } catch (error) {
      logger.error('Failed to handle message', { error });
      this.send(client.ws, { type: 'error', error: 'Invalid message' });
    }
  }

  private async handleOperation(client: Client, operation: any) {
    try {
      // Apply operation using OT
      const result = await this.syncService.applyOperation(
        client.sessionId,
        client.userId,
        operation
      );

      if (!result.success) {
        // Send sync required
        this.send(client.ws, {
          type: 'sync_required',
          version: result.currentVersion,
          operations: result.missingOperations
        });
        return;
      }

      // Broadcast to other clients in the session
      this.broadcast(client.sessionId, client.userId, {
        type: 'sync',
        operation: result.operation
      });

      // Send acknowledgment
      this.send(client.ws, {
        type: 'ack',
        success: true,
        version: result.version
      });

    } catch (error) {
      logger.error('Failed to handle operation', { error });
      this.send(client.ws, { type: 'error', error: 'Operation failed' });
    }
  }

  private handleDisconnect(client: Client) {
    const clientId = `${client.sessionId}:${client.userId}`;
    this.clients.delete(clientId);
    logger.info('Client disconnected', {
      userId: client.userId,
      sessionId: client.sessionId
    });
  }

  private send(ws: WebSocket, message: any) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }

  private broadcast(sessionId: string, excludeUserId: string, message: any) {
    for (const [clientId, client] of this.clients) {
      if (client.sessionId === sessionId && client.userId !== excludeUserId) {
        this.send(client.ws, message);
      }
    }
  }
}
```

**Operational Transform Service**
```typescript
// src/services/ot.service.ts
import { v4 as uuidv4 } from 'uuid';

export enum OperationType {
  CREATE = 'create',
  UPDATE = 'update',
  DELETE = 'delete',
  TRANSFORM = 'transform'
}

export interface Operation {
  id: string;
  type: OperationType;
  objectId: string;
  path: string;
  value: any;
  version: number;
  userId?: string;
  timestamp: number;
}

export class OTService {
  /**
   * Transform two operations for concurrent execution
   */
  transform(op1: Operation, op2: Operation): [Operation | null, Operation | null] {
    // Different objects - no conflict
    if (op1.objectId !== op2.objectId) {
      return [op1, op2];
    }

    // Different paths - no conflict
    if (op1.path !== op2.path) {
      return [op1, op2];
    }

    // Same object, same path - resolve conflict
    return this.resolveConflict(op1, op2);
  }

  private resolveConflict(op1: Operation, op2: Operation): [Operation | null, Operation | null] {
    // Delete always wins
    if (op1.type === OperationType.DELETE) {
      return [op1, null];
    }
    if (op2.type === OperationType.DELETE) {
      return [null, op2];
    }

    // For updates, use Last Write Wins based on timestamp
    if (op1.type === OperationType.UPDATE && op2.type === OperationType.UPDATE) {
      if (op1.timestamp > op2.timestamp) {
        return [op1, null];
      } else if (op2.timestamp > op1.timestamp) {
        return [null, op2];
      } else {
        // Same timestamp, use userId for deterministic resolution
        return op1.userId! > op2.userId! ? [op1, null] : [null, op2];
      }
    }

    // Create conflicts
    if (op1.type === OperationType.CREATE && op2.type === OperationType.CREATE) {
      // First create wins
      return op1.timestamp < op2.timestamp ? [op1, null] : [null, op2];
    }

    return [op1, op2];
  }

  /**
   * Apply transformation to a series of operations
   */
  transformAgainstHistory(
    operation: Operation,
    history: Operation[]
  ): Operation | null {
    let transformed = operation;

    for (const historicOp of history) {
      if (historicOp.timestamp >= operation.timestamp) {
        const [op1] = this.transform(transformed, historicOp);
        if (!op1) {
          return null; // Operation was dropped
        }
        transformed = op1;
      }
    }

    return transformed;
  }
}
```

---

## Code Standards

### Go Code Standards

```go
// Package naming: lowercase, single word
package session

// Exported types: PascalCase
type SessionService struct {}

// Unexported types: camelCase
type sessionRepository struct {}

// Constants: PascalCase or SCREAMING_SNAKE_CASE
const (
    MaxSessionDuration = 24 * time.Hour
    DEFAULT_TIMEOUT    = 30 * time.Second
)

// Error variables: Err prefix
var (
    ErrSessionNotFound   = errors.New("session not found")
    ErrPermissionDenied  = errors.New("permission denied")
)

// Interface naming: -er suffix
type SessionManager interface {
    Create(ctx context.Context, req *CreateRequest) (*Session, error)
    Get(ctx context.Context, id string) (*Session, error)
}

// Always use context as first parameter
func (s *Service) CreateSession(ctx context.Context, req *Request) error {
    // Implementation
}

// Use named return values for clarity
func (s *Service) GetSession(ctx context.Context, id string) (session *Session, err error) {
    // Implementation
}

// Table-driven tests
func TestSessionService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateRequest
        want    *Session
        wantErr bool
    }{
        {
            name: "valid session",
            input: &CreateRequest{Name: "Test"},
            want: &Session{Name: "Test"},
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := svc.Create(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
            }
            // Assert got == tt.want
        })
    }
}
```

### TypeScript Code Standards

```typescript
// Use PascalCase for classes and interfaces
export class SyncService {
  private clients: Map<string, Client>;

  constructor() {
    this.clients = new Map();
  }
}

export interface Operation {
  id: string;
  type: OperationType;
}

// Use camelCase for variables and functions
const sessionId = 'sess_123';

function handleOperation(op: Operation): void {
  // Implementation
}

// Use SCREAMING_SNAKE_CASE for constants
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 30000;

// Prefer const over let
const users = [];

// Use async/await over callbacks
async function fetchSession(id: string): Promise<Session> {
  const response = await fetch(`/api/sessions/${id}`);
  return response.json();
}

// Type everything
function createSession(name: string, ownerId: string): Session {
  return {
    id: uuidv4(),
    name,
    ownerId,
    createdAt: new Date()
  };
}

// Use Zod for runtime validation
import { z } from 'zod';

const OperationSchema = z.object({
  type: z.enum(['create', 'update', 'delete']),
  objectId: z.string().uuid(),
  path: z.string(),
  value: z.any(),
  version: z.number().int().positive()
});

type Operation = z.infer<typeof OperationSchema>;

function validateOperation(data: unknown): Operation {
  return OperationSchema.parse(data);
}
```

### C# Unity Code Standards

```csharp
// Use PascalCase for public members
public class CollaborationManager : MonoBehaviour
{
    // Use PascalCase for public fields
    public string SessionId { get; private set; }

    // Use camelCase with _ prefix for private fields
    private WebSocketClient _wsClient;
    private Dictionary<string, GameObject> _syncedObjects;

    // Use PascalCase for methods
    public async Task JoinSession(string sessionId)
    {
        // Implementation
    }

    // Use async/await
    private async Task<Session> FetchSessionData(string sessionId)
    {
        var response = await _httpClient.GetAsync($"/api/sessions/{sessionId}");
        return await response.Content.ReadAsAsync<Session>();
    }

    // Unity lifecycle methods
    private void Start()
    {
        Initialize();
    }

    private void Update()
    {
        ProcessSyncQueue();
    }

    private void OnDestroy()
    {
        Cleanup();
    }
}

// Use regions for organization
#region Public Methods
public void Connect() { }
#endregion

#region Private Methods
private void ProcessQueue() { }
#endregion

// Null checking with null-conditional operator
var name = user?.Name ?? "Unknown";

// Pattern matching
if (obj is Player player)
{
    player.Move();
}
```

---

## Testing Strategy

### Unit Tests

**Go Unit Test Example**
```go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/yourorg/collab/services/session/internal/service"
)

// Mock repository
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, session *models.Session) error {
    args := m.Called(ctx, session)
    return args.Error(0)
}

func TestSessionService_CreateSession(t *testing.T) {
    // Arrange
    mockRepo := new(MockRepository)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

    svc := service.NewSessionService(mockRepo, logger)

    req := &service.CreateSessionRequest{
        ProjectID: "proj_123",
        UserID:    "user_456",
        Name:      "Test Session",
    }

    // Act
    session, err := svc.CreateSession(context.Background(), req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, session)
    assert.Equal(t, "Test Session", session.Name)
    mockRepo.AssertExpectations(t)
}
```

**TypeScript Unit Test Example**
```typescript
import { SyncService } from '../services/sync.service';
import { OTService } from '../services/ot.service';

describe('SyncService', () => {
  let syncService: SyncService;
  let otService: OTService;

  beforeEach(() => {
    otService = new OTService();
    syncService = new SyncService(otService);
  });

  describe('applyOperation', () => {
    it('should apply operation and increment version', async () => {
      // Arrange
      const sessionId = 'sess_123';
      const userId = 'user_456';
      const operation = {
        type: 'update',
        objectId: 'obj_1',
        path: 'transform.position.x',
        value: 10,
        version: 0,
        timestamp: Date.now()
      };

      // Act
      const result = await syncService.applyOperation(sessionId, userId, operation);

      // Assert
      expect(result.success).toBe(true);
      expect(result.version).toBe(1);
      expect(result.operation.userId).toBe(userId);
    });

    it('should require sync when version mismatch', async () => {
      // Test implementation
    });
  });
});
```

### Integration Tests

```go
func TestSessionAPI_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // Start test server
    server := startTestServer(t, db)
    defer server.Close()

    t.Run("Create and retrieve session", func(t *testing.T) {
        // Create session
        createResp := createSession(t, server.URL, &CreateRequest{
            Name:      "Integration Test",
            ProjectID: "proj_123",
        })

        assert.Equal(t, http.StatusCreated, createResp.StatusCode)

        var session Session
        json.NewDecoder(createResp.Body).Decode(&session)

        // Retrieve session
        getResp := getSession(t, server.URL, session.ID)
        assert.Equal(t, http.StatusOK, getResp.StatusCode)

        var retrieved Session
        json.NewDecoder(getResp.Body).Decode(&retrieved)

        assert.Equal(t, session.ID, retrieved.ID)
        assert.Equal(t, session.Name, retrieved.Name)
    })
}
```

### E2E Tests

```typescript
import { test, expect } from '@playwright/test';

test.describe('Collaboration Flow', () => {
  test('should allow two users to collaborate on a scene', async ({ browser }) => {
    // Create two browser contexts (two users)
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    // User 1: Create session
    await page1.goto('/');
    await page1.click('[data-testid="create-session"]');
    await page1.fill('[name="sessionName"]', 'E2E Test Session');
    await page1.click('[data-testid="submit"]');

    const sessionUrl = await page1.url();

    // User 2: Join session
    await page2.goto(sessionUrl);

    // Verify both users see each other
    await expect(page1.locator('[data-testid="participant-count"]')).toHaveText('2');
    await expect(page2.locator('[data-testid="participant-count"]')).toHaveText('2');

    // User 1: Move object
    await page1.click('[data-testid="object-cube"]');
    await page1.fill('[name="position-x"]', '10');

    // User 2: Should see the change
    await expect(page2.locator('[data-testid="cube-position-x"]')).toHaveText('10');
  });
});
```

---

## Performance Optimization

### Database Optimization

```sql
-- Proper indexing
CREATE INDEX idx_sessions_project_id ON sessions(project_id);
CREATE INDEX idx_sessions_owner_id ON sessions(owner_id);
CREATE INDEX idx_sessions_status_created ON sessions(status, created_at DESC);

-- Composite index for common queries
CREATE INDEX idx_sessions_project_status ON sessions(project_id, status) WHERE status = 'active';

-- Partial index
CREATE INDEX idx_active_sessions ON sessions(created_at) WHERE status = 'active';

-- Connection pooling
-- PostgreSQL config
max_connections = 200
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 64MB
```

### Caching Strategy

```go
// Multi-level caching
type Cache struct {
    l1 *sync.Map          // In-memory cache
    l2 *redis.Client      // Redis cache
    l3 Database           // Database
}

func (c *Cache) Get(ctx context.Context, key string) (interface{}, error) {
    // L1: In-memory
    if val, ok := c.l1.Load(key); ok {
        return val, nil
    }

    // L2: Redis
    val, err := c.l2.Get(ctx, key).Result()
    if err == nil {
        // Populate L1
        c.l1.Store(key, val)
        return val, nil
    }

    // L3: Database
    val, err = c.l3.Get(ctx, key)
    if err != nil {
        return nil, err
    }

    // Populate L2 and L1
    go c.l2.Set(ctx, key, val, 1*time.Hour)
    c.l1.Store(key, val)

    return val, nil
}
```

### WebSocket Optimization

```typescript
// Connection pooling
class WebSocketPool {
  private pools: Map<string, WebSocket[]> = new Map();
  private maxPoolSize = 100;

  getConnection(url: string): WebSocket {
    const pool = this.pools.get(url) || [];

    // Reuse existing connection
    const available = pool.find(ws => ws.readyState === WebSocket.OPEN);
    if (available) {
      return available;
    }

    // Create new connection
    const ws = new WebSocket(url);
    pool.push(ws);
    this.pools.set(url, pool);

    // Cleanup on close
    ws.on('close', () => {
      const index = pool.indexOf(ws);
      if (index > -1) {
        pool.splice(index, 1);
      }
    });

    return ws;
  }
}

// Message batching
class MessageBatcher {
  private queue: any[] = [];
  private flushInterval = 50; // ms

  constructor() {
    setInterval(() => this.flush(), this.flushInterval);
  }

  add(message: any) {
    this.queue.push(message);

    if (this.queue.length >= 100) {
      this.flush();
    }
  }

  private flush() {
    if (this.queue.length === 0) return;

    const batch = this.queue.splice(0);
    this.send({ type: 'batch', messages: batch });
  }

  private send(data: any) {
    // Send over WebSocket
  }
}
```

---

## Security Implementation

### Input Validation

```go
import "github.com/go-playground/validator/v10"

type CreateSessionRequest struct {
    Name      string `json:"name" validate:"required,min=1,max=255"`
    ProjectID string `json:"projectId" validate:"required,uuid"`
    Settings  struct {
        MaxParticipants int  `json:"maxParticipants" validate:"min=1,max=50"`
        IsPublic        bool `json:"isPublic"`
    } `json:"settings"`
}

var validate = validator.New()

func (r *CreateSessionRequest) Validate() error {
    return validate.Struct(r)
}
```

### SQL Injection Prevention

```go
// GOOD: Use parameterized queries
query := "SELECT * FROM sessions WHERE session_id = $1"
row := db.QueryRow(query, sessionID)

// BAD: String concatenation (vulnerable to SQL injection)
query := fmt.Sprintf("SELECT * FROM sessions WHERE session_id = '%s'", sessionID)
```

### XSS Prevention

```typescript
// Sanitize user input
import DOMPurify from 'dompurify';

function renderUserMessage(message: string): string {
  return DOMPurify.sanitize(message, {
    ALLOWED_TAGS: ['b', 'i', 'em', 'strong'],
    ALLOWED_ATTR: []
  });
}
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func (rl *RateLimiter) Allow(userID string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[userID]
    if !exists {
        // 100 requests per minute
        limiter = rate.NewLimiter(rate.Every(time.Minute/100), 100)
        rl.limiters[userID] = limiter
    }

    return limiter.Allow()
}
```

---

## Monitoring & Logging

### Structured Logging

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()

logger.Info("Session created",
    zap.String("sessionId", session.ID),
    zap.String("userId", userID),
    zap.Int("participants", len(session.Participants)),
    zap.Duration("duration", time.Since(start)),
)

logger.Error("Failed to create session",
    zap.Error(err),
    zap.String("userId", userID),
    zap.Any("request", req),
)
```

### Metrics Collection

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    sessionsCreated = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "sessions_created_total",
            Help: "Total number of sessions created",
        },
        []string{"project_id"},
    )

    sessionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "session_duration_seconds",
            Help:    "Session duration in seconds",
            Buckets: prometheus.ExponentialBuckets(60, 2, 10),
        },
        []string{"project_id"},
    )

    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "websocket_connections_active",
            Help: "Number of active WebSocket connections",
        },
    )
)

func init() {
    prometheus.MustRegister(sessionsCreated, sessionDuration, activeConnections)
}

// Usage
sessionsCreated.WithLabelValues(projectID).Inc()
activeConnections.Inc()
defer activeConnections.Dec()
```

### Distributed Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *Service) CreateSession(ctx context.Context, req *Request) (*Session, error) {
    ctx, span := otel.Tracer("session-service").Start(ctx, "CreateSession")
    defer span.End()

    span.SetAttributes(
        attribute.String("project.id", req.ProjectID),
        attribute.String("user.id", req.UserID),
    )

    // Business logic...

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    return session, nil
}
```

---

This technical specification provides a comprehensive guide for implementing the Unity Collaboration Platform with best practices, code examples, and detailed standards for all technology stack components.
