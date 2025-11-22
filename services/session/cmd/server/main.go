package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/session/internal/config"
	"github.com/yourorg/collab/services/session/internal/database"
	"github.com/yourorg/collab/services/session/internal/handler"
	"github.com/yourorg/collab/services/session/internal/middleware"
	"github.com/yourorg/collab/services/session/internal/repository"
	"github.com/yourorg/collab/services/session/internal/service"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Session Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer database.Close(db)
	logger.Info("Connected to PostgreSQL")

	// Connect to Redis
	redisClient, err := database.NewRedisClient(
		cfg.Redis.GetRedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer database.CloseRedis(redisClient)
	logger.Info("Connected to Redis")

	// Initialize repositories
	sessionRepo := repository.NewSessionRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	redisRepo := repository.NewRedisRepository(redisClient)

	// Initialize services
	sessionService := service.NewSessionService(
		sessionRepo,
		participantRepo,
		redisRepo,
		logger,
	)

	// Initialize WebSocket hub
	wsHub := handler.NewWebSocketHub(logger, sessionService)
	go wsHub.Run()

	// Initialize handlers
	sessionHandler := handler.NewSessionHandler(sessionService, logger)
	wsHandler := handler.NewWebSocketHandler(wsHub, sessionService, logger)

	// Setup Gin router
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "session",
			"version": "1.0.0",
		})
	})

	// Readiness check endpoint
	router.GET("/ready", func(c *gin.Context) {
		// Check database connection
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database not available",
			})
			return
		}

		// Check Redis connection
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "redis not available",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no auth required)
		v1.GET("/sessions/:id", sessionHandler.GetSession)
		v1.GET("/sessions", sessionHandler.ListSessions)

		// Protected routes (auth required)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.Auth.JWTSecret, logger))
		{
			// Session management
			protected.POST("/sessions", sessionHandler.CreateSession)
			protected.PATCH("/sessions/:id", sessionHandler.UpdateSession)
			protected.DELETE("/sessions/:id", sessionHandler.DeleteSession)

			// Scene data
			protected.PUT("/sessions/:id/scene", sessionHandler.UpdateSceneData)

			// Participant management
			protected.POST("/sessions/:id/join", sessionHandler.JoinSession)
			protected.POST("/sessions/:id/leave", sessionHandler.LeaveSession)
			protected.GET("/sessions/:id/participants", sessionHandler.GetParticipants)

			// Presence
			protected.PATCH("/sessions/:id/presence", sessionHandler.UpdatePresence)

			// WebSocket
			protected.GET("/sessions/:id/ws", wsHandler.HandleWebSocket)
		}
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in goroutine
	go func() {
		logger.Info("Session Service listening",
			zap.String("address", addr),
			zap.String("mode", cfg.Server.Mode),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Session Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Session Service stopped gracefully")
}
