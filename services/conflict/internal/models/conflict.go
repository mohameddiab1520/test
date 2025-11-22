package models

import (
	"time"
)

// Conflict represents a resource conflict in the database
type Conflict struct {
	ID                 string          `json:"id" db:"id"`
	ProjectID          string          `json:"project_id" db:"project_id"`
	ResourceID         string          `json:"resource_id" db:"resource_id"`
	ResourceType       ResourceType    `json:"resource_type" db:"resource_type"`
	ConflictType       ConflictType    `json:"conflict_type" db:"conflict_type"`
	Status             ConflictStatus  `json:"status" db:"status"`
	ResolvedBy         *string         `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolutionStrategy *string         `json:"resolution_strategy,omitempty" db:"resolution_strategy"`
	DetectedAt         time.Time       `json:"detected_at" db:"detected_at"`
	ResolvedAt         *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// ConflictVersion represents a version involved in a conflict
type ConflictVersion struct {
	ID         string                 `json:"id" db:"id"`
	ConflictID string                 `json:"conflict_id" db:"conflict_id"`
	VersionID  string                 `json:"version_id" db:"version_id"`
	UserID     string                 `json:"user_id" db:"user_id"`
	UserName   string                 `json:"user_name" db:"user_name"`
	Content    []byte                 `json:"content" db:"content"`
	Hash       string                 `json:"hash" db:"hash"`
	Timestamp  time.Time              `json:"timestamp" db:"timestamp"`
	Changes    map[string]interface{} `json:"changes,omitempty" db:"changes"`
}

// ResourceType enum
type ResourceType string

const (
	ResourceTypeUnknown  ResourceType = "unknown"
	ResourceTypeScene    ResourceType = "scene"
	ResourceTypePrefab   ResourceType = "prefab"
	ResourceTypeScript   ResourceType = "script"
	ResourceTypeAsset    ResourceType = "asset"
	ResourceTypeSettings ResourceType = "settings"
	ResourceTypeMaterial ResourceType = "material"
	ResourceTypeTexture  ResourceType = "texture"
)

// ConflictType enum
type ConflictType string

const (
	ConflictTypeUnknown    ConflictType = "unknown"
	ConflictTypeEditEdit   ConflictType = "edit_edit"
	ConflictTypeDeleteEdit ConflictType = "delete_edit"
	ConflictTypeMoveEdit   ConflictType = "move_edit"
	ConflictTypeMerge      ConflictType = "merge"
	ConflictTypeVersion    ConflictType = "version"
)

// ConflictStatus enum
type ConflictStatus string

const (
	ConflictStatusUnknown   ConflictStatus = "unknown"
	ConflictStatusDetected  ConflictStatus = "detected"
	ConflictStatusPending   ConflictStatus = "pending"
	ConflictStatusResolving ConflictStatus = "resolving"
	ConflictStatusResolved  ConflictStatus = "resolved"
	ConflictStatusFailed    ConflictStatus = "failed"
)

// ResolutionStrategy enum
type ResolutionStrategy string

const (
	ResolutionStrategyUnknown        ResolutionStrategy = "unknown"
	ResolutionStrategyManual         ResolutionStrategy = "manual"
	ResolutionStrategyAcceptTheirs   ResolutionStrategy = "accept_theirs"
	ResolutionStrategyAcceptMine     ResolutionStrategy = "accept_mine"
	ResolutionStrategyAutoMerge      ResolutionStrategy = "auto_merge"
	ResolutionStrategyLastWriteWins  ResolutionStrategy = "last_write_wins"
	ResolutionStrategyFirstWriteWins ResolutionStrategy = "first_write_wins"
)
