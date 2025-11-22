package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unity-collab/presence-service/internal/config"
	"github.com/unity-collab/presence-service/internal/database"
	"github.com/unity-collab/presence-service/internal/handler"
	"github.com/unity-collab/presence-service/internal/middleware"
	"github.com/unity-collab/presence-service/internal/repository"
	"github.com/unity-collab/presence-service/internal/service"
	"github.com/unity-collab/presence-service/internal/websocket"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg := config.Load()

	// Initialize databases
	db, err := database.NewPostgres(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize repositories
	presenceRepo := repository.NewPresenceRepository(db, redisClient)
	activityRepo := repository.NewActivityRepository(db)

	// Initialize services
	presenceService := service.NewPresenceService(presenceRepo, activityRepo, redisClient, logger)

	// Initialize WebSocket hub
	hub := websocket.NewHub(presenceService, logger)
	go hub.Run()

	// Initialize HTTP handler
	presenceHandler := handler.NewPresenceHandler(presenceService, hub, logger)

	// Setup router
	router := setupRouter(cfg, presenceHandler, hub)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServicePort),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Presence Service started", zap.Int("port", cfg.ServicePort))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func setupRouter(cfg *config.Config, presenceHandler *handler.PresenceHandler, hub *websocket.Hub) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "presence-service",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Metrics
	router.GET("/metrics", presenceHandler.Metrics)

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		websocket.ServeWS(hub, c.Writer, c.Request)
	})

	// API routes
	api := router.Group("/api/v1")
	{
		presence := api.Group("/presence")
		{
			presence.GET("/sessions/:sessionId", presenceHandler.GetSessionPresence)
			presence.GET("/users/:userId", presenceHandler.GetUserPresence)
			presence.PUT("/status", presenceHandler.UpdatePresenceStatus)
			presence.POST("/activity-feed/:sessionId", presenceHandler.GetActivityFeed)
			presence.DELETE("/sessions/:sessionId/:userId", presenceHandler.ClearPresence)
		}
	}

	return router
}
