# API Gateway

Central API Gateway for Unity Collaboration Platform.

## Overview

The API Gateway serves as the single entry point for all client requests, providing:
- Request routing to microservices
- Authentication and authorization
- Rate limiting
- CORS handling
- Request/response logging
- Error handling
- WebSocket proxying

## Tech Stack

- **Runtime**: Node.js 20+
- **Language**: TypeScript
- **Framework**: Express.js
- **Proxy**: http-proxy-middleware

## Features

- ✅ JWT-based authentication
- ✅ Service-to-service routing
- ✅ Rate limiting per endpoint
- ✅ CORS configuration
- ✅ Security headers (Helmet)
- ✅ Request logging
- ✅ Error handling
- ✅ Health checks
- ✅ WebSocket support

## Architecture

```
Client Request
    ↓
API Gateway (Port 3000)
    ↓
┌───────────┬──────────┬──────────┬──────────┐
│ Auth      │ Session  │ Asset    │ Sync     │
│ Service   │ Service  │ Service  │ Service  │
│ :8083     │ :8080    │ :8082    │ :8081    │
└───────────┴──────────┴──────────┴──────────┘
```

## Routes

### Public Routes (No Authentication)

```
GET  /health                      - Gateway health check
GET  /metrics                     - Gateway metrics
POST /api/v1/auth/register        - User registration
POST /api/v1/auth/login           - User login
POST /api/v1/auth/refresh         - Refresh access token
```

### Protected Routes (Requires Authentication)

```
# Sessions
GET    /api/v1/sessions           - List sessions
POST   /api/v1/sessions           - Create session
GET    /api/v1/sessions/:id       - Get session details
DELETE /api/v1/sessions/:id       - Delete session

# Assets
GET    /api/v1/assets             - List assets
POST   /api/v1/assets             - Upload asset
GET    /api/v1/assets/:id         - Get asset
DELETE /api/v1/assets/:id         - Delete asset

# Presence
GET    /api/v1/presence           - Get presence info
POST   /api/v1/presence/update    - Update presence

# Voice
POST   /api/v1/voice/join         - Join voice channel
POST   /api/v1/voice/leave        - Leave voice channel

# Analytics
GET    /api/v1/analytics/stats    - Get analytics
POST   /api/v1/analytics/events   - Track event

# WebSocket
WS     /ws                        - Sync service WebSocket
```

## Running Locally

```bash
# Install dependencies
npm install

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Development mode
npm run dev

# Build for production
npm run build

# Run production build
npm start

# Or use Make
make run-gateway
```

## Environment Variables

See `.env.example` for required configuration.

Key variables:
- `PORT` - Gateway port (default: 3000)
- `JWT_SECRET` - Secret for JWT verification
- `*_SERVICE_URL` - URLs for backend services
- `RATE_LIMIT_*` - Rate limiting configuration
- `CORS_*` - CORS configuration

## Authentication

The gateway uses JWT tokens for authentication. Protected routes require a valid JWT token in the Authorization header:

```
Authorization: Bearer <token>
```

The gateway validates the token and forwards user information to backend services via headers:
- `X-User-Id` - User ID
- `X-User-Email` - User email
- `X-User-Role` - User role

## Rate Limiting

Default rate limits:
- 100 requests per 15 minutes per IP
- Applied to all `/api/*` routes

Configure via environment variables:
```bash
RATE_LIMIT_WINDOW_MS=900000    # 15 minutes
RATE_LIMIT_MAX=100             # Max requests
```

## Docker

```bash
# Build image
docker build -t collab/gateway:dev .

# Run container
docker run -p 3000:3000 --env-file .env collab/gateway:dev
```

## Health Checks

```bash
# Gateway health
curl http://localhost:3000/health

# Metrics
curl http://localhost:3000/metrics
```

## Development

```bash
# Watch mode
npm run dev

# Lint
npm run lint

# Format
npm run format

# Type check
npx tsc --noEmit
```

## Error Handling

The gateway provides consistent error responses:

```json
{
  "error": "Error Type",
  "message": "Error description",
  "timestamp": "2025-11-22T10:30:00.000Z"
}
```

Common status codes:
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error
- `503` - Service Unavailable

## Security

- Helmet.js for security headers
- CORS configuration
- Rate limiting
- JWT validation
- Request size limits (50MB)

## Monitoring

The gateway logs all requests with:
- Request method and path
- Response status code
- Response time
- User info (if authenticated)
- IP address
