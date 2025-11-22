#!/bin/bash
# Database Backup Script for Unity Collaboration Platform

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/backups}"
S3_BUCKET="${S3_BUCKET:-unity-collab-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# PostgreSQL Configuration
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-collab_dev}"
POSTGRES_USER="${POSTGRES_USER:-dev}"

# MongoDB Configuration
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
MONGODB_DB="${MONGODB_DB:-collab_analytics}"

# Create backup directory
mkdir -p "$BACKUP_DIR"

echo "=== Starting backup at $(date) ==="

# Backup PostgreSQL
echo "Backing up PostgreSQL database..."
POSTGRES_BACKUP_FILE="$BACKUP_DIR/postgres_${TIMESTAMP}.sql.gz"
PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
  -h "$POSTGRES_HOST" \
  -p "$POSTGRES_PORT" \
  -U "$POSTGRES_USER" \
  -d "$POSTGRES_DB" \
  --format=custom \
  | gzip > "$POSTGRES_BACKUP_FILE"

echo "PostgreSQL backup completed: $POSTGRES_BACKUP_FILE"

# Backup MongoDB
echo "Backing up MongoDB database..."
MONGODB_BACKUP_DIR="$BACKUP_DIR/mongodb_${TIMESTAMP}"
mongodump \
  --uri="$MONGODB_URI" \
  --db="$MONGODB_DB" \
  --out="$MONGODB_BACKUP_DIR" \
  --gzip

tar -czf "$BACKUP_DIR/mongodb_${TIMESTAMP}.tar.gz" -C "$BACKUP_DIR" "mongodb_${TIMESTAMP}"
rm -rf "$MONGODB_BACKUP_DIR"

echo "MongoDB backup completed: $BACKUP_DIR/mongodb_${TIMESTAMP}.tar.gz"

# Backup Redis (if needed)
if [ -n "$REDIS_HOST" ]; then
  echo "Backing up Redis..."
  REDIS_BACKUP_FILE="$BACKUP_DIR/redis_${TIMESTAMP}.rdb"
  redis-cli -h "$REDIS_HOST" -p "${REDIS_PORT:-6379}" --rdb "$REDIS_BACKUP_FILE"
  gzip "$REDIS_BACKUP_FILE"
  echo "Redis backup completed: ${REDIS_BACKUP_FILE}.gz"
fi

# Upload to S3
if [ -n "$S3_BUCKET" ]; then
  echo "Uploading backups to S3..."
  aws s3 cp "$BACKUP_DIR" "s3://$S3_BUCKET/$(date +%Y/%m/%d)/" \
    --recursive \
    --exclude "*" \
    --include "*_${TIMESTAMP}.*" \
    --storage-class STANDARD_IA
  echo "Upload to S3 completed"
fi

# Clean up old local backups
echo "Cleaning up old local backups (older than $RETENTION_DAYS days)..."
find "$BACKUP_DIR" -name "*.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "=== Backup completed at $(date) ==="

# Send notification (optional)
if [ -n "$SLACK_WEBHOOK" ]; then
  curl -X POST "$SLACK_WEBHOOK" \
    -H 'Content-Type: application/json' \
    -d "{\"text\": \"✅ Database backup completed successfully at $(date)\"}"
fi
