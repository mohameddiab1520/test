package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive SessionStatus = "active"
	SessionStatusPaused SessionStatus = "paused"
	SessionStatusEnded  SessionStatus = "ended"
)

// Session represents a collaboration session
type Session struct {
	ID               string         `json:"id" db:"session_id"`
	ProjectID        string         `json:"projectId" db:"project_id"`
	OwnerID          string         `json:"ownerId" db:"owner_id"`
	Name             string         `json:"name" db:"name"`
	Description      string         `json:"description" db:"description"`
	Settings         SessionSettings `json:"settings" db:"settings"`
	MaxParticipants  int            `json:"maxParticipants" db:"max_participants"`
	IsPublic         bool           `json:"isPublic" db:"is_public"`
	VoiceEnabled     bool           `json:"voiceEnabled" db:"voice_enabled"`
	RecordSession    bool           `json:"recordSession" db:"record_session"`
	Status           SessionStatus  `json:"status" db:"status"`
	SceneData        *SceneData     `json:"sceneData,omitempty" db:"scene_data"`
	RecordingURL     string         `json:"recordingUrl,omitempty" db:"recording_url"`
	CreatedAt        time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time      `json:"updatedAt" db:"updated_at"`
	StartedAt        *time.Time     `json:"startedAt,omitempty" db:"started_at"`
	EndedAt          *time.Time     `json:"endedAt,omitempty" db:"ended_at"`
}

// SessionSettings represents session configuration
type SessionSettings struct {
	AutoSave         bool              `json:"autoSave"`
	SaveInterval     int               `json:"saveInterval"`     // seconds
	AllowGuests      bool              `json:"allowGuests"`
	RequireApproval  bool              `json:"requireApproval"`
	MaxIdleTime      int               `json:"maxIdleTime"`      // minutes
	CustomProperties map[string]string `json:"customProperties"`
}

// SceneData represents the Unity scene state
type SceneData struct {
	SceneName    string                 `json:"sceneName"`
	Objects      []GameObject           `json:"objects"`
	Version      int                    `json:"version"`
	LastModified time.Time              `json:"lastModified"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// GameObject represents a Unity GameObject
type GameObject struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Transform Transform              `json:"transform"`
	Components []Component           `json:"components"`
	Children  []string               `json:"children"`
	Active    bool                   `json:"active"`
	Tags      []string               `json:"tags"`
	Layer     int                    `json:"layer"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Transform represents position, rotation, scale
type Transform struct {
	Position Vector3 `json:"position"`
	Rotation Vector3 `json:"rotation"`
	Scale    Vector3 `json:"scale"`
}

// Vector3 represents a 3D vector
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Component represents a Unity component
type Component struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Enabled    bool                   `json:"enabled"`
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	ProjectID       string          `json:"projectId" binding:"required,uuid"`
	Name            string          `json:"name" binding:"required,min=1,max=255"`
	Description     string          `json:"description"`
	Settings        SessionSettings `json:"settings"`
	MaxParticipants int             `json:"maxParticipants" binding:"min=1,max=100"`
	IsPublic        bool            `json:"isPublic"`
	VoiceEnabled    bool            `json:"voiceEnabled"`
	RecordSession   bool            `json:"recordSession"`
}

// UpdateSessionRequest represents a request to update a session
type UpdateSessionRequest struct {
	Name            *string          `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description     *string          `json:"description,omitempty"`
	Settings        *SessionSettings `json:"settings,omitempty"`
	MaxParticipants *int             `json:"maxParticipants,omitempty" binding:"omitempty,min=1,max=100"`
	IsPublic        *bool            `json:"isPublic,omitempty"`
	VoiceEnabled    *bool            `json:"voiceEnabled,omitempty"`
	Status          *SessionStatus   `json:"status,omitempty"`
}

// UpdateSceneDataRequest represents a request to update scene data
type UpdateSceneDataRequest struct {
	SceneData SceneData `json:"sceneData" binding:"required"`
}

// SessionListResponse represents a paginated list of sessions
type SessionListResponse struct {
	Sessions   []Session `json:"sessions"`
	TotalCount int       `json:"totalCount"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	HasMore    bool      `json:"hasMore"`
}

// SessionFilter represents filters for querying sessions
type SessionFilter struct {
	ProjectID string
	OwnerID   string
	Status    SessionStatus
	IsPublic  *bool
	Page      int
	PageSize  int
}

// Value implements driver.Valuer for SessionSettings
func (s SessionSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for SessionSettings
func (s *SessionSettings) Scan(value interface{}) error {
	if value == nil {
		*s = SessionSettings{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SessionSettings: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for SceneData
func (sd *SceneData) Value() (driver.Value, error) {
	if sd == nil {
		return nil, nil
	}
	return json.Marshal(sd)
}

// Scan implements sql.Scanner for SceneData
func (sd *SceneData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SceneData: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, sd)
}
