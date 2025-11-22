# Quick Start Guide 🚀

Get started with Unity Collaboration Platform in 5 minutes!

## Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Unity 2022.3+ (for Unity Plugin)
- Make (optional, for convenience commands)

## 🏃 Quick Start

### 1. Start Development Environment

```bash
# Start all infrastructure services
make dev-up

# Or using docker-compose directly
docker-compose -f docker-compose.dev.yml up -d
```

This starts:
- ✅ PostgreSQL (port 5432)
- ✅ Redis (port 6379)
- ✅ MinIO/S3 (port 9000)
- ✅ MongoDB (port 27017)

### 2. Run Backend Services

**Option A: Using Make (Recommended)**
```bash
# Run all services in separate terminals
make run-auth      # Terminal 1
make run-session   # Terminal 2
make run-asset     # Terminal 3
```

**Option B: Using Go directly**
```bash
# Auth Service (port 8081)
cd services/auth && go run ./cmd/server

# Session Service (port 8082)
cd services/session && go run ./cmd/server

# Asset Service (port 8083)
cd services/asset && go run ./cmd/server
```

### 3. Test the API

```bash
# Health check
curl http://localhost:8081/health  # Auth Service
curl http://localhost:8082/health  # Session Service
curl http://localhost:8083/health  # Asset Service

# Register a user
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "Password123!",
    "firstName": "Test",
    "lastName": "User"
  }'

# Login
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Password123!"
  }'
# Save the accessToken from response

# Create a session
curl -X POST http://localhost:8082/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "projectId": "00000000-0000-0000-0000-000000000001",
    "name": "My First Session",
    "maxParticipants": 10,
    "isPublic": true,
    "voiceEnabled": true
  }'
```

## 🎮 Unity Plugin Setup

### 1. Import Plugin

Copy the `unity-plugin` folder into your Unity project:
```
Assets/
  Plugins/
    Collab/
      Runtime/
        CollabManager.cs
        CollabNetworkClient.cs
        CollabSessionManager.cs
        ...
```

### 2. Add Collab Manager to Scene

```csharp
using Collab.Unity;
using UnityEngine;

public class CollabExample : MonoBehaviour
{
    private CollabManager collabManager;

    async void Start()
    {
        // Initialize
        collabManager = gameObject.AddComponent<CollabManager>();
        
        var config = new CollabConfig
        {
            apiUrl = "http://localhost:8081",
            wsUrl = "http://localhost:8082",
            debugMode = true
        };
        
        collabManager.Initialize(config);
        
        // Authenticate
        await collabManager.Authenticate("test@example.com", "Password123!");
        
        // Create and join session
        await collabManager.CreateSession("My Session", "project-123");
        
        Debug.Log("Connected to collaboration session!");
    }
}
```

### 3. WebSocket Real-time Sync

```csharp
// Listen to events
collabManager.OnSessionJoined += (session) => {
    Debug.Log($"Joined session: {session.name}");
};

collabManager.OnParticipantJoined += (participant) => {
    Debug.Log($"User joined: {participant.userId}");
};

collabManager.OnObjectTransformUpdate += (objectId, transform) => {
    // Update object transform in Unity
    var obj = FindObjectById(objectId);
    if (obj != null) {
        obj.transform.position = transform.position;
        obj.transform.rotation = transform.rotation;
    }
};

// Broadcast changes
collabManager.BroadcastTransform(
    "player-001", 
    transform.position, 
    transform.rotation, 
    transform.localScale
);
```

## 🧪 Running Tests

```bash
# Run all tests
make test-all

# Run specific service tests
make test-auth
make test-session

# With coverage
make test-coverage
```

## 🐳 Docker Deployment

```bash
# Build Docker images
make docker-build-all

# Or build individually
docker build -t collab-auth:latest ./services/auth
docker build -t collab-session:latest ./services/session
docker build -t collab-asset:latest ./services/asset
```

## 📊 Monitoring

Access monitoring tools:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger (Tracing)**: http://localhost:16686
- **MinIO Console**: http://localhost:9001

## 🔧 Development Commands

```bash
make help              # Show all available commands
make build-all         # Build all services
make test-all          # Run all tests
make lint              # Run linter
make fmt               # Format code
make clean             # Clean build artifacts
make deps              # Download dependencies
make dev               # Start dev environment
```

## 🛠️ Troubleshooting

### Port already in use
```bash
# Find and kill process on port
lsof -ti:8081 | xargs kill -9  # Auth Service
lsof -ti:8082 | xargs kill -9  # Session Service
```

### Database connection error
```bash
# Reset database
make db-reset
```

### WebSocket connection failed
- Ensure Session Service is running on port 8082
- Check that you're using the correct session ID
- Verify JWT token is valid (not expired)

## 📚 Next Steps

- Read [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- Check [API_DESIGN.md](API_DESIGN.md) for API documentation
- See [TESTING.md](TESTING.md) for testing guide
- Review [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details.
