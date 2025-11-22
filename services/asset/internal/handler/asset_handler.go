package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/asset/internal/models"
	"github.com/yourorg/collab/services/asset/internal/repository"
	"github.com/yourorg/collab/services/asset/internal/service"
)

// AssetHandler handles HTTP requests for assets
type AssetHandler struct {
	service *service.AssetService
	logger  *zap.Logger
}

// NewAssetHandler creates a new asset handler
func NewAssetHandler(service *service.AssetService, logger *zap.Logger) *AssetHandler {
	return &AssetHandler{
		service: service,
		logger:  logger,
	}
}

// RequestUploadURL handles upload URL requests
// POST /api/v1/assets/upload-url
func (h *AssetHandler) RequestUploadURL(c *gin.Context) {
	var req models.UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resp, err := h.service.RequestUploadURL(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("Failed to request upload URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate upload URL"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ConfirmUpload confirms an asset upload
// POST /api/v1/assets/:id/confirm
func (h *AssetHandler) ConfirmUpload(c *gin.Context) {
	assetID := c.Param("id")

	var req struct {
		FileSize int64  `json:"fileSize" binding:"required"`
		MD5Hash  string `json:"md5Hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ConfirmUpload(c.Request.Context(), assetID, req.FileSize, req.MD5Hash); err != nil {
		h.logger.Error("Failed to confirm upload", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm upload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "upload confirmed"})
}

// GetAsset retrieves an asset by ID
// GET /api/v1/assets/:id
func (h *AssetHandler) GetAsset(c *gin.Context) {
	assetID := c.Param("id")

	asset, err := h.service.GetAsset(c.Request.Context(), assetID)
	if err != nil {
		if err == repository.ErrAssetNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		h.logger.Error("Failed to get asset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get asset"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

// ListAssets lists assets with filters
// GET /api/v1/assets
func (h *AssetHandler) ListAssets(c *gin.Context) {
	filter := &models.AssetFilter{
		ProjectID: c.Query("projectId"),
		Category:  c.Query("category"),
		Search:    c.Query("search"),
		Page:      1,
		PageSize:  20,
	}

	// Get user ID from context
	if userID, exists := c.Get("user_id"); exists {
		filter.UserID = userID.(string)
	}

	resp, err := h.service.ListAssets(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list assets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assets"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GenerateDownloadURL generates a download URL
// GET /api/v1/assets/:id/download
func (h *AssetHandler) GenerateDownloadURL(c *gin.Context) {
	assetID := c.Param("id")

	resp, err := h.service.GenerateDownloadURL(c.Request.Context(), assetID)
	if err != nil {
		if err == repository.ErrAssetNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		h.logger.Error("Failed to generate download URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateAsset updates asset metadata
// PUT /api/v1/assets/:id
func (h *AssetHandler) UpdateAsset(c *gin.Context) {
	assetID := c.Param("id")

	var req models.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset, err := h.service.UpdateAsset(c.Request.Context(), assetID, &req)
	if err != nil {
		if err == repository.ErrAssetNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		h.logger.Error("Failed to update asset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update asset"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

// DeleteAsset marks an asset as deleted
// DELETE /api/v1/assets/:id
func (h *AssetHandler) DeleteAsset(c *gin.Context) {
	assetID := c.Param("id")

	if err := h.service.DeleteAsset(c.Request.Context(), assetID); err != nil {
		if err == repository.ErrAssetNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		h.logger.Error("Failed to delete asset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete asset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
}
