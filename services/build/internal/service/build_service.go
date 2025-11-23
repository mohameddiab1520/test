package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/collab/services/build/internal/builder"
	"github.com/yourorg/collab/services/build/internal/config"
	"github.com/yourorg/collab/services/build/internal/models"
	"github.com/yourorg/collab/services/build/internal/repository"
	"github.com/yourorg/collab/services/build/internal/storage"
)

type BuildService struct {
	config     *config.Config
	repo       repository.BuildRepository
	queue      *builder.BuildQueue
	storage    *storage.S3Storage
	unityPath  string
	workspace  string
}

func NewBuildService(
	cfg *config.Config,
	repo repository.BuildRepository,
	queue *builder.BuildQueue,
	storage *storage.S3Storage,
) *BuildService {
	return &BuildService{
		config:    cfg,
		repo:      repo,
		queue:     queue,
		storage:   storage,
		unityPath: cfg.UnityPath,
		workspace: cfg.WorkspaceDir,
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

	// Save to database
	if err := s.repo.Create(ctx, build); err != nil {
		return nil, fmt.Errorf("failed to save build: %w", err)
	}

	// Enqueue build job
	if err := s.queue.Enqueue(ctx, build); err != nil {
		return nil, fmt.Errorf("failed to enqueue build: %w", err)
	}

	return build, nil
}

func (s *BuildService) GetBuild(ctx context.Context, buildID string) (*models.Build, error) {
	return s.repo.GetByID(ctx, buildID)
}

func (s *BuildService) ListBuilds(ctx context.Context, projectID string, limit, offset int) ([]*models.Build, error) {
	return s.repo.List(ctx, projectID, limit, offset)
}

func (s *BuildService) CancelBuild(ctx context.Context, buildID string) error {
	// Update build status to cancelled
	if err := s.repo.UpdateStatus(ctx, buildID, models.StatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel build: %w", err)
	}

	return nil
}

func (s *BuildService) GetBuildLogs(ctx context.Context, buildID string) ([]string, error) {
	// Get build to find logs URL
	build, err := s.repo.GetByID(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if build.LogsURL == "" {
		return []string{}, nil
	}

	// Download logs from S3
	logs, err := s.storage.DownloadLogs(ctx, build.LogsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download logs: %w", err)
	}

	return logs, nil
}

func (s *BuildService) GetBuildArtifacts(ctx context.Context, buildID string) ([]*models.BuildArtifact, error) {
	return s.repo.GetArtifacts(ctx, buildID)
}

func (s *BuildService) ExecuteBuild(ctx context.Context, build *models.Build) error {
	// Update status to running
	build.Status = models.StatusRunning
	now := time.Now()
	build.StartedAt = &now
	if err := s.repo.Update(ctx, build); err != nil {
		return fmt.Errorf("failed to update build status: %w", err)
	}

	// Create log buffer
	var logBuffer bytes.Buffer

	// Create Unity builder
	unityBuilder := builder.NewUnityBuilder(s.unityPath, s.workspace, &logBuffer)
	defer unityBuilder.Cleanup(build.ID)

	// Execute build
	opts := &builder.BuildOptions{
		ProjectPath:  ".",
		BuildTarget:  build.BuildTarget,
		OutputPath:   "Build",
		UnityVersion: build.UnityVersion,
	}

	err := unityBuilder.Execute(ctx, build, opts)

	// Upload logs to S3
	logsURL, uploadErr := s.storage.UploadLogs(ctx, build.ID, logBuffer.Bytes())
	if uploadErr == nil {
		build.LogsURL = logsURL
	}

	// Update build status
	completedAt := time.Now()
	build.CompletedAt = &completedAt
	build.Duration = int64(completedAt.Sub(*build.StartedAt).Seconds())

	if err != nil {
		build.Status = models.StatusFailed
		build.ErrorMessage = err.Error()
	} else {
		build.Status = models.StatusSuccess

		// Upload artifacts
		if err := s.uploadArtifacts(ctx, build, unityBuilder, opts.OutputPath); err != nil {
			build.ErrorMessage = fmt.Sprintf("Build succeeded but artifact upload failed: %v", err)
		}
	}

	// Update build in database
	if updateErr := s.repo.Update(ctx, build); updateErr != nil {
		return fmt.Errorf("failed to update build: %w", updateErr)
	}

	// Mark as complete in queue
	if queueErr := s.queue.Complete(ctx, build.ID); queueErr != nil {
		return fmt.Errorf("failed to complete build in queue: %w", queueErr)
	}

	return err
}

func (s *BuildService) uploadArtifacts(ctx context.Context, build *models.Build, unityBuilder *builder.UnityBuilder, outputPath string) error {
	// Get build artifacts from builder
	artifactPaths, err := unityBuilder.GetBuildArtifacts(build.ID, outputPath)
	if err != nil {
		return err
	}

	// Upload each artifact to S3
	for _, path := range artifactPaths {
		artifact := &models.BuildArtifact{
			ID:        uuid.New().String(),
			BuildID:   build.ID,
			Name:      path,
			Type:      "binary",
			CreatedAt: time.Now(),
		}

		// Upload to S3
		downloadURL, size, checksum, err := s.storage.UploadArtifact(ctx, build.ID, path)
		if err != nil {
			return fmt.Errorf("failed to upload artifact %s: %w", path, err)
		}

		artifact.DownloadURL = downloadURL
		artifact.Size = size
		artifact.Checksum = checksum

		// Save artifact metadata to database
		if err := s.repo.CreateArtifact(ctx, artifact); err != nil {
			return fmt.Errorf("failed to save artifact metadata: %w", err)
		}
	}

	return nil
}
