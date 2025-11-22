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
	"github.com/yourorg/collab/services/build/internal/config"
	"github.com/yourorg/collab/services/build/internal/handler"
	"github.com/yourorg/collab/services/build/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize services
	buildService := service.NewBuildService(cfg)

	// Setup HTTP server
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "build",
			"version": "1.0.0",
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		builds := v1.Group("/builds")
		{
			builds.POST("", handler.CreateBuild(buildService))
			builds.GET("/:id", handler.GetBuild(buildService))
			builds.GET("", handler.ListBuilds(buildService))
			builds.POST("/:id/cancel", handler.CancelBuild(buildService))
			builds.GET("/:id/logs", handler.GetBuildLogs(buildService))
			builds.GET("/:id/artifacts", handler.GetBuildArtifacts(buildService))
		}
	}

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Build Service started on port %d", cfg.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
