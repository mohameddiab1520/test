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
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Session Service...")

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "session",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"sessions": []string{}})
		})

		v1.POST("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{
				"message": "Session created successfully",
			})
		})

		v1.GET("/sessions/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{
				"id":   id,
				"name": "Sample Session",
			})
		})
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Session Service listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Session Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Session Service stopped")
}
