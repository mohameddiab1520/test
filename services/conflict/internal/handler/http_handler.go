package handler

import (
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/conflict/internal/models"
	"github.com/yourorg/collab/services/conflict/internal/service"
)

// ConflictHandler handles HTTP requests for conflict operations
type ConflictHandler struct {
	service *service.ConflictService
	logger  *zap.Logger
}

// NewConflictHandler creates a new conflict handler
func NewConflictHandler(service *service.ConflictService, logger *zap.Logger) *ConflictHandler {
	return &ConflictHandler{
		service: service,
		logger:  logger,
	}
}

// DetectConflictRequest represents a conflict detection request
type DetectConflictRequest struct {
	ProjectID    string `json:"project_id" binding:"required"`
	ResourceID   string `json:"resource_id" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required"`
	UserID       string `json:"user_id" binding:"required"`
	Content      string `json:"content" binding:"required"` // base64 encoded
	ContentHash  string `json:"content_hash" binding:"required"`
	BaseVersion  int64  `json:"base_version"`
}

// ResolveConflictRequest represents a conflict resolution request
type ResolveConflictRequest struct {
	UserID          string `json:"user_id" binding:"required"`
	Strategy        string `json:"strategy" binding:"required"`
	ResolvedContent string `json:"resolved_content"` // base64 encoded
	ResolutionNote  string `json:"resolution_note"`
}

// DetectConflict handles conflict detection
func (h *ConflictHandler) DetectConflict(c *gin.Context) {
	var req DetectConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode content
	content, err := base64.StdEncoding.DecodeString(req.Content)
	if err != nil {
		h.logger.Error("Failed to decode content", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content encoding"})
		return
	}

	// Detect conflict
	conflict, hasConflict, err := h.service.DetectConflict(
		c.Request.Context(),
		req.ProjectID,
		req.ResourceID,
		models.ResourceType(req.ResourceType),
		req.UserID,
		content,
		req.ContentHash,
		req.BaseVersion,
	)

	if err != nil {
		h.logger.Error("Failed to detect conflict", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detect conflict"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_conflict": hasConflict,
		"conflict":     conflict,
	})
}

// ResolveConflict handles conflict resolution
func (h *ConflictHandler) ResolveConflict(c *gin.Context) {
	conflictID := c.Param("id")

	var req ResolveConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode resolved content if provided
	var resolvedContent []byte
	var err error
	if req.ResolvedContent != "" {
		resolvedContent, err = base64.StdEncoding.DecodeString(req.ResolvedContent)
		if err != nil {
			h.logger.Error("Failed to decode resolved content", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content encoding"})
			return
		}
	}

	// Resolve conflict
	conflict, err := h.service.ResolveConflict(
		c.Request.Context(),
		conflictID,
		req.UserID,
		models.ResolutionStrategy(req.Strategy),
		resolvedContent,
	)

	if err != nil {
		h.logger.Error("Failed to resolve conflict", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"conflict": conflict,
	})
}

// GetConflict retrieves a conflict by ID
func (h *ConflictHandler) GetConflict(c *gin.Context) {
	conflictID := c.Param("id")

	conflict, err := h.service.GetConflict(c.Request.Context(), conflictID)
	if err != nil {
		h.logger.Error("Failed to get conflict", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "conflict not found"})
		return
	}

	c.JSON(http.StatusOK, conflict)
}

// ListConflicts lists conflicts for a project
func (h *ConflictHandler) ListConflicts(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	status := models.ConflictStatus(c.Query("status"))
	if status == "" {
		status = models.ConflictStatusUnknown
	}

	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	conflicts, total, err := h.service.ListConflicts(
		c.Request.Context(),
		projectID,
		status,
		limit,
		offset,
	)

	if err != nil {
		h.logger.Error("Failed to list conflicts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list conflicts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conflicts": conflicts,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// AutoResolve attempts to automatically resolve a conflict
func (h *ConflictHandler) AutoResolve(c *gin.Context) {
	conflictID := c.Param("id")

	strategy := c.Query("strategy")
	if strategy == "" {
		strategy = string(models.ResolutionStrategyAutoMerge)
	}

	resolvedContent, err := h.service.AutoResolve(
		c.Request.Context(),
		conflictID,
		models.ResolutionStrategy(strategy),
	)

	if err != nil {
		h.logger.Error("Failed to auto-resolve conflict", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Encode content
	encodedContent := base64.StdEncoding.EncodeToString(resolvedContent)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"resolved_content": encodedContent,
	})
}
