package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/conflict/internal/models"
	"github.com/yourorg/collab/services/conflict/internal/resolver"
)

// ConflictService handles conflict detection and resolution
type ConflictService struct {
	db       *sql.DB
	resolver *resolver.ConflictResolver
	logger   *zap.Logger
}

// NewConflictService creates a new conflict service
func NewConflictService(db *sql.DB, logger *zap.Logger) *ConflictService {
	return &ConflictService{
		db:       db,
		resolver: resolver.NewConflictResolver(),
		logger:   logger,
	}
}

// DetectConflict detects if there's a conflict for a resource
func (s *ConflictService) DetectConflict(
	ctx context.Context,
	projectID, resourceID string,
	resourceType models.ResourceType,
	userID string,
	content []byte,
	contentHash string,
	baseVersion int64,
) (*models.Conflict, bool, error) {
	// Get the current version from database
	var currentHash string
	var currentContent []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT content_hash, content FROM resource_versions
		 WHERE project_id = $1 AND resource_id = $2
		 ORDER BY created_at DESC LIMIT 1`,
		projectID, resourceID,
	).Scan(&currentHash, &currentContent)

	if err != nil && err != sql.ErrNoRows {
		return nil, false, fmt.Errorf("failed to get current version: %w", err)
	}

	// If no current version exists, no conflict
	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	// Detect conflict
	hasConflict, conflictType := s.resolver.DetectConflict(currentContent, currentHash, content, baseVersion)

	if !hasConflict {
		return nil, false, nil
	}

	// Create conflict record
	conflict := &models.Conflict{
		ID:           uuid.New().String(),
		ProjectID:    projectID,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		ConflictType: conflictType,
		Status:       models.ConflictStatusDetected,
		DetectedAt:   time.Now(),
	}

	// Save conflict to database
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO conflicts (id, project_id, resource_id, resource_type, conflict_type, status, detected_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		conflict.ID, conflict.ProjectID, conflict.ResourceID, conflict.ResourceType,
		conflict.ConflictType, conflict.Status, conflict.DetectedAt,
	)

	if err != nil {
		return nil, false, fmt.Errorf("failed to save conflict: %w", err)
	}

	// Save conflict versions
	versions := []*models.ConflictVersion{
		{
			ID:         uuid.New().String(),
			ConflictID: conflict.ID,
			VersionID:  fmt.Sprintf("%d", baseVersion),
			UserID:     userID,
			Content:    content,
			Hash:       contentHash,
			Timestamp:  time.Now(),
		},
		{
			ID:         uuid.New().String(),
			ConflictID: conflict.ID,
			VersionID:  "current",
			UserID:     "system",
			Content:    currentContent,
			Hash:       currentHash,
			Timestamp:  time.Now(),
		},
	}

	for _, v := range versions {
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO conflict_versions (id, conflict_id, version_id, user_id, content, hash, timestamp)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			v.ID, v.ConflictID, v.VersionID, v.UserID, v.Content, v.Hash, v.Timestamp,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to save conflict version: %w", err)
		}
	}

	return conflict, true, nil
}

// ResolveConflict resolves a conflict using the specified strategy
func (s *ConflictService) ResolveConflict(
	ctx context.Context,
	conflictID, userID string,
	strategy models.ResolutionStrategy,
	resolvedContent []byte,
) (*models.Conflict, error) {
	// Get conflict
	conflict, err := s.GetConflict(ctx, conflictID)
	if err != nil {
		return nil, err
	}

	if conflict.Status == models.ConflictStatusResolved {
		return nil, fmt.Errorf("conflict already resolved")
	}

	// Get conflict versions
	versions, err := s.getConflictVersions(ctx, conflictID)
	if err != nil {
		return nil, err
	}

	var finalContent []byte

	// If manual resolution, use provided content
	if strategy == models.ResolutionStrategyManual {
		finalContent = resolvedContent
	} else {
		// Auto-resolve using strategy
		finalContent, err = s.resolver.Resolve(conflict, versions, strategy)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve conflict: %w", err)
		}
	}

	// Update conflict status
	now := time.Now()
	strategyStr := string(strategy)
	_, err = s.db.ExecContext(ctx,
		`UPDATE conflicts
		 SET status = $1, resolved_by = $2, resolution_strategy = $3, resolved_at = $4
		 WHERE id = $5`,
		models.ConflictStatusResolved, userID, strategyStr, now, conflictID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update conflict: %w", err)
	}

	// Update conflict object
	conflict.Status = models.ConflictStatusResolved
	conflict.ResolvedBy = &userID
	conflict.ResolutionStrategy = &strategyStr
	conflict.ResolvedAt = &now

	return conflict, nil
}

// GetConflict retrieves a conflict by ID
func (s *ConflictService) GetConflict(ctx context.Context, conflictID string) (*models.Conflict, error) {
	conflict := &models.Conflict{}

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, resource_id, resource_type, conflict_type, status,
		        resolved_by, resolution_strategy, detected_at, resolved_at
		 FROM conflicts WHERE id = $1`,
		conflictID,
	).Scan(
		&conflict.ID, &conflict.ProjectID, &conflict.ResourceID, &conflict.ResourceType,
		&conflict.ConflictType, &conflict.Status, &conflict.ResolvedBy,
		&conflict.ResolutionStrategy, &conflict.DetectedAt, &conflict.ResolvedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conflict not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get conflict: %w", err)
	}

	return conflict, nil
}

// ListConflicts lists conflicts for a project
func (s *ConflictService) ListConflicts(
	ctx context.Context,
	projectID string,
	status models.ConflictStatus,
	limit, offset int,
) ([]*models.Conflict, int, error) {
	query := `SELECT id, project_id, resource_id, resource_type, conflict_type, status,
	                 resolved_by, resolution_strategy, detected_at, resolved_at
	          FROM conflicts WHERE project_id = $1`

	args := []interface{}{projectID}

	if status != models.ConflictStatusUnknown {
		query += ` AND status = $2`
		args = append(args, status)
	}

	query += ` ORDER BY detected_at DESC LIMIT $3 OFFSET $4`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conflicts: %w", err)
	}
	defer rows.Close()

	conflicts := make([]*models.Conflict, 0)
	for rows.Next() {
		conflict := &models.Conflict{}
		err := rows.Scan(
			&conflict.ID, &conflict.ProjectID, &conflict.ResourceID, &conflict.ResourceType,
			&conflict.ConflictType, &conflict.Status, &conflict.ResolvedBy,
			&conflict.ResolutionStrategy, &conflict.DetectedAt, &conflict.ResolvedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan conflict: %w", err)
		}
		conflicts = append(conflicts, conflict)
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM conflicts WHERE project_id = $1`
	countArgs := []interface{}{projectID}

	if status != models.ConflictStatusUnknown {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, status)
	}

	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count conflicts: %w", err)
	}

	return conflicts, total, nil
}

// AutoResolve attempts to automatically resolve a conflict
func (s *ConflictService) AutoResolve(
	ctx context.Context,
	conflictID string,
	strategy models.ResolutionStrategy,
) ([]byte, error) {
	// Get conflict
	conflict, err := s.GetConflict(ctx, conflictID)
	if err != nil {
		return nil, err
	}

	// Check if conflict can be auto-resolved
	if !s.resolver.CanAutoResolve(conflict.ConflictType) {
		return nil, fmt.Errorf("conflict type %s cannot be auto-resolved", conflict.ConflictType)
	}

	// Get versions
	versions, err := s.getConflictVersions(ctx, conflictID)
	if err != nil {
		return nil, err
	}

	// Resolve
	resolvedContent, err := s.resolver.Resolve(conflict, versions, strategy)
	if err != nil {
		return nil, err
	}

	return resolvedContent, nil
}

// getConflictVersions retrieves all versions for a conflict
func (s *ConflictService) getConflictVersions(ctx context.Context, conflictID string) ([]*models.ConflictVersion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conflict_id, version_id, user_id, user_name, content, hash, timestamp
		 FROM conflict_versions WHERE conflict_id = $1 ORDER BY timestamp`,
		conflictID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict versions: %w", err)
	}
	defer rows.Close()

	versions := make([]*models.ConflictVersion, 0)
	for rows.Next() {
		v := &models.ConflictVersion{}
		var userName sql.NullString
		err := rows.Scan(&v.ID, &v.ConflictID, &v.VersionID, &v.UserID, &userName, &v.Content, &v.Hash, &v.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		if userName.Valid {
			v.UserName = userName.String
		}
		versions = append(versions, v)
	}

	return versions, nil
}
