# Unity Collaboration Platform - Quick Start Guide

## 🚀 Get Started in 5 Minutes

This guide will help you get the Unity Collaboration Platform up and running quickly.

---

## Prerequisites Check

Before starting, ensure you have:

```bash
# Check Go version (need 1.21+)
go version

# Check Node.js version (need 20.x+)
node --version

# Check Docker
docker --version
docker-compose --version
```

---

## Step 1: Start Infrastructure (2 minutes)

Start all required services using Docker Compose:

```bash
# Start PostgreSQL, Redis, MongoDB, RabbitMQ, MinIO, etc.
make dev-up
```

This will start:
- **PostgreSQL** on port 5432
- **Redis** on port 6379
- **MongoDB** on port 27017
- **RabbitMQ** on port 5672 (Management UI: 15672)
- **MinIO** on port 9000 (Console: 9001)
- **Prometheus** on port 9090
- **Grafana** on port 3000
- **Jaeger** on port 16686

---

## Step 2: Initialize Database (1 minute)

Run migrations and seed test data:

```bash
# Create database schema
make migrate-up

# Add test users and projects
make seed-dev
```

**Test Users Created:**
- alice@example.com / password123
- bob@example.com / password123
- charlie@example.com / password123
- diana@example.com / password123

---

## Step 3: Install Service Dependencies (1 minute)

```bash
# Install all dependencies
make install-deps
```

Or manually:

```bash
# Go services
cd services/session && go mod download && cd ../..
cd services/asset && go mod download && cd ../..
cd services/auth && go mod download && cd ../..

# Node.js services
cd services/sync && npm install && cd ../..
cd gateway && npm install && cd ../..
```

---

## Step 4: Start Services (1 minute)

Open 5 terminal windows and run:

**Terminal 1 - Session Service:**
```bash
make run-session
# Runs on http://localhost:8080
```

**Terminal 2 - Sync Service:**
```bash
make run-sync
# Runs on http://localhost:8081 (WebSocket)
```

**Terminal 3 - Asset Service:**
```bash
make run-asset
# Runs on http://localhost:8082
```

**Terminal 4 - Auth Service:**
```bash
make run-auth
# Runs on http://localhost:8083
```

**Terminal 5 - API Gateway:**
```bash
make run-gateway
# Runs on http://localhost:3000
```

---

## Step 5: Test the Platform

### Test 1: Run Platform Tests

```bash
# Run comprehensive test suite
./scripts/test-platform.sh
```

### Test 2: Check Service Health

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

### Test 3: Login with Test User

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

You should receive a JWT token in the response.

### Test 4: Create a Collaboration Session

```bash
# Replace <TOKEN> with the token from previous step
curl -X POST http://localhost:3000/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "name": "My First Session",
    "projectId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  }'
```

### Test 5: Test WebSocket Connection

```bash
# Install wscat if not installed
npm install -g wscat

# Connect to Sync Service
wscat -c ws://localhost:8081

# Send a test message
> {"type": "ping", "data": "hello"}

# You should receive acknowledgment and see the message broadcast
```

---

## Install Unity Plugin

### Option 1: From Package Manager (Recommended)

1. Open Unity 2022.3 LTS or higher
2. Go to **Window → Package Manager**
3. Click **+** → **Add package from git URL**
4. Enter: `file:///absolute/path/to/unity-plugin`
5. Click **Add**

### Option 2: Manual Installation

1. Copy the `unity-plugin` folder to your Unity project's `Packages` folder
2. Unity will automatically detect and import the package

### Configure the Plugin

1. In Unity, go to **Window → Collaboration**
2. The Collaboration window will open
3. Configure your connection settings:
   - **Server URL:** `http://localhost:3000`
   - **WebSocket URL:** `ws://localhost:8081`
   - **Email:** `alice@example.com`
   - **Password:** `password123`
4. Click **Connect**
5. You're now ready to collaborate!

---

## Verify Installation

Check that everything is working:

```bash
# Check all Docker services
make dev-ps

# Check database
make db-shell
# Then run: SELECT COUNT(*) FROM users;
# Should show 4 test users

# Check Redis
make redis-cli
# Then run: PING
# Should return PONG

# Check MongoDB
make mongo-shell
# Should connect successfully
```

---

## What's Next?

### Development

- **Read Architecture:** [ARCHITECTURE.md](./ARCHITECTURE.md)
- **API Reference:** [API_DESIGN.md](./API_DESIGN.md)
- **Database Schema:** [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md)
- **Unity Plugin Guide:** [UNITY_PLUGIN.md](./UNITY_PLUGIN.md)

### Testing

```bash
# Run all tests
make test-all

# Run specific service tests
make test-session
make test-asset
make test-auth
make test-sync
```

### Building

```bash
# Build all services
make build-all

# Build Docker images
make docker-build-all
```

### Deployment

When ready to deploy to production, see [DEPLOYMENT.md](./DEPLOYMENT.md) for:
- Kubernetes setup
- CI/CD pipeline configuration
- Production environment variables
- Monitoring and logging setup

---

## Common Issues

### Port Already in Use

If you see "port already in use" errors:

```bash
# Kill process on specific port (replace 8080 with your port)
lsof -ti:8080 | xargs kill -9
```

### Docker Services Won't Start

```bash
# Check Docker daemon
docker ps

# View logs
make dev-logs

# Restart services
make dev-restart
```

### Database Connection Failed

```bash
# Check PostgreSQL status
docker logs collab_postgres

# Restart PostgreSQL
docker-compose -f docker-compose.dev.yml restart collab_postgres

# Verify connection
docker exec -it collab_postgres psql -U dev -d collab_dev
```

### Service Won't Compile

```bash
# For Go services
cd services/<service-name>
go mod tidy
go build cmd/server/main.go

# For Node.js services
cd services/<service-name>
rm -rf node_modules package-lock.json
npm install
```

---

## Clean Up

When you're done:

```bash
# Stop all services
make dev-down

# Clean build artifacts
make clean

# Full cleanup (including Docker volumes)
make clean-all
```

---

## Getting Help

- **Documentation:** Check the `docs/` folder
- **Examples:** See API examples in [API_DESIGN.md](./API_DESIGN.md)
- **Issues:** GitHub Issues (if applicable)
- **Discord:** Join our developer community

---

## Summary

You now have:

✅ Infrastructure running (PostgreSQL, Redis, MongoDB, etc.)
✅ Database initialized with test data
✅ All microservices ready to run
✅ Unity plugin ready for installation
✅ Complete API documentation

**Start collaborating in Unity!** 🎮✨

---

*For detailed information, see the comprehensive documentation in the repository.*
