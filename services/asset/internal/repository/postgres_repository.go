package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yourorg/collab/services/asset/internal/models"
)

// PostgresRepository implements AssetRepository for PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) AssetRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new asset
func (r *PostgresRepository) Create(ctx context.Context, asset *models.Asset) error {
	query := `
		INSERT INTO assets (
			asset_id, project_id, user_id, file_name, file_size, content_type,
			md5_hash, storage_path, cdn_url, thumbnail_url, metadata, category,
			tags, status, error_message, version_number, parent_asset_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		asset.ID, asset.ProjectID, asset.UserID, asset.FileName, asset.FileSize,
		asset.ContentType, asset.MD5Hash, asset.StoragePath, asset.CDNURL,
		asset.ThumbnailURL, asset.Metadata, asset.Category, asset.Tags,
		asset.Status, asset.ErrorMessage, asset.VersionNumber, asset.ParentAssetID,
		asset.CreatedAt, asset.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return nil
}

// GetByID retrieves an asset by ID
func (r *PostgresRepository) GetByID(ctx context.Context, assetID string) (*models.Asset, error) {
	var asset models.Asset
	query := `
		SELECT
			asset_id, project_id, user_id, file_name, file_size, content_type,
			md5_hash, storage_path, cdn_url, thumbnail_url, metadata, category,
			tags, status, error_message, version_number, parent_asset_id,
			created_at, updated_at, deleted_at
		FROM assets
		WHERE asset_id = $1 AND deleted_at IS NULL
	`

	err := r.db.GetContext(ctx, &asset, query, assetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAssetNotFound
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return &asset, nil
}

// List retrieves assets with filters
func (r *PostgresRepository) List(ctx context.Context, filter *models.AssetFilter) (*models.AssetListResponse, error) {
	query := `
		SELECT
			asset_id, project_id, user_id, file_name, file_size, content_type,
			md5_hash, storage_path, cdn_url, thumbnail_url, metadata, category,
			tags, status, error_message, version_number, parent_asset_id,
			created_at, updated_at
		FROM assets
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.ProjectID != "" {
		query += fmt.Sprintf(" AND project_id = $%d", argCount)
		args = append(args, filter.ProjectID)
		argCount++
	}

	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, filter.UserID)
		argCount++
	}

	if filter.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, filter.Category)
		argCount++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND file_name ILIKE $%d", argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS count_query"
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count assets: %w", err)
	}

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, pageSize, offset)

	// Query assets
	assets := []models.Asset{}
	err = r.db.SelectContext(ctx, &assets, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	return &models.AssetListResponse{
		Assets:     assets,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    (page * pageSize) < totalCount,
	}, nil
}

// Update updates an asset
func (r *PostgresRepository) Update(ctx context.Context, asset *models.Asset) error {
	query := `
		UPDATE assets SET
			file_name = $2,
			file_size = $3,
			content_type = $4,
			md5_hash = $5,
			cdn_url = $6,
			thumbnail_url = $7,
			metadata = $8,
			category = $9,
			tags = $10,
			status = $11,
			error_message = $12,
			updated_at = $13,
			deleted_at = $14
		WHERE asset_id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		asset.ID, asset.FileName, asset.FileSize, asset.ContentType,
		asset.MD5Hash, asset.CDNURL, asset.ThumbnailURL, asset.Metadata,
		asset.Category, asset.Tags, asset.Status, asset.ErrorMessage,
		asset.UpdatedAt, asset.DeletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update asset: %w", err)
	}

	return nil
}

// Delete deletes an asset (soft delete)
func (r *PostgresRepository) Delete(ctx context.Context, assetID string) error {
	query := `
		UPDATE assets SET
			deleted_at = NOW(),
			status = $2
		WHERE asset_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, assetID, models.AssetStatusDeleted)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	return nil
}
