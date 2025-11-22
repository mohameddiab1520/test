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
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Asset Service...")

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

	// API v1 routes (placeholder)
	v1 := router.Group("/api/v1")
	{
		v1.GET("/assets", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"assets": []interface{}{}})
		})
		v1.POST("/assets/upload", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Upload endpoint (to be implemented)"})
		})
		v1.GET("/assets/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Get asset endpoint (to be implemented)"})
		})
		v1.GET("/assets/:id/download", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Download endpoint (to be implemented)"})
		})
	}

	// Start server
	port := os.Getenv("PORT")
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
