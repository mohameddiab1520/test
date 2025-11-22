# Unity Collaboration Platform - Testing Guide

## Overview

This document provides comprehensive testing guidelines for the Unity Collaboration Platform, covering unit tests, integration tests, end-to-end tests, and manual testing procedures.

---

## Table of Contents

1. [Quick Testing](#quick-testing)
2. [Testing Strategy](#testing-strategy)
3. [Unit Testing](#unit-testing)
4. [Integration Testing](#integration-testing)
5. [End-to-End Testing](#end-to-end-testing)
6. [Performance Testing](#performance-testing)
7. [Security Testing](#security-testing)
8. [Manual Testing](#manual-testing)
9. [Continuous Integration](#continuous-integration)

---

## Quick Testing

### Verify Platform Structure

```bash
# Run structure verification
./scripts/verify-structure.sh
```

This checks that all required files and directories are in place.

### Run All Tests

```bash
# Run comprehensive test suite
make test-all
```

This runs tests for all services:
- Session Service tests
- Asset Service tests
- Auth Service tests
- Sync Service tests

---

## Testing Strategy

### Testing Pyramid

```
          /\
         /E2E\      ← Few (10%)
        /------\
       /  Intg  \   ← Some (30%)
      /----------\
     /    Unit    \ ← Many (60%)
    /--------------\
```

- **60% Unit Tests**: Fast, isolated tests for individual functions/methods
- **30% Integration Tests**: Test interactions between components
- **10% E2E Tests**: Full system tests simulating real user scenarios

### Coverage Requirements

- **Minimum Coverage**: 80% for all services
- **Critical Paths**: 95% coverage for authentication, session management, sync
- **New Code**: 100% coverage required for new features

---

## Unit Testing

### Go Services (Session, Asset, Auth)

#### Running Tests

```bash
# Test specific service
cd services/session
go test ./... -v -cover

# Test with coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Test with race detector
go test ./... -race
```

#### Test Structure

```go
// internal/handler/session_test.go
package handler_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateSession(t *testing.T) {
    // Arrange
    handler := NewSessionHandler()
    req := CreateSessionRequest{
        Name: "Test Session",
        ProjectID: "proj-123",
    }

    // Act
    session, err := handler.CreateSession(req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, session)
    assert.Equal(t, "Test Session", session.Name)
}
```

#### Mocking

```go
// Use interfaces for mocking
type SessionRepository interface {
    Create(session *Session) error
    FindByID(id string) (*Session, error)
}

// Mock implementation
type MockSessionRepo struct {
    mock.Mock
}

func (m *MockSessionRepo) Create(s *Session) error {
    args := m.Called(s)
    return args.Error(0)
}
```

### Node.js Services (Sync, Gateway)

#### Running Tests

```bash
# Test Sync Service
cd services/sync
npm test

# Test with coverage
npm test -- --coverage

# Watch mode
npm run test:watch
```

#### Test Structure

```typescript
// src/handlers/sync.test.ts
import { SyncHandler } from './sync';

describe('SyncHandler', () => {
  let handler: SyncHandler;

  beforeEach(() => {
    handler = new SyncHandler();
  });

  test('should broadcast message to all clients', async () => {
    // Arrange
    const message = { type: 'update', data: {} };

    // Act
    await handler.broadcast(message);

    // Assert
    expect(handler.getClientCount()).toBe(0);
  });
});
```

---

## Integration Testing

### Database Integration Tests

```go
// tests/integration/db_test.go
func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // Run tests
    t.Run("CreateUser", func(t *testing.T) {
        user := &User{Email: "test@example.com"}
        err := db.Create(user)
        assert.NoError(t, err)
    })
}
```

### Service-to-Service Integration

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
make test-integration

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### API Integration Tests

```bash
# Test complete API flow
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123","name":"Test"}'

# Verify response
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'
```

---

## End-to-End Testing

### WebSocket E2E Test

```typescript
// tests/e2e/websocket.test.ts
import WebSocket from 'ws';

describe('WebSocket E2E', () => {
  test('should connect and sync messages', async () => {
    const ws1 = new WebSocket('ws://localhost:8081');
    const ws2 = new WebSocket('ws://localhost:8081');

    await Promise.all([
      new Promise(resolve => ws1.on('open', resolve)),
      new Promise(resolve => ws2.on('open', resolve))
    ]);

    const message = { type: 'operation', data: { x: 10 } };
    ws1.send(JSON.stringify(message));

    const received = await new Promise(resolve => {
      ws2.on('message', data => resolve(JSON.parse(data)));
    });

    expect(received.type).toBe('sync');
  });
});
```

### Full Collaboration Flow

```bash
#!/bin/bash
# tests/e2e/collaboration-flow.sh

# 1. Register user
TOKEN=$(curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@test.com","password":"pass","name":"Alice"}' \
  | jq -r '.token')

# 2. Create session
SESSION_ID=$(curl -X POST http://localhost:3000/api/v1/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Session","projectId":"proj-1"}' \
  | jq -r '.id')

# 3. Join session
curl -X POST "http://localhost:3000/api/v1/sessions/$SESSION_ID/join" \
  -H "Authorization: Bearer $TOKEN"

# 4. Test WebSocket sync
wscat -c "ws://localhost:8081?token=$TOKEN" \
  -x '{"type":"operation","sessionId":"'$SESSION_ID'","data":{}}'
```

---

## Performance Testing

### Load Testing with k6

```javascript
// tests/performance/load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 100 },   // Ramp up to 100 users
    { duration: '5m', target: 100 },   // Stay at 100 users
    { duration: '1m', target: 0 },     // Ramp down to 0
  ],
  thresholds: {
    http_req_duration: ['p95<200'],    // 95% of requests < 200ms
    http_req_failed: ['rate<0.01'],    // < 1% errors
  },
};

export default function() {
  const res = http.get('http://localhost:3000/health');

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  sleep(1);
}
```

Run with:
```bash
k6 run tests/performance/load-test.js
```

### WebSocket Performance

```javascript
// tests/performance/websocket-test.js
import ws from 'k6/ws';
import { check } from 'k6';

export default function() {
  const url = 'ws://localhost:8081';

  const res = ws.connect(url, function(socket) {
    socket.on('open', () => {
      socket.send(JSON.stringify({ type: 'ping' }));
    });

    socket.on('message', (data) => {
      check(data, {
        'received pong': (d) => JSON.parse(d).type === 'pong',
      });
      socket.close();
    });

    socket.setTimeout(() => socket.close(), 10000);
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
```

### Database Performance

```sql
-- tests/performance/db-benchmark.sql

-- Test query performance
EXPLAIN ANALYZE
SELECT * FROM sessions
WHERE project_id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'
  AND created_at > NOW() - INTERVAL '7 days';

-- Should use index and complete in < 10ms
```

---

## Security Testing

### Authentication Tests

```bash
# Test without token (should fail)
curl -X GET http://localhost:3000/api/v1/sessions \
  -w "\nHTTP Status: %{http_code}\n"

# Expected: 401 Unauthorized

# Test with invalid token (should fail)
curl -X GET http://localhost:3000/api/v1/sessions \
  -H "Authorization: Bearer invalid-token" \
  -w "\nHTTP Status: %{http_code}\n"

# Expected: 401 Unauthorized
```

### SQL Injection Test

```bash
# Try SQL injection in email field
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com OR 1=1--","password":"test"}'

# Expected: Should be safely handled, not expose DB structure
```

### XSS Testing

```bash
# Try XSS in session name
curl -X POST http://localhost:3000/api/v1/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"<script>alert(1)</script>","projectId":"proj-1"}'

# Expected: Script should be escaped/sanitized
```

### Rate Limiting

```bash
# Attempt many requests rapidly
for i in {1..200}; do
  curl http://localhost:3000/api/v1/sessions &
done
wait

# Expected: Should see 429 Too Many Requests after limit
```

---

## Manual Testing

### Session Service Manual Tests

```bash
# 1. Health Check
curl http://localhost:8080/health

# 2. Create Session
curl -X POST http://localhost:3000/api/v1/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Manual Test Session",
    "projectId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  }'

# 3. Get Session
curl http://localhost:3000/api/v1/sessions/$SESSION_ID \
  -H "Authorization: Bearer $TOKEN"

# 4. Join Session
curl -X POST http://localhost:3000/api/v1/sessions/$SESSION_ID/join \
  -H "Authorization: Bearer $TOKEN"

# 5. Leave Session
curl -X POST http://localhost:3000/api/v1/sessions/$SESSION_ID/leave \
  -H "Authorization: Bearer $TOKEN"
```

### WebSocket Manual Test

```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8081

# Send messages
> {"type":"ping"}
> {"type":"operation","sessionId":"sess-123","data":{"x":10}}

# Verify responses
```

### Unity Plugin Manual Test

1. Open Unity 2022.3 LTS
2. Install the collaboration plugin
3. Open **Window → Collaboration**
4. Configure connection settings
5. Click **Connect**
6. Create/Join a session
7. Add `CollabSyncedObject` to a GameObject
8. Move the object and verify sync

---

## Continuous Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Test Session Service
        run: cd services/session && go test ./... -cover

      - name: Test Asset Service
        run: cd services/asset && go test ./... -cover

      - name: Test Auth Service
        run: cd services/auth && go test ./... -cover

  test-node:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Test Sync Service
        run: cd services/sync && npm install && npm test

      - name: Test Gateway
        run: cd gateway && npm install && npm test

  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
      redis:
        image: redis:7

    steps:
      - uses: actions/checkout@v3
      - name: Run Integration Tests
        run: make test-integration
```

---

## Test Coverage Reports

### Generate Coverage

```bash
# Go services
cd services/session
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Node.js services
cd services/sync
npm test -- --coverage
```

### View Coverage

```bash
# Go - Open in browser
open coverage.html

# Node.js - Coverage in terminal
npm test -- --coverage --verbose
```

---

## Testing Checklist

Before every release:

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] E2E tests pass
- [ ] Coverage > 80%
- [ ] No security vulnerabilities
- [ ] Performance benchmarks met
- [ ] Load tests successful (100+ concurrent users)
- [ ] WebSocket stress test passed
- [ ] Manual testing completed
- [ ] Unity plugin tested in Unity Editor

---

## Troubleshooting Tests

### Tests Failing Locally

```bash
# Clean and reinstall dependencies
make clean
make install-deps

# Restart test database
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.test.yml up -d

# Run tests again
make test-all
```

### Flaky Tests

- Increase timeouts
- Add proper waits for async operations
- Use test fixtures and mocks
- Ensure test isolation

### Database Tests Failing

```bash
# Reset test database
make migrate-down
make migrate-up
make seed-dev
```

---

## Resources

- **Testing Best Practices**: See `docs/testing-best-practices.md`
- **CI/CD Guide**: See [DEPLOYMENT.md](./DEPLOYMENT.md#cicd)
- **Performance Benchmarks**: See `docs/performance-benchmarks.md`

---

**Happy Testing!** 🧪✨
