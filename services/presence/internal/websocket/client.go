package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourorg/collab/services/presence/internal/models"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	SessionID string
	UserID    string
	Logger    *zap.Logger
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var update models.PresenceUpdate
	if err := json.Unmarshal(message, &update); err != nil {
		c.Logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	if update.Type == "presence" && update.Presence != nil {
		// Update presence in service
		req := &models.UpdateStatusRequest{
			SessionID:       update.SessionID,
			UserID:          c.UserID,
			Status:          update.Presence.Status,
			CurrentScene:    update.Presence.CurrentScene,
			SelectedObject:  update.Presence.SelectedObject,
			CursorPosition:  update.Presence.CursorPosition,
			CameraTransform: update.Presence.CameraTransform,
			IsTyping:        update.Presence.IsTyping,
		}

		if err := c.Hub.service.UpdatePresence(context.Background(), req); err != nil {
			c.Logger.Error("Failed to update presence", zap.Error(err))
			return
		}

		// Broadcast to other clients
		broadcastMsg := models.PresenceUpdateMessage{
			Type:     "presence_update",
			UserID:   c.UserID,
			Presence: update.Presence,
		}

		c.Hub.BroadcastToSession(c.SessionID, broadcastMsg)
	}
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	userID := r.URL.Query().Get("userId")

	if sessionID == "" || userID == "" {
		conn.Close()
		return
	}

	logger, _ := zap.NewProduction()

	client := &Client{
		Hub:       hub,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		SessionID: sessionID,
		UserID:    userID,
		Logger:    logger,
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
