# Sync Service

Real-time synchronization microservice using WebSocket for Unity Collaboration Platform.

## Overview

The Sync Service handles real-time data synchronization:
- WebSocket connections management
- Operational Transform (OT) for conflict-free edits
- Real-time scene state broadcasting
- Event propagation to connected clients

## Tech Stack

- **Language**: TypeScript
- **Runtime**: Node.js 20+
- **WebSocket**: ws library
- **Cache**: Redis (ioredis)
- **Protocol**: WebSocket + JSON

## Running Locally

```bash
# Install dependencies
npm install

# Run in development mode
npm run dev

# Build for production
npm run build

# Run production build
npm start

# Or use Make
make run-sync
```

## WebSocket Protocol

### Client → Server Messages

```json
{
  "type": "operation",
  "sessionId": "sess_abc123",
  "operation": {
    "type": "update",
    "objectId": "obj_player_001",
    "path": "transform.position.x",
    "value": 10.5,
    "version": 42
  }
}
```

### Server → Client Messages

```json
{
  "type": "sync",
  "operation": {
    "id": "op_xyz789",
    "type": "update",
    "objectId": "obj_player_001",
    "userId": "user_abc",
    "value": 10.5,
    "timestamp": "2025-11-22T10:30:00Z"
  }
}
```

## Environment Variables

- `PORT` - WebSocket server port (default: 8081)
- `REDIS_URL` - Redis connection string
- `LOG_LEVEL` - Logging level
- `JWT_SECRET` - JWT secret for authentication

## Testing

```bash
npm test
```

## Docker

```bash
# Build image
docker build -t collab/sync:dev .

# Run container
docker run -p 8081:8081 collab/sync:dev
```

## WebSocket Testing

You can test the WebSocket connection using wscat:

```bash
npm install -g wscat
wscat -c ws://localhost:8081
```
