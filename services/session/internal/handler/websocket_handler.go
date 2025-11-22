package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/session/internal/models"
	"github.com/yourorg/collab/services/session/internal/repository"
	"github.com/yourorg/collab/services/session/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check against allowed origins from environment
		// For development, allow all origins
		origin := r.Header.Get("Origin")

		// Allow same origin
		if origin == "" {
			return true
		}

		// TODO: Load allowed origins from config/environment
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://localhost:8081",
			"http://localhost:8082",
			"http://localhost:8083",
		}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		// For development: allow all (remove in production)
		return true
	},
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"sessionId,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID        string
	SessionID string
	UserID    string
	Conn      *websocket.Conn
	Send      chan *WebSocketMessage
	Hub       *WebSocketHub
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[string]*WebSocketClient // clientID -> client
	sessions   map[string]map[string]*WebSocketClient // sessionID -> clientID -> client
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan *WebSocketMessage
	mu         sync.RWMutex
	logger     *zap.Logger
	service    *service.SessionService
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *zap.Logger, service *service.SessionService) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]*WebSocketClient),
		sessions:   make(map[string]map[string]*WebSocketClient),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan *WebSocketMessage, 256),
		logger:     logger,
		service:    service,
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client

			if _, ok := h.sessions[client.SessionID]; !ok {
				h.sessions[client.SessionID] = make(map[string]*WebSocketClient)
			}
			h.sessions[client.SessionID][client.ID] = client
			h.mu.Unlock()

			h.logger.Info("Client connected",
				zap.String("clientId", client.ID),
				zap.String("sessionId", client.SessionID),
				zap.String("userId", client.UserID),
			)

			// Broadcast join event
			h.broadcastToSession(client.SessionID, &WebSocketMessage{
				Type:      "participant_joined",
				SessionID: client.SessionID,
				UserID:    client.UserID,
				Timestamp: time.Now().Unix(),
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)

				if session, ok := h.sessions[client.SessionID]; ok {
					delete(session, client.ID)
					if len(session) == 0 {
						delete(h.sessions, client.SessionID)
					}
				}

				close(client.Send)
			}
			h.mu.Unlock()

			h.logger.Info("Client disconnected",
				zap.String("clientId", client.ID),
				zap.String("sessionId", client.SessionID),
			)

			// Broadcast leave event
			h.broadcastToSession(client.SessionID, &WebSocketMessage{
				Type:      "participant_left",
				SessionID: client.SessionID,
				UserID:    client.UserID,
				Timestamp: time.Now().Unix(),
			})

		case message := <-h.broadcast:
			h.broadcastToSession(message.SessionID, message)
		}
	}
}

// broadcastToSession sends a message to all clients in a session
func (h *WebSocketHub) broadcastToSession(sessionID string, message *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.sessions[sessionID]; ok {
		for _, client := range clients {
			select {
			case client.Send <- message:
			default:
				// Client's send channel is full, skip
				h.logger.Warn("Client send buffer full",
					zap.String("clientId", client.ID),
				)
			}
		}
	}
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Hub.logger.Error("Failed to unmarshal message", zap.Error(err))
			continue
		}

		// Set metadata
		msg.SessionID = c.SessionID
		msg.UserID = c.UserID
		msg.Timestamp = time.Now().Unix()

		// Handle different message types
		c.handleMessage(&msg)
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				c.Hub.logger.Error("Failed to marshal message", zap.Error(err))
				continue
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WebSocketClient) handleMessage(msg *WebSocketMessage) {
	ctx := context.Background()

	switch msg.Type {
	case "presence_update":
		// Update presence in database
		req := &models.UpdatePresenceRequest{}

		if status, ok := msg.Data["status"].(string); ok {
			s := models.ParticipantStatus(status)
			req.Status = &s
		}
		if scene, ok := msg.Data["currentScene"].(string); ok {
			req.CurrentScene = &scene
		}
		if obj, ok := msg.Data["selectedObject"].(string); ok {
			req.SelectedObject = &obj
		}

		if err := c.Hub.service.UpdatePresence(ctx, c.SessionID, c.UserID, req); err != nil {
			c.Hub.logger.Error("Failed to update presence", zap.Error(err))
		}

		// Broadcast to other participants
		c.Hub.broadcast <- msg

	case "scene_update":
		// Broadcast scene update to all participants
		c.Hub.broadcast <- msg

	case "object_transform":
		// Broadcast object transform to all participants
		c.Hub.broadcast <- msg

	case "chat_message":
		// Broadcast chat message
		c.Hub.broadcast <- msg

	default:
		c.Hub.logger.Warn("Unknown message type", zap.String("type", msg.Type))
	}
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub     *WebSocketHub
	service *service.SessionService
	logger  *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *WebSocketHub, service *service.SessionService, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:     hub,
		service: service,
		logger:  logger,
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
// GET /api/v1/sessions/:id/ws
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Param("id")

	// Get user from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify session exists and user is a participant
	isParticipant, err := h.service.IsParticipant(c.Request.Context(), sessionID, userID.(string))
	if err != nil {
		h.logger.Error("Failed to check participant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a participant in this session"})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket", zap.Error(err))
		return
	}

	// Create client
	client := &WebSocketClient{
		ID:        generateClientID(sessionID, userID.(string)),
		SessionID: sessionID,
		UserID:    userID.(string),
		Conn:      conn,
		Send:      make(chan *WebSocketMessage, 256),
		Hub:       h.hub,
	}

	// Register client
	h.hub.register <- client

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}

// IsParticipant checks if user is a participant (helper method)
func (s *service.SessionService) IsParticipant(ctx context.Context, sessionID, userID string) (bool, error) {
	participant, err := s.GetParticipantBySessionAndUser(ctx, sessionID, userID)
	if err != nil {
		if err == repository.ErrParticipantNotFound {
			return false, nil
		}
		return false, err
	}
	return participant != nil, nil
}

// GetParticipantBySessionAndUser is a helper method
func (s *SessionService) GetParticipantBySessionAndUser(ctx context.Context, sessionID, userID string) (*models.Participant, error) {
	// This would use participantRepo.GetBySessionAndUser
	// For now, we'll use a simpler check
	participants, err := s.GetParticipants(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	for _, p := range participants.Participants {
		if p.UserID == userID && p.LeftAt == nil {
			return &p, nil
		}
	}

	return nil, repository.ErrParticipantNotFound
}

// generateClientID generates a unique client ID
func generateClientID(sessionID, userID string) string {
	return sessionID + ":" + userID + ":" + time.Now().Format("20060102150405")
}
