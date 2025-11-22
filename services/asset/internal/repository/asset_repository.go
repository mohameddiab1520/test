package repository

import (
	"context"
	"errors"

	"github.com/yourorg/collab/services/asset/internal/models"
)

var (
	ErrAssetNotFound = errors.New("asset not found")
	ErrDuplicateAsset = errors.New("asset already exists")
)

// AssetRepository defines the interface for asset data access
type AssetRepository interface {
	Create(ctx context.Context, asset *models.Asset) error
	GetByID(ctx context.Context, assetID string) (*models.Asset, error)
	List(ctx context.Context, filter *models.AssetFilter) (*models.AssetListResponse, error)
	Update(ctx context.Context, asset *models.Asset) error
	Delete(ctx context.Context, assetID string) error
}
