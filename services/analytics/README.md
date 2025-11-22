# Analytics Service

Analytics and metrics collection service for Unity Collaboration Platform.

## Overview

The Analytics Service provides comprehensive metrics collection, analytics processing, and reporting capabilities for the collaboration platform.

## Features

- **Session Analytics**: Track collaboration session metrics
- **Project Analytics**: Aggregate project-level statistics
- **User Metrics**: User productivity and contribution tracking
- **Custom Reports**: Generate custom analytics reports
- **Performance Dashboard**: Real-time performance monitoring
- **Prometheus Integration**: Export metrics for monitoring

## Technology Stack

- **Framework**: FastAPI (Python 3.11+)
- **Databases**:
  - MongoDB (analytics data storage)
  - PostgreSQL (relational data)
  - Redis (caching)
- **Monitoring**: Prometheus metrics export
- **Libraries**: Pandas, NumPy for analytics

## API Endpoints

### Session Analytics
```
GET /api/v1/analytics/sessions/:sessionId
```

### Project Analytics
```
GET /api/v1/analytics/projects/:projectId?from=&to=
```

### User Metrics
```
GET /api/v1/analytics/users/:userId/metrics
```

### Custom Reports
```
POST /api/v1/analytics/custom-report
```

### Performance Dashboard
```
GET /api/v1/analytics/performance/dashboard
```

### Metrics Ingestion
```
POST /api/v1/analytics/ingest
```

## Configuration

Environment variables:

```bash
# Service configuration
SERVICE_NAME=analytics-service
SERVICE_PORT=8084
ENVIRONMENT=development

# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=collab_analytics

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=collab_dev
POSTGRES_USER=dev
POSTGRES_PASSWORD=devpass

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Analytics
METRICS_RETENTION_DAYS=30
AGGREGATION_INTERVAL_MINUTES=1
CACHE_TTL_SECONDS=300
```

## Development

### Install Dependencies
```bash
pip install -r requirements.txt
```

### Run Service
```bash
uvicorn main:app --host 0.0.0.0 --port 8084 --reload
```

### Run with Docker
```bash
docker build -t analytics-service .
docker run -p 8084:8084 analytics-service
```

## Metrics Exported

- `analytics_requests_total` - Total number of requests
- `analytics_query_duration_seconds` - Query duration histogram
- `analytics_data_points_ingested_total` - Data points ingested
- `analytics_active_sessions` - Active collaboration sessions

## Database Schema

### MongoDB Collections

**analytics**: Time-series analytics data
```javascript
{
  session_id: "sess_abc123",
  project_id: "proj_123",
  timestamp: ISODate(),
  metrics: { ... }
}
```

**session_analytics**: Aggregated session analytics
**user_metrics**: User productivity metrics

## Testing

```bash
pytest tests/ -v
```

## License

MIT
