package models

import "time"

type BuildStatus string

const (
	StatusQueued    BuildStatus = "queued"
	StatusRunning   BuildStatus = "running"
	StatusSuccess   BuildStatus = "success"
	StatusFailed    BuildStatus = "failed"
	StatusCancelled BuildStatus = "cancelled"
)

type Build struct {
	ID           string      `json:"id"`
	ProjectID    string      `json:"project_id"`
	CommitSHA    string      `json:"commit_sha"`
	Branch       string      `json:"branch"`
	Status       BuildStatus `json:"status"`
	UnityVersion string      `json:"unity_version"`
	BuildTarget  string      `json:"build_target"`
	StartedAt    *time.Time  `json:"started_at"`
	CompletedAt  *time.Time  `json:"completed_at"`
	Duration     int64       `json:"duration_seconds"`
	TriggeredBy  string      `json:"triggered_by"`
	LogsURL      string      `json:"logs_url"`
	ArtifactsURL string      `json:"artifacts_url"`
	ErrorMessage string      `json:"error_message,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type BuildArtifact struct {
	ID          string    `json:"id"`
	BuildID     string    `json:"build_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	Checksum    string    `json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
}
