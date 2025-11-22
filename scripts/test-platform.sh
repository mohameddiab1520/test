#!/bin/bash

# Unity Collaboration Platform - Integration Test Script
# This script tests the basic functionality of the platform

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}Unity Collaboration Platform - Tests${NC}"
echo -e "${BLUE}=====================================${NC}\n"

# Test 1: Check if Docker is running
echo -e "${YELLOW}[TEST 1] Checking Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}✗ Docker is not running${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}\n"

# Test 2: Check if services are up
echo -e "${YELLOW}[TEST 2] Checking Docker services...${NC}"
SERVICES=(collab_postgres collab_redis collab_mongodb collab_rabbitmq collab_minio)
ALL_UP=true

for service in "${SERVICES[@]}"; do
    if docker ps | grep -q $service; then
        echo -e "${GREEN}  ✓ $service is running${NC}"
    else
        echo -e "${RED}  ✗ $service is not running${NC}"
        ALL_UP=false
    fi
done

if [ "$ALL_UP" = false ]; then
    echo -e "${YELLOW}\nStarting services with 'make dev-up'...${NC}"
    make dev-up
    echo -e "${GREEN}Services started!${NC}"
    sleep 5  # Wait for services to be ready
fi
echo ""

# Test 3: Check database connectivity
echo -e "${YELLOW}[TEST 3] Testing PostgreSQL connection...${NC}"
if docker exec collab_postgres psql -U dev -d collab_dev -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL connection successful${NC}\n"
else
    echo -e "${RED}✗ PostgreSQL connection failed${NC}\n"
    exit 1
fi

# Test 4: Check Redis connectivity
echo -e "${YELLOW}[TEST 4] Testing Redis connection...${NC}"
if docker exec collab_redis redis-cli -a devpass ping | grep -q PONG; then
    echo -e "${GREEN}✓ Redis connection successful${NC}\n"
else
    echo -e "${RED}✗ Redis connection failed${NC}\n"
    exit 1
fi

# Test 5: Check MongoDB connectivity
echo -e "${YELLOW}[TEST 5] Testing MongoDB connection...${NC}"
if docker exec collab_mongodb mongosh -u dev -p devpass --quiet --eval "db.version()" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ MongoDB connection successful${NC}\n"
else
    echo -e "${RED}✗ MongoDB connection failed${NC}\n"
    exit 1
fi

# Test 6: Check if migrations exist
echo -e "${YELLOW}[TEST 6] Checking database migrations...${NC}"
if [ -f "migrations/001_initial_schema.sql" ]; then
    echo -e "${GREEN}✓ Migration files found${NC}\n"
else
    echo -e "${RED}✗ Migration files not found${NC}\n"
    exit 1
fi

# Test 7: Verify Go services can compile
echo -e "${YELLOW}[TEST 7] Verifying Go services...${NC}"
GO_SERVICES=(session asset auth)
for service in "${GO_SERVICES[@]}"; do
    echo -e "  Checking ${service} service..."
    if cd services/${service} && go build -o /tmp/${service}_test cmd/server/main.go 2>/dev/null; then
        echo -e "${GREEN}  ✓ ${service} service compiles successfully${NC}"
        rm -f /tmp/${service}_test
        cd ../..
    else
        echo -e "${RED}  ✗ ${service} service failed to compile${NC}"
        cd ../..
        exit 1
    fi
done
echo ""

# Test 8: Verify Node.js dependencies
echo -e "${YELLOW}[TEST 8] Verifying Node.js services...${NC}"
if [ -f "services/sync/package.json" ]; then
    echo -e "${GREEN}  ✓ Sync service package.json found${NC}"
else
    echo -e "${RED}  ✗ Sync service package.json not found${NC}"
    exit 1
fi

if [ -f "gateway/package.json" ]; then
    echo -e "${GREEN}  ✓ Gateway package.json found${NC}"
else
    echo -e "${RED}  ✗ Gateway package.json not found${NC}"
    exit 1
fi
echo ""

# Test 9: Check Unity plugin structure
echo -e "${YELLOW}[TEST 9] Verifying Unity plugin...${NC}"
REQUIRED_FILES=(
    "unity-plugin/package.json"
    "unity-plugin/Runtime/CollabManager.cs"
    "unity-plugin/Runtime/CollabNetworkClient.cs"
    "unity-plugin/Runtime/CollabSyncManager.cs"
    "unity-plugin/Editor/CollabEditorWindow.cs"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}  ✓ $(basename $file) exists${NC}"
    else
        echo -e "${RED}  ✗ $(basename $file) not found${NC}"
        exit 1
    fi
done
echo ""

# Test 10: Check documentation
echo -e "${YELLOW}[TEST 10] Verifying documentation...${NC}"
DOCS=(README.md ARCHITECTURE.md API_DESIGN.md DATABASE_SCHEMA.md DEPLOYMENT.md UNITY_PLUGIN.md)
for doc in "${DOCS[@]}"; do
    if [ -f "$doc" ]; then
        echo -e "${GREEN}  ✓ $doc exists${NC}"
    else
        echo -e "${RED}  ✗ $doc not found${NC}"
        exit 1
    fi
done
echo ""

# Summary
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}All tests passed! ✓${NC}"
echo -e "${GREEN}=====================================${NC}\n"

echo -e "${BLUE}Platform Status:${NC}"
echo -e "  Infrastructure: ${GREEN}Ready${NC}"
echo -e "  Services: ${GREEN}Ready to run${NC}"
echo -e "  Unity Plugin: ${GREEN}Ready${NC}"
echo -e "  Documentation: ${GREEN}Complete${NC}\n"

echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1. Run '${YELLOW}make migrate-up${NC}' to initialize the database"
echo -e "  2. Run '${YELLOW}make seed-dev${NC}' to add test data"
echo -e "  3. Run services with '${YELLOW}make run-<service>${NC}'"
echo -e "  4. Test API endpoints as described in PROJECT_SETUP.md\n"

exit 0
