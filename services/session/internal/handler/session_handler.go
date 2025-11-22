package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/session/internal/models"
	"github.com/yourorg/collab/services/session/internal/repository"
	"github.com/yourorg/collab/services/session/internal/service"
)

// SessionHandler handles HTTP requests for sessions
type SessionHandler struct {
	sessionService *service.SessionService
	logger         *zap.Logger
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionService *service.SessionService, logger *zap.Logger) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		logger:         logger,
	}
}

// CreateSession creates a new session
// POST /api/v1/sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req models.CreateSessionRequest
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

	session, err := h.sessionService.CreateSession(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("Failed to create session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession retrieves a session by ID
// GET /api/v1/sessions/:id
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	session, err := h.sessionService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if err == repository.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		h.logger.Error("Failed to get session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSession updates a session
// PATCH /api/v1/sessions/:id
func (h *SessionHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("id")

	var req models.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	session, err := h.sessionService.UpdateSession(c.Request.Context(), sessionID, userID.(string), &req)
	if err != nil {
		if err == repository.ErrNotSessionOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "only session owner can update"})
			return
		}
		if err == repository.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if err == repository.ErrSessionEnded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session has ended"})
			return
		}
		h.logger.Error("Failed to update session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession deletes a session
// DELETE /api/v1/sessions/:id
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.sessionService.DeleteSession(c.Request.Context(), sessionID, userID.(string))
	if err != nil {
		if err == repository.ErrNotSessionOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "only session owner can delete"})
			return
		}
		if err == repository.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		h.logger.Error("Failed to delete session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted successfully"})
}

// ListSessions lists sessions with filters
// GET /api/v1/sessions
func (h *SessionHandler) ListSessions(c *gin.Context) {
	filter := &models.SessionFilter{
		ProjectID: c.Query("projectId"),
		OwnerID:   c.Query("ownerId"),
		Status:    models.SessionStatus(c.Query("status")),
		Page:      1,
		PageSize:  20,
	}

	// Parse pagination
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if pageSize := c.Query("pageSize"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			filter.PageSize = ps
		}
	}

	// Parse isPublic
	if isPublic := c.Query("isPublic"); isPublic != "" {
		if ip, err := strconv.ParseBool(isPublic); err == nil {
			filter.IsPublic = &ip
		}
	}

	response, err := h.sessionService.ListSessions(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSceneData updates session scene data
// PUT /api/v1/sessions/:id/scene
func (h *SessionHandler) UpdateSceneData(c *gin.Context) {
	sessionID := c.Param("id")

	var req models.UpdateSceneDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.sessionService.UpdateSceneData(c.Request.Context(), sessionID, userID.(string), &req.SceneData)
	if err != nil {
		if err == repository.ErrParticipantNotInSession {
			c.JSON(http.StatusForbidden, gin.H{"error": "must be a participant to update scene"})
			return
		}
		h.logger.Error("Failed to update scene data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scene data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "scene data updated successfully"})
}

// JoinSession joins a session
// POST /api/v1/sessions/:id/join
func (h *SessionHandler) JoinSession(c *gin.Context) {
	sessionID := c.Param("id")

	var req models.JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Set default role if not provided
		req.Role = models.ParticipantRoleViewer
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Default to viewer role if not specified
	if req.Role == "" {
		req.Role = models.ParticipantRoleViewer
	}

	err := h.sessionService.JoinSession(c.Request.Context(), sessionID, userID.(string), req.Role)
	if err != nil {
		if err == repository.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if err == repository.ErrSessionEnded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session has ended"})
			return
		}
		if err == repository.ErrSessionFull {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session is full"})
			return
		}
		if err == repository.ErrParticipantAlreadyJoined {
			c.JSON(http.StatusBadRequest, gin.H{"error": "already joined this session"})
			return
		}
		h.logger.Error("Failed to join session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "joined session successfully"})
}

// LeaveSession leaves a session
// POST /api/v1/sessions/:id/leave
func (h *SessionHandler) LeaveSession(c *gin.Context) {
	sessionID := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.sessionService.LeaveSession(c.Request.Context(), sessionID, userID.(string))
	if err != nil {
		if err == repository.ErrParticipantNotInSession {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not in this session"})
			return
		}
		h.logger.Error("Failed to leave session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to leave session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left session successfully"})
}

// GetParticipants retrieves session participants
// GET /api/v1/sessions/:id/participants
func (h *SessionHandler) GetParticipants(c *gin.Context) {
	sessionID := c.Param("id")

	response, err := h.sessionService.GetParticipants(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Failed to get participants", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get participants"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePresence updates participant presence
// PATCH /api/v1/sessions/:id/presence
func (h *SessionHandler) UpdatePresence(c *gin.Context) {
	sessionID := c.Param("id")

	var req models.UpdatePresenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.sessionService.UpdatePresence(c.Request.Context(), sessionID, userID.(string), &req)
	if err != nil {
		if err == repository.ErrParticipantNotInSession {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not in this session"})
			return
		}
		h.logger.Error("Failed to update presence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update presence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "presence updated successfully"})
}
