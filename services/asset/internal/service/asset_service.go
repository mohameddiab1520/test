package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/asset/internal/models"
	"github.com/yourorg/collab/services/asset/internal/repository"
	"github.com/yourorg/collab/services/asset/internal/storage"
)

// AssetService handles asset business logic
type AssetService struct {
	repo    repository.AssetRepository
	storage *storage.S3Storage
	logger  *zap.Logger
}

// NewAssetService creates a new asset service
func NewAssetService(repo repository.AssetRepository, storage *storage.S3Storage, logger *zap.Logger) *AssetService {
	return &AssetService{
		repo:    repo,
		storage: storage,
		logger:  logger,
	}
}

// RequestUploadURL generates a presigned upload URL
func (s *AssetService) RequestUploadURL(ctx context.Context, userID string, req *models.UploadRequest) (*models.UploadURLResponse, error) {
	// Create asset record
	assetID := uuid.New().String()
	storagePath := fmt.Sprintf("%s/%s/%s", req.ProjectID, assetID, req.FileName)

	asset := &models.Asset{
		ID:            assetID,
		ProjectID:     req.ProjectID,
		UserID:        userID,
		FileName:      req.FileName,
		StoragePath:   storagePath,
		Category:      req.Category,
		Tags:          req.Tags,
		Metadata:      req.Metadata,
		Status:        models.AssetStatusUploading,
		VersionNumber: 1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	// Generate presigned upload URL (valid for 15 minutes)
	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, storagePath, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &models.UploadURLResponse{
		UploadURL: uploadURL,
		AssetID:   assetID,
		ExpiresIn: 900, // 15 minutes
	}, nil
}

// ConfirmUpload confirms an asset upload and updates its status
func (s *AssetService) ConfirmUpload(ctx context.Context, assetID string, fileSize int64, md5Hash string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Update asset with file info
	asset.FileSize = fileSize
	asset.MD5Hash = md5Hash
	asset.Status = models.AssetStatusReady
	asset.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, asset); err != nil {
		return fmt.Errorf("failed to update asset: %w", err)
	}

	s.logger.Info("Asset upload confirmed",
		zap.String("assetId", assetID),
		zap.Int64("fileSize", fileSize),
	)

	return nil
}

// GetAsset retrieves an asset by ID
func (s *AssetService) GetAsset(ctx context.Context, assetID string) (*models.Asset, error) {
	return s.repo.GetByID(ctx, assetID)
}

// ListAssets lists assets with filters
func (s *AssetService) ListAssets(ctx context.Context, filter *models.AssetFilter) (*models.AssetListResponse, error) {
	return s.repo.List(ctx, filter)
}

// GenerateDownloadURL generates a presigned download URL
func (s *AssetService) GenerateDownloadURL(ctx context.Context, assetID string) (*models.DownloadURLResponse, error) {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	if asset.Status != models.AssetStatusReady {
		return nil, fmt.Errorf("asset is not ready for download")
	}

	// Generate presigned download URL (valid for 15 minutes)
	downloadURL, err := s.storage.GeneratePresignedDownloadURL(ctx, asset.StoragePath, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &models.DownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   900, // 15 minutes
	}, nil
}

// DeleteAsset marks an asset as deleted
func (s *AssetService) DeleteAsset(ctx context.Context, assetID string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	now := time.Now()
	asset.Status = models.AssetStatusDeleted
	asset.DeletedAt = &now
	asset.UpdatedAt = now

	if err := s.repo.Update(ctx, asset); err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	// Optionally delete from storage
	// if err := s.storage.DeleteFile(ctx, asset.StoragePath); err != nil {
	//     s.logger.Error("Failed to delete from storage", zap.Error(err))
	// }

	return nil
}

// UpdateAsset updates asset metadata
func (s *AssetService) UpdateAsset(ctx context.Context, assetID string, req *models.UpdateAssetRequest) (*models.Asset, error) {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	// Update fields
	if req.FileName != nil {
		asset.FileName = *req.FileName
	}
	if req.Category != nil {
		asset.Category = *req.Category
	}
	if req.Tags != nil {
		asset.Tags = *req.Tags
	}
	if req.Metadata != nil {
		asset.Metadata = *req.Metadata
	}

	asset.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	return asset, nil
}
