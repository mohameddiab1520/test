# Presence Service

Real-time user presence and awareness service for Unity Collaboration Platform.

## Overview

The Presence Service tracks real-time user status, cursor positions, scene selections, and activity within collaboration sessions. It provides WebSocket-based real-time updates and maintains an activity feed for each session.

## Features

- **Real-Time Presence Tracking**: Online/away/idle/offline status
- **Cursor & Selection Tracking**: Real-time cursor positions and object selections
- **Camera Position Tracking**: Track user camera positions in Unity scenes
- **Typing Indicators**: Show when users are typing
- **Activity Feed**: Complete history of session activities
- **WebSocket Communication**: Low-latency bidirectional communication
- **Prometheus Metrics**: Performance monitoring

## Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin (HTTP), Gorilla WebSocket
- **Databases**: PostgreSQL (persistence), Redis (real-time cache)
- **Monitoring**: Prometheus metrics

## API Endpoints

### REST API

```
GET    /api/v1/presence/sessions/:sessionId           - Get all participants' presence
GET    /api/v1/presence/users/:userId                 - Get user's presence status
PUT    /api/v1/presence/status                        - Update presence status
POST   /api/v1/presence/activity-feed/:sessionId      - Get activity feed
DELETE /api/v1/presence/sessions/:sessionId/:userId   - Clear presence
```

### WebSocket

```
WS     /ws?sessionId=xxx&userId=yyy                   - WebSocket connection
```

## WebSocket Protocol

### Client → Server (Presence Update)

```json
{
  "type": "presence",
  "session_id": "sess_abc123",
  "presence": {
    "status": "online",
    "current_scene": "MainScene",
    "selected_object": "obj_player_001",
    "cursor_position": {"x": 100, "y": 200},
    "camera_transform": {
      "position": {"x": 0, "y": 5, "z": -10},
      "rotation": {"x": 15, "y": 0, "z": 0}
    },
    "is_typing": false
  }
}
```

### Server → Client (Presence Broadcast)

```json
{
  "type": "presence_update",
  "user_id": "user_abc",
  "presence": {
    "status": "online",
    "current_scene": "MainScene",
    "selected_object": "obj_player_001",
    "cursor_position": {"x": 100, "y": 200},
    "user_name": "John Doe",
    "user_avatar": "https://..."
  }
}
```

## Configuration

Environment variables:

```bash
SERVICE_NAME=presence-service
SERVICE_PORT=8085
ENVIRONMENT=development

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=collab_dev
POSTGRES_USER=dev
POSTGRES_PASSWORD=devpass

REDIS_HOST=localhost
REDIS_PORT=6379

PRESENCE_TTL=300
ACTIVITY_RETENTION_DAYS=7
STATUS_UPDATE_INTERVAL=30
```

## Development

### Build

```bash
go mod download
go build -o presence-service ./cmd/server
```

### Run

```bash
./presence-service
```

### Docker

```bash
docker build -t presence-service .
docker run -p 8085:8085 presence-service
```

## Database Schema

### PostgreSQL

**presence_states** table:
```sql
CREATE TABLE presence_states (
    presence_id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    current_scene VARCHAR(255),
    selected_object VARCHAR(255),
    cursor_position JSONB,
    camera_transform JSONB,
    is_typing BOOLEAN DEFAULT FALSE,
    last_activity_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE(session_id, user_id)
);
```

**activity_feed** table:
```sql
CREATE TABLE activity_feed (
    activity_id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,
    activity_type VARCHAR(50) NOT NULL,
    object_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL
);
```

### Redis

```
Key: presence:{sessionId}:{userId}
Value: JSON presence state
TTL: 5 minutes

Channel: presence:{sessionId}  (Pub/Sub for broadcasts)
```

## Metrics Exported

- `presence_updates_total` - Total presence updates by session
- `activity_feed_events_total` - Total activity events by type
- `websocket_connections` - Active WebSocket connections

## Testing

```bash
go test ./... -v
```

## License

MIT
