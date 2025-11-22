# Conflict Resolution Service

Intelligent conflict resolution for concurrent edits in Unity collaboration.

## Overview

Handles conflict detection and resolution using various strategies:
- Last-write-wins
- Three-way merge
- Operational Transform
- Manual resolution with preview
- Automatic conflict resolution

## Features

- ✅ Multiple resolution strategies
- ✅ Conflict preview and simulation
- ✅ Automatic conflict detection
- ✅ Manual resolution UI
- ✅ Conflict history tracking
- ✅ Rollback support

## API Endpoints

```
POST   /api/v1/conflicts/detect    - Detect conflicts
POST   /api/v1/conflicts/resolve   - Resolve conflict
GET    /api/v1/conflicts/:id       - Get conflict details
GET    /api/v1/conflicts           - List conflicts
POST   /api/v1/conflicts/:id/preview - Preview resolution
```

## Running

```bash
go run cmd/server/main.go
```
