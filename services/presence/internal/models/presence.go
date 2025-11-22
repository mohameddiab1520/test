package models

import "time"

type PresenceStatus string

const (
	StatusOnline  PresenceStatus = "online"
	StatusAway    PresenceStatus = "away"
	StatusIdle    PresenceStatus = "idle"
	StatusOffline PresenceStatus = "offline"
)

type CursorPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CameraTransform struct {
	Position Vector3 `json:"position"`
	Rotation Vector3 `json:"rotation"`
}

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type PresenceState struct {
	PresenceID       string           `json:"presence_id" db:"presence_id"`
	SessionID        string           `json:"session_id" db:"session_id"`
	UserID           string           `json:"user_id" db:"user_id"`
	Status           PresenceStatus   `json:"status" db:"status"`
	CurrentScene     string           `json:"current_scene" db:"current_scene"`
	SelectedObject   string           `json:"selected_object" db:"selected_object"`
	CursorPosition   *CursorPosition  `json:"cursor_position" db:"cursor_position"`
	CameraTransform  *CameraTransform `json:"camera_transform" db:"camera_transform"`
	IsTyping         bool             `json:"is_typing" db:"is_typing"`
	LastActivityAt   time.Time        `json:"last_activity_at" db:"last_activity_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
	UserName         string           `json:"user_name,omitempty" db:"name"`
	UserAvatar       string           `json:"user_avatar,omitempty" db:"avatar_url"`
}

type ActivityType string

const (
	ActivityJoined   ActivityType = "joined"
	ActivityLeft     ActivityType = "left"
	ActivitySelected ActivityType = "selected"
	ActivityCreated  ActivityType = "created"
	ActivityDeleted  ActivityType = "deleted"
	ActivityModified ActivityType = "modified"
)

type ActivityFeedItem struct {
	ActivityID   string                 `json:"activity_id" db:"activity_id"`
	SessionID    string                 `json:"session_id" db:"session_id"`
	UserID       string                 `json:"user_id" db:"user_id"`
	UserName     string                 `json:"user_name" db:"name"`
	ActivityType ActivityType           `json:"activity_type" db:"activity_type"`
	ObjectID     string                 `json:"object_id,omitempty" db:"object_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

type PresenceUpdate struct {
	Type      string         `json:"type"`
	SessionID string         `json:"session_id"`
	Presence  *PresenceState `json:"presence"`
}

type PresenceUpdateMessage struct {
	Type     string         `json:"type"`
	UserID   string         `json:"user_id"`
	Presence *PresenceState `json:"presence"`
}

type UpdateStatusRequest struct {
	SessionID       string           `json:"session_id" binding:"required"`
	UserID          string           `json:"user_id" binding:"required"`
	Status          PresenceStatus   `json:"status"`
	CurrentScene    string           `json:"current_scene,omitempty"`
	SelectedObject  string           `json:"selected_object,omitempty"`
	CursorPosition  *CursorPosition  `json:"cursor_position,omitempty"`
	CameraTransform *CameraTransform `json:"camera_transform,omitempty"`
	IsTyping        bool             `json:"is_typing"`
}
