package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/collab/services/presence/internal/models"
	"github.com/yourorg/collab/services/presence/internal/service"
	"github.com/yourorg/collab/services/presence/internal/websocket"
	"go.uber.org/zap"
)

var (
	presenceUpdatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "presence_updates_total",
			Help: "Total number of presence updates",
		},
		[]string{"session_id"},
	)

	activityFeedEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "activity_feed_events_total",
			Help: "Total number of activity feed events",
		},
		[]string{"event_type"},
	)
)

func init() {
	prometheus.MustRegister(presenceUpdatesTotal)
	prometheus.MustRegister(activityFeedEventsTotal)
}

type PresenceHandler struct {
	service *service.PresenceService
	hub     *websocket.Hub
	logger  *zap.Logger
}

func NewPresenceHandler(service *service.PresenceService, hub *websocket.Hub, logger *zap.Logger) *PresenceHandler {
	return &PresenceHandler{
		service: service,
		hub:     hub,
		logger:  logger,
	}
}

func (h *PresenceHandler) GetSessionPresence(c *gin.Context) {
	sessionID := c.Param("sessionId")

	presences, err := h.service.GetSessionPresence(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Failed to get session presence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presences)
}

func (h *PresenceHandler) GetUserPresence(c *gin.Context) {
	userID := c.Param("userId")

	presence, err := h.service.GetUserPresence(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user presence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if presence == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "presence not found"})
		return
	}

	c.JSON(http.StatusOK, presence)
}

func (h *PresenceHandler) UpdatePresenceStatus(c *gin.Context) {
	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePresence(c.Request.Context(), &req); err != nil {
		h.logger.Error("Failed to update presence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	presenceUpdatesTotal.WithLabelValues(req.SessionID).Inc()

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *PresenceHandler) GetActivityFeed(c *gin.Context) {
	sessionID := c.Param("sessionId")
	limit := 50

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	activities, err := h.service.GetActivityFeed(c.Request.Context(), sessionID, limit)
	if err != nil {
		h.logger.Error("Failed to get activity feed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, activities)
}

func (h *PresenceHandler) ClearPresence(c *gin.Context) {
	sessionID := c.Param("sessionId")
	userID := c.Param("userId")

	if err := h.service.ClearPresence(c.Request.Context(), sessionID, userID); err != nil {
		h.logger.Error("Failed to clear presence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *PresenceHandler) Metrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
