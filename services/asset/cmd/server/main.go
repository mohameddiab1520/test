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

	"github.com/yourorg/collab/services/asset/internal/config"
	"github.com/yourorg/collab/services/asset/internal/database"
	"github.com/yourorg/collab/services/asset/internal/handler"
	"github.com/yourorg/collab/services/asset/internal/repository"
	"github.com/yourorg/collab/services/asset/internal/service"
	"github.com/yourorg/collab/services/asset/internal/storage"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Asset Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Connect to PostgreSQL
	db, err := database.ConnectPostgres(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL")

	// Initialize S3 storage
	s3Storage, err := storage.NewS3Storage(
		cfg.S3.Endpoint,
		cfg.S3.AccessKeyID,
		cfg.S3.SecretAccessKey,
		cfg.S3.BucketName,
		cfg.S3.Region,
		cfg.S3.UseSSL,
	)
	if err != nil {
		logger.Fatal("Failed to initialize S3 storage", zap.Error(err))
	}
	logger.Info("Connected to S3 storage")

	// Initialize repository, service, and handler
	assetRepo := repository.NewPostgresRepository(db)
	assetService := service.NewAssetService(assetRepo, s3Storage, logger)
	assetHandler := handler.NewAssetHandler(assetService, logger)

	// Setup Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "asset",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Asset routes
		v1.POST("/assets/upload-url", assetHandler.RequestUploadURL)
		v1.POST("/assets/:id/confirm", assetHandler.ConfirmUpload)
		v1.GET("/assets", assetHandler.ListAssets)
		v1.GET("/assets/:id", assetHandler.GetAsset)
		v1.GET("/assets/:id/download", assetHandler.GenerateDownloadURL)
		v1.PUT("/assets/:id", assetHandler.UpdateAsset)
		v1.DELETE("/assets/:id", assetHandler.DeleteAsset)
	}

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8083"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		logger.Info("Asset Service listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Asset Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Asset Service stopped")
}
