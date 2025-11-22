# Session Service

Session management microservice for Unity Collaboration Platform.

## Overview

The Session Service manages collaboration sessions, including:
- Creating and managing collaboration sessions
- Session participant management
- Session state tracking
- Real-time presence updates

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL + Redis
- **Protocol**: REST API + gRPC

## Running Locally

```bash
# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go

# Or use Make
make run-session
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/sessions` - List sessions
- `POST /api/v1/sessions` - Create session
- `GET /api/v1/sessions/:id` - Get session details
- `PUT /api/v1/sessions/:id` - Update session
- `DELETE /api/v1/sessions/:id` - Delete session
- `POST /api/v1/sessions/:id/join` - Join session
- `POST /api/v1/sessions/:id/leave` - Leave session

## Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Testing

```bash
go test ./... -v -cover
```

## Docker

```bash
# Build image
docker build -t collab/session:dev .

# Run container
docker run -p 8080:8080 collab/session:dev
```
