package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ParticipantRole represents the role of a participant
type ParticipantRole string

const (
	ParticipantRoleOwner  ParticipantRole = "owner"
	ParticipantRoleEditor ParticipantRole = "editor"
	ParticipantRoleViewer ParticipantRole = "viewer"
)

// ParticipantStatus represents the status of a participant
type ParticipantStatus string

const (
	ParticipantStatusOnline  ParticipantStatus = "online"
	ParticipantStatusAway    ParticipantStatus = "away"
	ParticipantStatusOffline ParticipantStatus = "offline"
)

// Participant represents a session participant
type Participant struct {
	SessionID        string             `json:"sessionId" db:"session_id"`
	UserID           string             `json:"userId" db:"user_id"`
	Role             ParticipantRole    `json:"role" db:"role"`
	Status           ParticipantStatus  `json:"status" db:"status"`
	CurrentScene     string             `json:"currentScene,omitempty" db:"current_scene"`
	SelectedObject   string             `json:"selectedObject,omitempty" db:"selected_object"`
	CursorPosition   *Position          `json:"cursorPosition,omitempty" db:"cursor_position"`
	CameraTransform  *CameraTransform   `json:"cameraTransform,omitempty" db:"camera_transform"`
	VoiceMuted       bool               `json:"voiceMuted" db:"voice_muted"`
	VoiceDeafened    bool               `json:"voiceDeafened" db:"voice_deafened"`
	JoinedAt         time.Time          `json:"joinedAt" db:"joined_at"`
	LeftAt           *time.Time         `json:"leftAt,omitempty" db:"left_at"`
	LastActivityAt   time.Time          `json:"lastActivityAt" db:"last_activity_at"`

	// Populated from user table via JOIN
	Username         string             `json:"username,omitempty" db:"username"`
	AvatarURL        string             `json:"avatarUrl,omitempty" db:"avatar_url"`
}

// Position represents cursor position in 3D space
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// CameraTransform represents camera position and rotation
type CameraTransform struct {
	Position Position `json:"position"`
	Rotation Rotation `json:"rotation"`
	FieldOfView float64 `json:"fieldOfView"`
}

// Rotation represents rotation in Euler angles
type Rotation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// JoinSessionRequest represents a request to join a session
type JoinSessionRequest struct {
	Role ParticipantRole `json:"role,omitempty"`
}

// UpdatePresenceRequest represents a request to update presence
type UpdatePresenceRequest struct {
	Status          *ParticipantStatus `json:"status,omitempty"`
	CurrentScene    *string            `json:"currentScene,omitempty"`
	SelectedObject  *string            `json:"selectedObject,omitempty"`
	CursorPosition  *Position          `json:"cursorPosition,omitempty"`
	CameraTransform *CameraTransform   `json:"cameraTransform,omitempty"`
	VoiceMuted      *bool              `json:"voiceMuted,omitempty"`
	VoiceDeafened   *bool              `json:"voiceDeafened,omitempty"`
}

// ParticipantListResponse represents a list of participants
type ParticipantListResponse struct {
	Participants []Participant `json:"participants"`
	TotalCount   int           `json:"totalCount"`
}

// Value implements driver.Valuer for Position
func (p *Position) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner for Position
func (p *Position) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Position: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, p)
}

// Value implements driver.Valuer for CameraTransform
func (ct *CameraTransform) Value() (driver.Value, error) {
	if ct == nil {
		return nil, nil
	}
	return json.Marshal(ct)
}

// Scan implements sql.Scanner for CameraTransform
func (ct *CameraTransform) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan CameraTransform: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, ct)
}
