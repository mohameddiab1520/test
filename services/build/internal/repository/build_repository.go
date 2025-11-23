package repository

import (
	"context"
	"errors"

	"github.com/yourorg/collab/services/build/internal/models"
)

var (
	ErrBuildNotFound      = errors.New("build not found")
	ErrArtifactNotFound   = errors.New("artifact not found")
	ErrDuplicateBuild     = errors.New("build already exists")
)

// BuildRepository defines the interface for build data access
type BuildRepository interface {
	// Build operations
	Create(ctx context.Context, build *models.Build) error
	GetByID(ctx context.Context, buildID string) (*models.Build, error)
	List(ctx context.Context, projectID string, limit, offset int) ([]*models.Build, error)
	Update(ctx context.Context, build *models.Build) error
	Delete(ctx context.Context, buildID string) error
	UpdateStatus(ctx context.Context, buildID string, status models.BuildStatus) error

	// Artifact operations
	CreateArtifact(ctx context.Context, artifact *models.BuildArtifact) error
	GetArtifacts(ctx context.Context, buildID string) ([]*models.BuildArtifact, error)
	GetArtifactByID(ctx context.Context, artifactID string) (*models.BuildArtifact, error)
}
