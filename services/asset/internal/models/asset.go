package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// AssetStatus represents the status of an asset
type AssetStatus string

const (
	AssetStatusUploading  AssetStatus = "uploading"
	AssetStatusProcessing AssetStatus = "processing"
	AssetStatusReady      AssetStatus = "ready"
	AssetStatusFailed     AssetStatus = "failed"
	AssetStatusDeleted    AssetStatus = "deleted"
)

// Asset represents an uploaded asset
type Asset struct {
	ID             string                 `json:"id" db:"asset_id"`
	ProjectID      string                 `json:"projectId" db:"project_id"`
	UserID         string                 `json:"userId" db:"user_id"`
	FileName       string                 `json:"fileName" db:"file_name"`
	FileSize       int64                  `json:"fileSize" db:"file_size"`
	ContentType    string                 `json:"contentType" db:"content_type"`
	MD5Hash        string                 `json:"md5Hash" db:"md5_hash"`
	StoragePath    string                 `json:"storagePath" db:"storage_path"`
	CDNURL         string                 `json:"cdnUrl,omitempty" db:"cdn_url"`
	ThumbnailURL   string                 `json:"thumbnailUrl,omitempty" db:"thumbnail_url"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Category       string                 `json:"category" db:"category"`
	Tags           []string               `json:"tags" db:"tags"`
	Status         AssetStatus            `json:"status" db:"status"`
	ErrorMessage   string                 `json:"errorMessage,omitempty" db:"error_message"`
	VersionNumber  int                    `json:"versionNumber" db:"version_number"`
	ParentAssetID  *string                `json:"parentAssetId,omitempty" db:"parent_asset_id"`
	CreatedAt      time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time              `json:"updatedAt" db:"updated_at"`
	DeletedAt      *time.Time             `json:"deletedAt,omitempty" db:"deleted_at"`
}

// UploadRequest represents an asset upload request
type UploadRequest struct {
	ProjectID   string                 `form:"projectId" binding:"required,uuid"`
	FileName    string                 `form:"fileName" binding:"required"`
	Category    string                 `form:"category"`
	Tags        []string               `form:"tags"`
	Metadata    map[string]interface{} `form:"metadata"`
}

// UpdateAssetRequest represents an asset update request
type UpdateAssetRequest struct {
	FileName *string                 `json:"fileName,omitempty"`
	Category *string                 `json:"category,omitempty"`
	Tags     *[]string               `json:"tags,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

// AssetListResponse represents a paginated list of assets
type AssetListResponse struct {
	Assets     []Asset `json:"assets"`
	TotalCount int     `json:"totalCount"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	HasMore    bool    `json:"hasMore"`
}

// AssetFilter represents filters for querying assets
type AssetFilter struct {
	ProjectID string
	UserID    string
	Category  string
	Tags      []string
	Status    AssetStatus
	Search    string
	Page      int
	PageSize  int
}

// UploadURLResponse represents a presigned upload URL response
type UploadURLResponse struct {
	UploadURL string `json:"uploadUrl"`
	AssetID   string `json:"assetId"`
	ExpiresIn int    `json:"expiresIn"` // seconds
}

// DownloadURLResponse represents a presigned download URL response
type DownloadURLResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresIn   int    `json:"expiresIn"` // seconds
}

// Scan implements sql.Scanner for metadata JSONB
func (m *map[string]interface{}) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan metadata: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, m)
}

// Value implements driver.Valuer for metadata JSONB
func (m map[string]interface{}) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}
