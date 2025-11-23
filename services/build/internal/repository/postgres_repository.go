package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourorg/collab/services/build/internal/models"
)

// PostgresRepository implements BuildRepository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL-backed build repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new build into the database
func (r *PostgresRepository) Create(ctx context.Context, build *models.Build) error {
	query := `
		INSERT INTO builds (
			id, project_id, commit_sha, branch, status, unity_version,
			build_target, started_at, completed_at, duration_seconds,
			triggered_by, logs_url, artifacts_url, error_message,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.ExecContext(ctx, query,
		build.ID,
		build.ProjectID,
		build.CommitSHA,
		build.Branch,
		build.Status,
		build.UnityVersion,
		build.BuildTarget,
		build.StartedAt,
		build.CompletedAt,
		build.Duration,
		build.TriggeredBy,
		build.LogsURL,
		build.ArtifactsURL,
		build.ErrorMessage,
		build.CreatedAt,
		build.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create build: %w", err)
	}

	return nil
}

// GetByID retrieves a build by ID
func (r *PostgresRepository) GetByID(ctx context.Context, buildID string) (*models.Build, error) {
	query := `
		SELECT id, project_id, commit_sha, branch, status, unity_version,
		       build_target, started_at, completed_at, duration_seconds,
		       triggered_by, logs_url, artifacts_url, error_message,
		       created_at, updated_at
		FROM builds
		WHERE id = $1
	`

	build := &models.Build{}
	err := r.db.QueryRowContext(ctx, query, buildID).Scan(
		&build.ID,
		&build.ProjectID,
		&build.CommitSHA,
		&build.Branch,
		&build.Status,
		&build.UnityVersion,
		&build.BuildTarget,
		&build.StartedAt,
		&build.CompletedAt,
		&build.Duration,
		&build.TriggeredBy,
		&build.LogsURL,
		&build.ArtifactsURL,
		&build.ErrorMessage,
		&build.CreatedAt,
		&build.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrBuildNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	return build, nil
}

// List retrieves builds for a project with pagination
func (r *PostgresRepository) List(ctx context.Context, projectID string, limit, offset int) ([]*models.Build, error) {
	query := `
		SELECT id, project_id, commit_sha, branch, status, unity_version,
		       build_target, started_at, completed_at, duration_seconds,
		       triggered_by, logs_url, artifacts_url, error_message,
		       created_at, updated_at
		FROM builds
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}
	defer rows.Close()

	builds := []*models.Build{}
	for rows.Next() {
		build := &models.Build{}
		err := rows.Scan(
			&build.ID,
			&build.ProjectID,
			&build.CommitSHA,
			&build.Branch,
			&build.Status,
			&build.UnityVersion,
			&build.BuildTarget,
			&build.StartedAt,
			&build.CompletedAt,
			&build.Duration,
			&build.TriggeredBy,
			&build.LogsURL,
			&build.ArtifactsURL,
			&build.ErrorMessage,
			&build.CreatedAt,
			&build.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build: %w", err)
		}
		builds = append(builds, build)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating builds: %w", err)
	}

	return builds, nil
}

// Update updates an existing build
func (r *PostgresRepository) Update(ctx context.Context, build *models.Build) error {
	build.UpdatedAt = time.Now()

	query := `
		UPDATE builds
		SET status = $2, started_at = $3, completed_at = $4, duration_seconds = $5,
		    logs_url = $6, artifacts_url = $7, error_message = $8, updated_at = $9
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		build.ID,
		build.Status,
		build.StartedAt,
		build.CompletedAt,
		build.Duration,
		build.LogsURL,
		build.ArtifactsURL,
		build.ErrorMessage,
		build.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update build: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrBuildNotFound
	}

	return nil
}

// Delete removes a build from the database
func (r *PostgresRepository) Delete(ctx context.Context, buildID string) error {
	query := `DELETE FROM builds WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, buildID)
	if err != nil {
		return fmt.Errorf("failed to delete build: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrBuildNotFound
	}

	return nil
}

// UpdateStatus updates only the status of a build
func (r *PostgresRepository) UpdateStatus(ctx context.Context, buildID string, status models.BuildStatus) error {
	query := `
		UPDATE builds
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, buildID, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update build status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrBuildNotFound
	}

	return nil
}

// CreateArtifact inserts a new build artifact
func (r *PostgresRepository) CreateArtifact(ctx context.Context, artifact *models.BuildArtifact) error {
	query := `
		INSERT INTO build_artifacts (
			id, build_id, name, type, size, download_url, checksum, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		artifact.ID,
		artifact.BuildID,
		artifact.Name,
		artifact.Type,
		artifact.Size,
		artifact.DownloadURL,
		artifact.Checksum,
		artifact.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}

	return nil
}

// GetArtifacts retrieves all artifacts for a build
func (r *PostgresRepository) GetArtifacts(ctx context.Context, buildID string) ([]*models.BuildArtifact, error) {
	query := `
		SELECT id, build_id, name, type, size, download_url, checksum, created_at
		FROM build_artifacts
		WHERE build_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := []*models.BuildArtifact{}
	for rows.Next() {
		artifact := &models.BuildArtifact{}
		err := rows.Scan(
			&artifact.ID,
			&artifact.BuildID,
			&artifact.Name,
			&artifact.Type,
			&artifact.Size,
			&artifact.DownloadURL,
			&artifact.Checksum,
			&artifact.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return artifacts, nil
}

// GetArtifactByID retrieves a specific artifact by ID
func (r *PostgresRepository) GetArtifactByID(ctx context.Context, artifactID string) (*models.BuildArtifact, error) {
	query := `
		SELECT id, build_id, name, type, size, download_url, checksum, created_at
		FROM build_artifacts
		WHERE id = $1
	`

	artifact := &models.BuildArtifact{}
	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(
		&artifact.ID,
		&artifact.BuildID,
		&artifact.Name,
		&artifact.Type,
		&artifact.Size,
		&artifact.DownloadURL,
		&artifact.Checksum,
		&artifact.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrArtifactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return artifact, nil
}
