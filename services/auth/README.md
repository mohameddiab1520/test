# Auth Service

Complete authentication and authorization service for Unity Collaboration Platform.

## Features

✅ **User Authentication**
- User registration with email validation
- Login with email/password
- JWT-based authentication
- Refresh token mechanism
- Secure logout

✅ **Security**
- Password hashing with bcrypt
- JWT token generation and validation
- Token refresh rotation
- Rate limiting
- CORS support

✅ **Database Integration**
- PostgreSQL for persistent data
- Redis for caching and session management
- Connection pooling
- Graceful shutdown

✅ **API Endpoints**

### Public Endpoints
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/validate` - Validate token

### Protected Endpoints (requires JWT)
- `POST /api/v1/auth/logout` - Logout user
- `GET /api/v1/auth/me` - Get current user info

### Health Checks
- `GET /health` - Service health check
- `GET /ready` - Readiness check (DB + Redis)

## Architecture

```
cmd/
  server/
    main.go                 # Application entry point
internal/
  config/
    config.go               # Configuration management
  database/
    postgres.go             # PostgreSQL connection
    redis.go                # Redis connection
  handler/
    auth_handler.go         # HTTP handlers
  middleware/
    auth.go                 # JWT authentication middleware
    ratelimit.go            # Rate limiting middleware
    cors.go                 # CORS middleware
  models/
    user.go                 # Data models
  repository/
    postgres.go             # User repository (PostgreSQL)
    refresh_token.go        # Refresh token repository
    redis.go                # Redis cache repository
    errors.go               # Repository errors
  service/
    auth_service.go         # Business logic
pkg/
  jwt/
    jwt.go                  # JWT utilities
    errors.go               # JWT errors
  password/
    password.go             # Password hashing utilities
    errors.go               # Password validation errors
```

## Configuration

### Environment Variables

```bash
# Server
SERVER_PORT=8083
SERVER_HOST=0.0.0.0
SERVER_MODE=release  # or debug

# PostgreSQL
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DBNAME=collab_auth
DATABASE_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-secret-key-change-this-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h  # 7 days
JWT_ISSUER=unity-collab-auth
```

## Running the Service

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### Development

```bash
# Install dependencies
go mod download

# Run service
go run cmd/server/main.go

# Or use environment-specific config
SERVER_MODE=debug go run cmd/server/main.go
```

### Production

```bash
# Build binary
go build -o auth-service cmd/server/main.go

# Run
./auth-service
```

### Docker

```bash
# Build image
docker build -t unity-collab/auth-service:latest .

# Run container
docker run -p 8083:8083 \
  -e DATABASE_HOST=postgres \
  -e REDIS_HOST=redis \
  unity-collab/auth-service:latest
```

## API Examples

### Register User

```bash
curl -X POST http://localhost:8083/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "johndoe",
    "password": "SecurePass123!",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

Response:
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 900,
  "tokenType": "Bearer",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "firstName": "John",
    "lastName": "Doe",
    "avatarUrl": ""
  }
}
```

### Login

```bash
curl -X POST http://localhost:8083/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

### Get Current User

```bash
curl -X GET http://localhost:8083/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Refresh Token

```bash
curl -X POST http://localhost:8083/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "YOUR_REFRESH_TOKEN"
  }'
```

### Logout

```bash
curl -X POST http://localhost:8083/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "YOUR_REFRESH_TOKEN"
  }'
```

## Security Features

### Password Requirements
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number

### Rate Limiting
- 10 requests per minute for auth endpoints
- IP-based for unauthenticated requests
- User-based for authenticated requests

### Token Expiry
- Access Token: 15 minutes
- Refresh Token: 7 days

## Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## Dependencies

- `github.com/gin-gonic/gin` - HTTP web framework
- `github.com/golang-jwt/jwt/v5` - JWT implementation
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `golang.org/x/crypto` - Bcrypt password hashing
- `go.uber.org/zap` - Structured logging
- `github.com/spf13/viper` - Configuration management

## Monitoring

The service provides:
- `/health` - Basic health check
- `/ready` - Readiness probe (checks DB and Redis)
- Structured JSON logs via Zap

## TODO

- [ ] OAuth 2.0 integration (Google, GitHub, Unity)
- [ ] Email verification flow
- [ ] Password reset flow
- [ ] Multi-factor authentication (MFA)
- [ ] Account lockout after failed attempts
- [ ] Password history
- [ ] Session management dashboard

## License

MIT
