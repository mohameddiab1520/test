package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/collab/services/build/internal/config"
	"github.com/yourorg/collab/services/build/internal/models"
)

type BuildService struct {
	config *config.Config
}

func NewBuildService(cfg *config.Config) *BuildService {
	return &BuildService{
		config: cfg,
	}
}

func (s *BuildService) CreateBuild(ctx context.Context, projectID, commitSHA, branch, triggeredBy string) (*models.Build, error) {
	build := &models.Build{
		ID:           uuid.New().String(),
		ProjectID:    projectID,
		CommitSHA:    commitSHA,
		Branch:       branch,
		Status:       models.StatusQueued,
		UnityVersion: s.config.DefaultUnityVersion,
		BuildTarget:  "StandaloneWindows64",
		TriggeredBy:  triggeredBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// TODO: Save to database
	// TODO: Enqueue build job

	return build, nil
}

func (s *BuildService) GetBuild(ctx context.Context, buildID string) (*models.Build, error) {
	// TODO: Get from database
	return &models.Build{
		ID:     buildID,
		Status: models.StatusSuccess,
	}, nil
}

func (s *BuildService) ListBuilds(ctx context.Context, projectID string, limit, offset int) ([]*models.Build, error) {
	// TODO: Get from database
	return []*models.Build{}, nil
}

func (s *BuildService) CancelBuild(ctx context.Context, buildID string) error {
	// TODO: Cancel running build
	return nil
}

func (s *BuildService) GetBuildLogs(ctx context.Context, buildID string) ([]string, error) {
	// TODO: Stream logs from storage
	return []string{"Build log line 1", "Build log line 2"}, nil
}

func (s *BuildService) GetBuildArtifacts(ctx context.Context, buildID string) ([]*models.BuildArtifact, error) {
	// TODO: Get artifacts from storage
	return []*models.BuildArtifact{}, nil
}

func (s *BuildService) ExecuteBuild(ctx context.Context, build *models.Build) error {
	// TODO: Implement Unity build execution
	fmt.Printf("Executing build %s\n", build.ID)
	return nil
}
