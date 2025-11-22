package websocket

import (
	"context"
	"encoding/json"

	"github.com/unity-collab/presence-service/internal/models"
	"github.com/unity-collab/presence-service/internal/service"
	"go.uber.org/zap"
)

type Hub struct {
	clients    map[string]map[*Client]bool // sessionID -> clients
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	service    *service.PresenceService
	logger     *zap.Logger
}

type BroadcastMessage struct {
	SessionID string
	Message   []byte
}

func NewHub(service *service.PresenceService, logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		service:    service,
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	if h.clients[client.SessionID] == nil {
		h.clients[client.SessionID] = make(map[*Client]bool)
	}
	h.clients[client.SessionID][client] = true

	h.logger.Info("Client registered",
		zap.String("sessionID", client.SessionID),
		zap.String("userID", client.UserID))

	// Send joined activity
	activity := &models.ActivityFeedItem{
		SessionID:    client.SessionID,
		UserID:       client.UserID,
		ActivityType: models.ActivityJoined,
	}
	h.service.RecordActivity(context.Background(), activity)
}

func (h *Hub) unregisterClient(client *Client) {
	if clients, ok := h.clients[client.SessionID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.Send)

			if len(clients) == 0 {
				delete(h.clients, client.SessionID)
			}

			h.logger.Info("Client unregistered",
				zap.String("sessionID", client.SessionID),
				zap.String("userID", client.UserID))

			// Clear presence
			h.service.ClearPresence(context.Background(), client.SessionID, client.UserID)
		}
	}
}

func (h *Hub) broadcastMessage(message *BroadcastMessage) {
	if clients, ok := h.clients[message.SessionID]; ok {
		for client := range clients {
			select {
			case client.Send <- message.Message:
			default:
				close(client.Send)
				delete(clients, client)
			}
		}
	}
}

func (h *Hub) BroadcastToSession(sessionID string, data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("Failed to marshal broadcast message", zap.Error(err))
		return
	}

	h.broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   message,
	}
}
