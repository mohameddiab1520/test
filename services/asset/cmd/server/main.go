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

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "asset",
			"version": "1.0.0",
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/assets", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"assets": []string{}})
		})

		v1.POST("/assets/upload-url", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"uploadUrl": "https://s3.example.com/upload",
				"assetId":   "asset-123",
			})
		})

		v1.GET("/assets/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":       id,
				"fileName": "sample.fbx",
				"status":   "ready",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
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
