# Unity Collaboration Platform - Project Setup Guide

## Quick Start

This guide will help you get the Unity Collaboration Platform running on your local machine.

## Prerequisites

### Required Software

```bash
# Go 1.21+
go version

# Node.js 20.x LTS
node --version
npm --version

# Docker & Docker Compose
docker --version
docker-compose --version

# PostgreSQL client (optional, for manual database access)
psql --version

# Redis CLI (optional, for manual cache access)
redis-cli --version
```

### Optional Tools

- **Make**: For running convenience commands
- **kubectl**: For Kubernetes deployment
- **Postman/Insomnia**: For API testing
- **wscat**: For WebSocket testing (`npm install -g wscat`)

## Installation Steps

### 1. Clone the Repository

```bash
git clone <repository-url>
cd unity-collab-platform
```

### 2. Start Infrastructure Services

Start all required databases and infrastructure services using Docker Compose:

```bash
# Start all services in detached mode
make dev-up

# Or manually:
docker-compose -f docker-compose.dev.yml up -d
```

This will start:
- **PostgreSQL** (port 5432)
- **Redis** (port 6379)
- **MongoDB** (port 27017)
- **RabbitMQ** (port 5672, management: 15672)
- **MinIO** (S3-compatible storage: port 9000, console: 9001)
- **TimescaleDB** (port 5433)
- **Prometheus** (port 9090)
- **Grafana** (port 3000)
- **Jaeger** (port 16686)

### 3. Initialize Database

Run database migrations and seed development data:

```bash
# Run migrations
make migrate-up

# Seed development data
make seed-dev
```

### 4. Install Service Dependencies

#### Go Services (Session, Asset, Auth)

```bash
# Install dependencies for all Go services
cd services/session && go mod download && cd ../..
cd services/asset && go mod download && cd ../..
cd services/auth && go mod download && cd ../..
```

#### Node.js Services (Sync, Gateway)

```bash
# Install Sync Service dependencies
cd services/sync && npm install && cd ../..

# Install Gateway dependencies
cd gateway && npm install && cd ../..
```

### 5. Run Services

Open multiple terminal windows/tabs and run each service:

#### Terminal 1: Session Service

```bash
make run-session
# Or manually:
cd services/session && go run cmd/server/main.go
```

#### Terminal 2: Sync Service

```bash
make run-sync
# Or manually:
cd services/sync && npm run dev
```

#### Terminal 3: Asset Service

```bash
make run-asset
# Or manually:
cd services/asset && go run cmd/server/main.go
```

#### Terminal 4: Auth Service

```bash
make run-auth
# Or manually:
cd services/auth && go run cmd/server/main.go
```

#### Terminal 5: API Gateway

```bash
make run-gateway
# Or manually:
cd gateway && npm run dev
```

## Verify Installation

### Check Service Health

```bash
# API Gateway
curl http://localhost:3000/health

# Session Service
curl http://localhost:8080/health

# Sync Service
curl http://localhost:8081/health

# Asset Service
curl http://localhost:8082/health

# Auth Service
curl http://localhost:8083/health
```

### Check Infrastructure Services

```bash
# PostgreSQL
docker exec -it collab_postgres psql -U dev -d collab_dev -c "SELECT COUNT(*) FROM users;"

# Redis
docker exec -it collab_redis redis-cli -a devpass ping

# MongoDB
docker exec -it collab_mongodb mongosh -u dev -p devpass --eval "db.version()"
```

## Service Ports

| Service | Port | URL |
|---------|------|-----|
| **API Gateway** | 3000 | http://localhost:3000 |
| **Session Service** | 8080 | http://localhost:8080 |
| **Sync Service** | 8081 | http://localhost:8081 (WS) |
| **Asset Service** | 8082 | http://localhost:8082 |
| **Auth Service** | 8083 | http://localhost:8083 |
| **PostgreSQL** | 5432 | postgresql://dev:devpass@localhost:5432/collab_dev |
| **Redis** | 6379 | redis://:devpass@localhost:6379 |
| **MongoDB** | 27017 | mongodb://dev:devpass@localhost:27017 |
| **RabbitMQ** | 5672 | amqp://dev:devpass@localhost:5672 |
| **RabbitMQ Management** | 15672 | http://localhost:15672 |
| **MinIO** | 9000 | http://localhost:9000 |
| **MinIO Console** | 9001 | http://localhost:9001 |
| **Prometheus** | 9090 | http://localhost:9090 |
| **Grafana** | 3000 | http://localhost:3000 (admin/admin) |
| **Jaeger** | 16686 | http://localhost:16686 |

## Testing the API

### Register a User

```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

### Login

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

### Create a Session

```bash
curl -X POST http://localhost:3000/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "name": "My Collaboration Session",
    "projectId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  }'
```

### Test WebSocket Connection

```bash
# Install wscat if you haven't
npm install -g wscat

# Connect to sync service
wscat -c ws://localhost:8081

# Send a message
> {"type": "ping"}

# You should receive a response
```

## Development Data

The seed script creates test users with the following credentials:

| Email | Password | Name |
|-------|----------|------|
| alice@example.com | password123 | Alice Johnson |
| bob@example.com | password123 | Bob Smith |
| charlie@example.com | password123 | Charlie Brown |
| diana@example.com | password123 | Diana Prince |

## Common Make Commands

```bash
# Start all infrastructure services
make dev-up

# Stop all services
make dev-down

# Restart all services
make dev-restart

# View service logs
make dev-logs

# Run database migrations
make migrate-up

# Rollback migrations
make migrate-down

# Seed development data
make seed-dev

# Access PostgreSQL shell
make db-shell

# Access Redis CLI
make redis-cli

# Access MongoDB shell
make mongo-shell

# Build all services
make build-all

# Run all tests
make test-all

# Clean build artifacts
make clean

# Full setup (infrastructure + migrations + dependencies)
make setup
```

## Troubleshooting

### Port Already in Use

If you see an error like "port already in use", you can either:

1. Stop the service using that port
2. Change the port in the service's environment variables
3. Kill the process: `lsof -ti:8080 | xargs kill -9` (replace 8080 with your port)

### Docker Services Not Starting

```bash
# Check Docker daemon status
docker ps

# Check service logs
docker-compose -f docker-compose.dev.yml logs <service-name>

# Restart a specific service
docker-compose -f docker-compose.dev.yml restart <service-name>

# Rebuild a service
docker-compose -f docker-compose.dev.yml up -d --build <service-name>
```

### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check PostgreSQL logs
docker logs collab_postgres

# Manually connect to verify
docker exec -it collab_postgres psql -U dev -d collab_dev
```

### Service Not Responding

1. Check if the service is running
2. Check the logs for errors
3. Verify environment variables are set correctly
4. Ensure all dependencies are installed
5. Try restarting the service

## Next Steps

- **API Documentation**: See [API_DESIGN.md](./API_DESIGN.md) for complete API reference
- **Architecture**: See [ARCHITECTURE.md](./ARCHITECTURE.md) for system architecture
- **Database Schema**: See [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) for database details
- **Deployment**: See [DEPLOYMENT.md](./DEPLOYMENT.md) for production deployment
- **Unity Plugin**: See [UNITY_PLUGIN.md](./UNITY_PLUGIN.md) for Unity integration

## Support

For issues and questions:
- GitHub Issues: <repository-url>/issues
- Documentation: See `/docs` folder
- Discord: Join our developer community

---

**Happy Coding! 🚀**
