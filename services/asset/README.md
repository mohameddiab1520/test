# Asset Service

Asset management microservice for Unity Collaboration Platform.

## Overview

The Asset Service handles all asset-related operations including:
- Asset upload and download
- Asset versioning and history
- Asset metadata management
- S3/MinIO storage integration
- Asset sharing and permissions

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL
- **Storage**: AWS S3 / MinIO
- **Cache**: Redis (optional)

## Features

- ✅ Multi-part file upload
- ✅ Asset versioning
- ✅ Thumbnail generation
- ✅ Asset metadata extraction
- ✅ Access control and permissions
- ✅ Search and filtering
- ✅ Asset compression

## API Endpoints

### Asset Management

```
POST   /api/v1/assets              - Upload new asset
GET    /api/v1/assets              - List all assets
GET    /api/v1/assets/:id          - Get asset details
PUT    /api/v1/assets/:id          - Update asset metadata
DELETE /api/v1/assets/:id          - Delete asset
GET    /api/v1/assets/:id/download - Download asset
GET    /api/v1/assets/:id/versions - Get asset versions
```

### Asset Versions

```
POST   /api/v1/assets/:id/versions - Create new version
GET    /api/v1/assets/:id/versions/:version - Get specific version
```

## Running Locally

```bash
# Install dependencies
go mod download

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Run the service
go run cmd/server/main.go

# Or use Make
make run-asset
```

## Environment Variables

See `.env.example` for required configuration.

Key variables:
- `PORT` - Service port (default: 8082)
- `DB_*` - PostgreSQL connection settings
- `AWS_*` - S3 configuration
- `S3_BUCKET` - S3 bucket name
- `MAX_FILE_SIZE` - Maximum file size for uploads

## Storage Configuration

### AWS S3

```bash
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
S3_BUCKET=unity-collab-assets
```

### MinIO (Local Development)

```bash
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin
S3_BUCKET=assets
S3_ENDPOINT=http://localhost:9000
```

## Docker

```bash
# Build image
docker build -t collab/asset:dev .

# Run container
docker run -p 8082:8082 --env-file .env collab/asset:dev
```

## Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## Database Schema

The Asset Service uses the following main tables:
- `assets` - Asset metadata
- `asset_versions` - Asset version history
- `asset_permissions` - Access control

See `migrations/` directory for schema definitions.
