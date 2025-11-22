# Build Service

Automated Unity build service for CI/CD automation.

## Overview

The Build Service provides automated Unity project builds with support for:
- Multiple Unity versions
- Cross-platform builds (Windows, Linux, macOS, Android, iOS, WebGL)
- Build queuing and execution
- Artifact management
- Build logs streaming
- Test execution

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL
- **Queue**: Redis
- **Storage**: AWS S3 / MinIO

## Features

- ✅ Automated builds triggered by commits
- ✅ Multiple Unity versions support
- ✅ Cross-platform builds
- ✅ Build queue management
- ✅ Real-time build logs
- ✅ Artifact storage and distribution
- ✅ Build caching
- ✅ Test execution

## API Endpoints

```
POST   /api/v1/builds              - Create new build
GET    /api/v1/builds              - List builds
GET    /api/v1/builds/:id          - Get build details
POST   /api/v1/builds/:id/cancel   - Cancel build
GET    /api/v1/builds/:id/logs     - Get build logs
GET    /api/v1/builds/:id/artifacts - Get build artifacts
```

## Running Locally

```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env

# Run service
go run cmd/server/main.go
```

## Docker

```bash
docker build -t collab/build:dev .
docker run -p 8087:8087 --env-file .env collab/build:dev
```

## Configuration

See `.env.example` for all configuration options.

Key settings:
- `UNITY_PATH` - Path to Unity installations
- `MAX_CONCURRENT_BUILDS` - Maximum parallel builds
- `BUILD_TIMEOUT` - Build timeout in seconds
- `S3_BUCKET` - Artifact storage bucket
