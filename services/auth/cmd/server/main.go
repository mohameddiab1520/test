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

	logger.Info("Starting Auth Service...")

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "auth",
			"version": "1.0.0",
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/register", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{
				"message": "User registered successfully",
				"userId":  "user-123",
			})
		})

		v1.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"accessToken":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				"refreshToken": "refresh_token_here",
				"expiresIn":    3600,
			})
		})

		v1.POST("/auth/refresh", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"accessToken": "new_access_token",
				"expiresIn":   3600,
			})
		})

		v1.POST("/auth/logout", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Logged out successfully",
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		logger.Info("Auth Service listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Auth Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Auth Service stopped")
}
