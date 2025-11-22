# Unity Collaboration Platform - API Design

## Table of Contents

1. [API Overview](#api-overview)
2. [Authentication API](#authentication-api)
3. [Session API](#session-api)
4. [Sync API (WebSocket)](#sync-api-websocket)
5. [Asset API](#asset-api)
6. [User API](#user-api)
7. [Project API](#project-api)
8. [Analytics API](#analytics-api)
9. [Webhook API](#webhook-api)
10. [Error Handling](#error-handling)
11. [Rate Limiting](#rate-limiting)

---

## API Overview

### Base URL
```
Production:  https://api.collab.example.com/v1
Staging:     https://api-staging.collab.example.com/v1
Development: http://localhost:8080/v1
```

### Authentication
All API requests (except `/auth/*`) require authentication via JWT bearer token:

```http
Authorization: Bearer <jwt_token>
```

### Response Format
All responses follow this structure:

**Success Response**
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2025-11-22T10:00:00Z",
    "requestId": "req_abc123"
  }
}
```

**Error Response**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameters",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format"
      }
    ]
  },
  "meta": {
    "timestamp": "2025-11-22T10:00:00Z",
    "requestId": "req_abc123"
  }
}
```

### Pagination
List endpoints support pagination:

```http
GET /api/v1/sessions?page=1&limit=20&sort=created_at&order=desc
```

**Pagination Response**
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "totalPages": 8,
    "hasNext": true,
    "hasPrev": false
  }
}
```

---

## Authentication API

### Register User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "name": "John Doe"
}
```

**Response 201 Created**
```json
{
  "success": true,
  "data": {
    "user": {
      "userId": "user_abc123",
      "email": "user@example.com",
      "name": "John Doe",
      "avatar": null,
      "emailVerified": false,
      "createdAt": "2025-11-22T10:00:00Z"
    },
    "tokens": {
      "accessToken": "eyJhbGciOiJSUzI1NiIs...",
      "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
      "tokenType": "Bearer",
      "expiresIn": 3600
    }
  }
}
```

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "user": { ... },
    "tokens": { ... }
  }
}
```

### OAuth Login

```http
GET /api/v1/auth/oauth/{provider}?redirect_uri=https://app.example.com/callback

Supported providers: google, github, unity
```

**Response 302 Redirect**
Redirects to OAuth provider authorization page

### OAuth Callback

```http
GET /api/v1/auth/oauth/{provider}/callback?code=abc123&state=xyz
```

**Response 302 Redirect**
Redirects to `redirect_uri` with tokens:
```
https://app.example.com/callback?access_token=...&refresh_token=...
```

### Refresh Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "tokenType": "Bearer",
    "expiresIn": 3600
  }
}
```

### Logout

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

### Verify Email

```http
POST /api/v1/auth/verify-email
Content-Type: application/json

{
  "token": "verification_token_abc123"
}
```

---

## Session API

### Create Session

```http
POST /api/v1/sessions
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "projectId": "proj_123456",
  "name": "Main Scene Collaboration",
  "settings": {
    "maxParticipants": 10,
    "isPublic": false,
    "voiceEnabled": true,
    "recordSession": false,
    "permissions": {
      "allowGuests": false,
      "defaultRole": "viewer"
    }
  }
}
```

**Response 201 Created**
```json
{
  "success": true,
  "data": {
    "session": {
      "sessionId": "sess_abc123",
      "projectId": "proj_123456",
      "ownerId": "user_xyz",
      "name": "Main Scene Collaboration",
      "settings": { ... },
      "status": "active",
      "participants": [],
      "createdAt": "2025-11-22T10:00:00Z",
      "updatedAt": "2025-11-22T10:00:00Z"
    },
    "wsUrl": "wss://sync.collab.example.com/ws?token=..."
  }
}
```

### Get Session

```http
GET /api/v1/sessions/:sessionId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "session": {
      "sessionId": "sess_abc123",
      "projectId": "proj_123456",
      "ownerId": "user_xyz",
      "name": "Main Scene Collaboration",
      "settings": { ... },
      "status": "active",
      "participants": [
        {
          "userId": "user_xyz",
          "name": "John Doe",
          "avatar": "https://...",
          "role": "owner",
          "status": "online",
          "joinedAt": "2025-11-22T10:00:00Z"
        }
      ],
      "createdAt": "2025-11-22T10:00:00Z",
      "updatedAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### List Sessions

```http
GET /api/v1/sessions?projectId=proj_123&status=active&page=1&limit=20
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "sessions": [
      {
        "sessionId": "sess_abc123",
        "name": "Main Scene Collaboration",
        "status": "active",
        "participantCount": 3,
        "createdAt": "2025-11-22T10:00:00Z"
      }
    ]
  },
  "pagination": { ... }
}
```

### Update Session

```http
PUT /api/v1/sessions/:sessionId
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "Updated Session Name",
  "settings": {
    "maxParticipants": 20
  }
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "session": { ... }
  }
}
```

### Delete Session

```http
DELETE /api/v1/sessions/:sessionId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Session deleted successfully"
  }
}
```

### Join Session

```http
POST /api/v1/sessions/:sessionId/join
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "wsUrl": "wss://sync.collab.example.com/ws?token=...",
    "sessionData": {
      "participants": [ ... ],
      "sceneState": { ... }
    }
  }
}
```

### Leave Session

```http
POST /api/v1/sessions/:sessionId/leave
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Left session successfully"
  }
}
```

### Invite to Session

```http
POST /api/v1/sessions/:sessionId/invites
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "emails": ["user@example.com"],
  "role": "editor",
  "message": "Join me in this collaboration session"
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "invites": [
      {
        "inviteId": "inv_123",
        "email": "user@example.com",
        "role": "editor",
        "status": "pending",
        "expiresAt": "2025-11-23T10:00:00Z"
      }
    ]
  }
}
```

### Update Participant Role

```http
PUT /api/v1/sessions/:sessionId/participants/:userId
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "role": "editor"
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "participant": {
      "userId": "user_456",
      "role": "editor",
      "updatedAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### Kick Participant

```http
DELETE /api/v1/sessions/:sessionId/participants/:userId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Participant removed successfully"
  }
}
```

---

## Sync API (WebSocket)

### WebSocket Connection

```
wss://sync.collab.example.com/ws?token=<jwt_token>
```

### Client → Server Messages

#### Operation (Scene Change)

```json
{
  "type": "operation",
  "sessionId": "sess_abc123",
  "operation": {
    "type": "update",
    "objectId": "obj_player_001",
    "path": "transform.position.x",
    "value": 10.5,
    "version": 42,
    "timestamp": 1700650000000
  }
}
```

#### Create Object

```json
{
  "type": "operation",
  "sessionId": "sess_abc123",
  "operation": {
    "type": "create",
    "objectId": "obj_new_001",
    "data": {
      "name": "New Cube",
      "transform": {
        "position": {"x": 0, "y": 0, "z": 0},
        "rotation": {"x": 0, "y": 0, "z": 0},
        "scale": {"x": 1, "y": 1, "z": 1}
      },
      "components": [
        {
          "type": "MeshRenderer",
          "properties": {
            "material": "mat_default"
          }
        }
      ]
    },
    "version": 43,
    "timestamp": 1700650000100
  }
}
```

#### Delete Object

```json
{
  "type": "operation",
  "sessionId": "sess_abc123",
  "operation": {
    "type": "delete",
    "objectId": "obj_old_001",
    "version": 44,
    "timestamp": 1700650000200
  }
}
```

#### Presence Update

```json
{
  "type": "presence",
  "sessionId": "sess_abc123",
  "presence": {
    "status": "online",
    "currentScene": "MainScene",
    "selectedObject": "obj_player_001",
    "cursorPosition": {"x": 100, "y": 200},
    "cameraTransform": {
      "position": {"x": 0, "y": 5, "z": -10},
      "rotation": {"x": 15, "y": 0, "z": 0}
    }
  }
}
```

#### Voice State

```json
{
  "type": "voice",
  "sessionId": "sess_abc123",
  "voice": {
    "muted": false,
    "deafened": false,
    "speaking": true
  }
}
```

#### Ping

```json
{
  "type": "ping",
  "timestamp": 1700650000000
}
```

### Server → Client Messages

#### Sync (Operation Broadcast)

```json
{
  "type": "sync",
  "operation": {
    "id": "op_xyz789",
    "type": "update",
    "objectId": "obj_player_001",
    "path": "transform.position.x",
    "value": 10.5,
    "version": 43,
    "userId": "user_abc",
    "userName": "John Doe",
    "timestamp": 1700650000100
  }
}
```

#### Acknowledgment

```json
{
  "type": "ack",
  "success": true,
  "version": 43,
  "operationId": "op_local_123"
}
```

#### Sync Required

```json
{
  "type": "sync_required",
  "reason": "version_mismatch",
  "currentVersion": 50,
  "yourVersion": 42,
  "missingOperations": [
    { ... },
    { ... }
  ]
}
```

#### Participant Joined

```json
{
  "type": "participant_joined",
  "participant": {
    "userId": "user_new",
    "name": "Jane Smith",
    "avatar": "https://...",
    "role": "editor",
    "joinedAt": "2025-11-22T10:00:00Z"
  }
}
```

#### Participant Left

```json
{
  "type": "participant_left",
  "userId": "user_old",
  "reason": "disconnected"
}
```

#### Presence Update

```json
{
  "type": "presence_update",
  "userId": "user_abc",
  "presence": {
    "status": "online",
    "currentScene": "MainScene",
    "selectedObject": "obj_player_001"
  }
}
```

#### Error

```json
{
  "type": "error",
  "error": {
    "code": "OPERATION_FAILED",
    "message": "Failed to apply operation",
    "details": { ... }
  }
}
```

#### Pong

```json
{
  "type": "pong",
  "timestamp": 1700650000000,
  "serverTime": 1700650000050
}
```

---

## Asset API

### Request Upload URL

```http
POST /api/v1/assets/upload-url
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "projectId": "proj_123",
  "fileName": "PlayerModel.fbx",
  "fileSize": 52428800,
  "contentType": "application/octet-stream",
  "metadata": {
    "category": "models",
    "tags": ["character", "player"]
  }
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "assetId": "asset_abc123",
    "uploadUrl": "https://s3.amazonaws.com/bucket/path?signature=...",
    "uploadMethod": "PUT",
    "uploadHeaders": {
      "Content-Type": "application/octet-stream"
    },
    "expiresIn": 3600
  }
}
```

### Confirm Upload

```http
POST /api/v1/assets/:assetId/confirm
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "md5": "5d41402abc4b2a76b9719d911017c592"
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "asset": {
      "assetId": "asset_abc123",
      "projectId": "proj_123",
      "fileName": "PlayerModel.fbx",
      "fileSize": 52428800,
      "contentType": "application/octet-stream",
      "status": "processing",
      "metadata": { ... },
      "createdAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### Get Asset

```http
GET /api/v1/assets/:assetId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "asset": {
      "assetId": "asset_abc123",
      "projectId": "proj_123",
      "fileName": "PlayerModel.fbx",
      "fileSize": 52428800,
      "contentType": "application/octet-stream",
      "status": "ready",
      "downloadUrl": "https://cdn.example.com/assets/...",
      "thumbnailUrl": "https://cdn.example.com/thumbnails/...",
      "metadata": {
        "category": "models",
        "tags": ["character", "player"],
        "dimensions": "1024x1024",
        "polyCount": 15000
      },
      "versions": [
        {
          "versionId": "ver_1",
          "versionNumber": 1,
          "createdAt": "2025-11-22T10:00:00Z"
        }
      ],
      "createdAt": "2025-11-22T10:00:00Z",
      "updatedAt": "2025-11-22T10:05:00Z"
    }
  }
}
```

### Download Asset

```http
GET /api/v1/assets/:assetId/download
Authorization: Bearer <jwt_token>
```

**Response 302 Redirect**
Redirects to CDN URL or presigned S3 URL

### List Assets

```http
GET /api/v1/assets?projectId=proj_123&category=models&page=1&limit=20
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "assets": [
      {
        "assetId": "asset_abc123",
        "fileName": "PlayerModel.fbx",
        "fileSize": 52428800,
        "thumbnailUrl": "https://...",
        "createdAt": "2025-11-22T10:00:00Z"
      }
    ]
  },
  "pagination": { ... }
}
```

### Update Asset Metadata

```http
PUT /api/v1/assets/:assetId/metadata
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "metadata": {
    "category": "characters",
    "tags": ["character", "player", "hero"],
    "description": "Main player character model"
  }
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "asset": { ... }
  }
}
```

### Delete Asset

```http
DELETE /api/v1/assets/:assetId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Asset deleted successfully"
  }
}
```

### Create Asset Version

```http
POST /api/v1/assets/:assetId/versions
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "fileName": "PlayerModel_v2.fbx",
  "fileSize": 55428800,
  "contentType": "application/octet-stream"
}
```

**Response 201 Created**
```json
{
  "success": true,
  "data": {
    "version": {
      "versionId": "ver_2",
      "versionNumber": 2,
      "uploadUrl": "https://...",
      "expiresIn": 3600
    }
  }
}
```

### Search Assets

```http
POST /api/v1/assets/search
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "query": "player character",
  "filters": {
    "projectId": "proj_123",
    "category": ["models", "textures"],
    "tags": ["player"],
    "fileSize": {
      "min": 0,
      "max": 104857600
    }
  },
  "sort": "relevance",
  "page": 1,
  "limit": 20
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "assets": [ ... ],
    "facets": {
      "categories": {
        "models": 15,
        "textures": 8
      },
      "tags": {
        "player": 20,
        "character": 18,
        "hero": 5
      }
    }
  },
  "pagination": { ... }
}
```

---

## User API

### Get Current User

```http
GET /api/v1/users/me
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "user": {
      "userId": "user_abc123",
      "email": "user@example.com",
      "name": "John Doe",
      "avatar": "https://...",
      "emailVerified": true,
      "mfaEnabled": false,
      "preferences": {
        "theme": "dark",
        "notifications": {
          "email": true,
          "push": false
        }
      },
      "createdAt": "2025-01-01T00:00:00Z",
      "lastLoginAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### Update User Profile

```http
PUT /api/v1/users/me
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "John Smith",
  "avatar": "https://...",
  "preferences": {
    "theme": "light"
  }
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "user": { ... }
  }
}
```

### Get User

```http
GET /api/v1/users/:userId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "user": {
      "userId": "user_abc123",
      "name": "John Doe",
      "avatar": "https://...",
      "createdAt": "2025-01-01T00:00:00Z"
    }
  }
}
```

### Change Password

```http
POST /api/v1/users/me/change-password
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "currentPassword": "OldPass123!",
  "newPassword": "NewPass456!"
}
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "message": "Password changed successfully"
  }
}
```

---

## Project API

### Create Project

```http
POST /api/v1/projects
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "My Unity Game",
  "description": "An awesome game project",
  "settings": {
    "unityVersion": "2022.3.15f1",
    "vcsProvider": "git",
    "vcsUrl": "https://github.com/user/repo.git"
  }
}
```

**Response 201 Created**
```json
{
  "success": true,
  "data": {
    "project": {
      "projectId": "proj_123456",
      "name": "My Unity Game",
      "description": "An awesome game project",
      "ownerId": "user_abc",
      "settings": { ... },
      "createdAt": "2025-11-22T10:00:00Z",
      "updatedAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### Get Project

```http
GET /api/v1/projects/:projectId
Authorization: Bearer <jwt_token>
```

### List Projects

```http
GET /api/v1/projects?page=1&limit=20
Authorization: Bearer <jwt_token>
```

### Update Project

```http
PUT /api/v1/projects/:projectId
Authorization: Bearer <jwt_token>
```

### Delete Project

```http
DELETE /api/v1/projects/:projectId
Authorization: Bearer <jwt_token>
```

### Add Project Member

```http
POST /api/v1/projects/:projectId/members
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "userId": "user_xyz",
  "role": "developer"
}
```

---

## Analytics API

### Get Session Analytics

```http
GET /api/v1/analytics/sessions/:sessionId
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "analytics": {
      "sessionId": "sess_abc123",
      "duration": 7200,
      "participantCount": 5,
      "operationCount": 1523,
      "topContributors": [
        {
          "userId": "user_abc",
          "name": "John Doe",
          "operationCount": 650
        }
      ],
      "objectsCreated": 23,
      "objectsDeleted": 5,
      "objectsModified": 145
    }
  }
}
```

### Get Project Analytics

```http
GET /api/v1/analytics/projects/:projectId?from=2025-11-01&to=2025-11-22
Authorization: Bearer <jwt_token>
```

**Response 200 OK**
```json
{
  "success": true,
  "data": {
    "analytics": {
      "projectId": "proj_123",
      "dateRange": {
        "from": "2025-11-01T00:00:00Z",
        "to": "2025-11-22T23:59:59Z"
      },
      "totalSessions": 45,
      "totalCollaborationTime": 129600,
      "activeUsers": 12,
      "assetCount": 234,
      "storageUsed": 5242880000,
      "sessionDurationAvg": 2880,
      "participantsPerSessionAvg": 3.5
    }
  }
}
```

---

## Webhook API

### Create Webhook

```http
POST /api/v1/webhooks
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "url": "https://myapp.com/webhook",
  "events": [
    "session.created",
    "session.ended",
    "participant.joined",
    "participant.left",
    "asset.uploaded"
  ],
  "secret": "my_webhook_secret",
  "active": true
}
```

**Response 201 Created**
```json
{
  "success": true,
  "data": {
    "webhook": {
      "webhookId": "hook_abc123",
      "url": "https://myapp.com/webhook",
      "events": [ ... ],
      "active": true,
      "createdAt": "2025-11-22T10:00:00Z"
    }
  }
}
```

### Webhook Payload Format

```json
{
  "event": "session.created",
  "timestamp": "2025-11-22T10:00:00Z",
  "data": {
    "session": {
      "sessionId": "sess_abc123",
      "projectId": "proj_123",
      "name": "Main Scene Collaboration",
      "ownerId": "user_xyz",
      "createdAt": "2025-11-22T10:00:00Z"
    }
  },
  "signature": "sha256=..."
}
```

### Webhook Signature Verification

```javascript
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  const digest = 'sha256=' + hmac.update(payload).digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(digest));
}
```

---

## Error Handling

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input parameters |
| `AUTHENTICATION_REQUIRED` | 401 | Missing or invalid authentication |
| `PERMISSION_DENIED` | 403 | Insufficient permissions |
| `RESOURCE_NOT_FOUND` | 404 | Requested resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g., duplicate) |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_SERVER_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

### Error Response Examples

**Validation Error**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameters",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format",
        "value": "invalid-email"
      },
      {
        "field": "password",
        "message": "Password must be at least 8 characters"
      }
    ]
  }
}
```

**Not Found**
```json
{
  "success": false,
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Session not found",
    "details": {
      "resource": "session",
      "id": "sess_nonexistent"
    }
  }
}
```

**Rate Limit**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests",
    "details": {
      "limit": 1000,
      "remaining": 0,
      "resetAt": "2025-11-22T11:00:00Z"
    }
  }
}
```

---

## Rate Limiting

### Rate Limit Headers

All API responses include rate limit headers:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1700654400
```

### Rate Limits by Tier

| Tier | Requests/Hour | Burst |
|------|---------------|-------|
| Free | 1,000 | 50 |
| Pro | 10,000 | 200 |
| Enterprise | 100,000 | 1,000 |

### Rate Limit by Endpoint

| Endpoint | Limit |
|----------|-------|
| `/auth/login` | 5/min |
| `/auth/register` | 3/min |
| `/sessions/*` | 100/min |
| `/assets/upload` | 50/hour |
| WebSocket connections | 10/min |

---

## API Versioning

### Version in URL
```
/api/v1/sessions
/api/v2/sessions
```

### Version in Header (Alternative)
```http
Accept: application/vnd.collab.v1+json
```

### Deprecation Notice
```http
X-API-Deprecation: This endpoint is deprecated and will be removed on 2026-01-01
X-API-Sunset: 2026-01-01
```

---

This API design provides a comprehensive, RESTful API with clear endpoints, request/response formats, and proper error handling for the Unity Collaboration Platform.
