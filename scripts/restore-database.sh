#!/bin/bash
# Database Restore Script for Unity Collaboration Platform

set -e

# Check arguments
if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <backup_date> <database_type>"
  echo "Example: $0 20250122_143000 postgres"
  echo "Database types: postgres, mongodb, redis"
  exit 1
fi

BACKUP_TIMESTAMP="$1"
DATABASE_TYPE="$2"

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/backups}"
S3_BUCKET="${S3_BUCKET:-unity-collab-backups}"

# PostgreSQL Configuration
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-collab_dev}"
POSTGRES_USER="${POSTGRES_USER:-dev}"

# MongoDB Configuration
MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
MONGODB_DB="${MONGODB_DB:-collab_analytics}"

echo "=== Starting restore at $(date) ==="

# Download from S3 if not local
if [ ! -f "$BACKUP_DIR/${DATABASE_TYPE}_${BACKUP_TIMESTAMP}.sql.gz" ] && [ ! -f "$BACKUP_DIR/${DATABASE_TYPE}_${BACKUP_TIMESTAMP}.tar.gz" ]; then
  echo "Backup not found locally, downloading from S3..."
  aws s3 cp "s3://$S3_BUCKET/" "$BACKUP_DIR/" \
    --recursive \
    --exclude "*" \
    --include "*${DATABASE_TYPE}_${BACKUP_TIMESTAMP}*"
fi

# Restore based on database type
case "$DATABASE_TYPE" in
  postgres)
    echo "Restoring PostgreSQL database..."
    BACKUP_FILE="$BACKUP_DIR/postgres_${BACKUP_TIMESTAMP}.sql.gz"

    if [ ! -f "$BACKUP_FILE" ]; then
      echo "Error: Backup file not found: $BACKUP_FILE"
      exit 1
    fi

    # Drop existing database (WARNING: This will delete all data!)
    echo "WARNING: This will drop the existing database!"
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
      echo "Restore cancelled"
      exit 0
    fi

    PGPASSWORD="$POSTGRES_PASSWORD" psql \
      -h "$POSTGRES_HOST" \
      -p "$POSTGRES_PORT" \
      -U "$POSTGRES_USER" \
      -d postgres \
      -c "DROP DATABASE IF EXISTS $POSTGRES_DB;"

    PGPASSWORD="$POSTGRES_PASSWORD" psql \
      -h "$POSTGRES_HOST" \
      -p "$POSTGRES_PORT" \
      -U "$POSTGRES_USER" \
      -d postgres \
      -c "CREATE DATABASE $POSTGRES_DB;"

    gunzip -c "$BACKUP_FILE" | PGPASSWORD="$POSTGRES_PASSWORD" pg_restore \
      -h "$POSTGRES_HOST" \
      -p "$POSTGRES_PORT" \
      -U "$POSTGRES_USER" \
      -d "$POSTGRES_DB" \
      --no-owner \
      --no-acl

    echo "PostgreSQL restore completed"
    ;;

  mongodb)
    echo "Restoring MongoDB database..."
    BACKUP_FILE="$BACKUP_DIR/mongodb_${BACKUP_TIMESTAMP}.tar.gz"

    if [ ! -f "$BACKUP_FILE" ]; then
      echo "Error: Backup file not found: $BACKUP_FILE"
      exit 1
    fi

    # Extract backup
    tar -xzf "$BACKUP_FILE" -C "$BACKUP_DIR"

    # Restore
    mongorestore \
      --uri="$MONGODB_URI" \
      --db="$MONGODB_DB" \
      --gzip \
      --drop \
      "$BACKUP_DIR/mongodb_${BACKUP_TIMESTAMP}/$MONGODB_DB"

    # Clean up
    rm -rf "$BACKUP_DIR/mongodb_${BACKUP_TIMESTAMP}"

    echo "MongoDB restore completed"
    ;;

  redis)
    echo "Restoring Redis database..."
    BACKUP_FILE="$BACKUP_DIR/redis_${BACKUP_TIMESTAMP}.rdb.gz"

    if [ ! -f "$BACKUP_FILE" ]; then
      echo "Error: Backup file not found: $BACKUP_FILE"
      exit 1
    fi

    gunzip -c "$BACKUP_FILE" > "$BACKUP_DIR/dump.rdb"

    echo "Please copy $BACKUP_DIR/dump.rdb to Redis data directory and restart Redis"
    ;;

  *)
    echo "Error: Unknown database type: $DATABASE_TYPE"
    echo "Valid types: postgres, mongodb, redis"
    exit 1
    ;;
esac

echo "=== Restore completed at $(date) ==="

# Send notification
if [ -n "$SLACK_WEBHOOK" ]; then
  curl -X POST "$SLACK_WEBHOOK" \
    -H 'Content-Type: application/json' \
    -d "{\"text\": \"✅ Database restore completed successfully at $(date)\"}"
fi
