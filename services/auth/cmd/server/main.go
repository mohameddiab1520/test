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

	"github.com/yourorg/collab/services/auth/internal/config"
	"github.com/yourorg/collab/services/auth/internal/database"
	"github.com/yourorg/collab/services/auth/internal/handler"
	"github.com/yourorg/collab/services/auth/internal/middleware"
	"github.com/yourorg/collab/services/auth/internal/repository"
	"github.com/yourorg/collab/services/auth/internal/service"
	"github.com/yourorg/collab/services/auth/pkg/jwt"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Auth Service...")

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
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	redisRepo := repository.NewRedisRepository(redisClient)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
		cfg.JWT.Issuer,
	)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		redisRepo,
		jwtManager,
		logger,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, logger)

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
			"service": "auth",
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
		// Public auth routes with rate limiting
		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimitMiddleware(redisRepo, 10, 1*time.Minute, logger))
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/validate", authHandler.ValidateToken)
		}

		// Protected routes
		protected := v1.Group("/auth")
		protected.Use(middleware.AuthMiddleware(jwtManager, logger))
		{
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/me", authHandler.GetCurrentUser)
		}
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in goroutine
	go func() {
		logger.Info("Auth Service listening",
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

	logger.Info("Shutting down Auth Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Auth Service stopped gracefully")
}
