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
	"github.com/yourorg/collab/services/build/internal/builder"
	"github.com/yourorg/collab/services/build/internal/config"
	"github.com/yourorg/collab/services/build/internal/database"
	"github.com/yourorg/collab/services/build/internal/handler"
	"github.com/yourorg/collab/services/build/internal/models"
	"github.com/yourorg/collab/services/build/internal/repository"
	"github.com/yourorg/collab/services/build/internal/service"
	"github.com/yourorg/collab/services/build/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize PostgreSQL database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	log.Println("Connected to PostgreSQL database")

	// Initialize repository
	repo := repository.NewPostgresRepository(db)

	// Initialize build queue
	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
	queue, err := builder.NewBuildQueue(redisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to initialize build queue: %v", err)
	}
	defer queue.Close()

	log.Println("Connected to Redis build queue")

	// Initialize S3 storage
	s3Storage, err := storage.NewS3Storage(
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3Bucket,
		cfg.S3Region,
		cfg.S3UseSSL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	log.Println("Connected to S3 storage")

	// Initialize build service
	buildService := service.NewBuildService(cfg, repo, queue, s3Storage)

	// Start build worker
	go startBuildWorker(buildService, queue, cfg.MaxConcurrentBuilds)

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

// startBuildWorker starts background workers to process build queue
func startBuildWorker(buildService *service.BuildService, queue *builder.BuildQueue, maxConcurrent int) {
	log.Printf("Starting %d build workers", maxConcurrent)

	// Create semaphore to limit concurrent builds
	semaphore := make(chan struct{}, maxConcurrent)

	for {
		// Wait for available slot
		semaphore <- struct{}{}

		// Dequeue next build
		ctx := context.Background()
		build, err := queue.Dequeue(ctx)
		if err != nil {
			log.Printf("Error dequeuing build: %v", err)
			<-semaphore
			time.Sleep(5 * time.Second)
			continue
		}

		if build == nil {
			// No builds in queue, wait and retry
			<-semaphore
			time.Sleep(2 * time.Second)
			continue
		}

		// Process build in goroutine
		go func(b *models.Build) {
			defer func() { <-semaphore }()

			log.Printf("Processing build %s for project %s", b.ID, b.ProjectID)

			if err := buildService.ExecuteBuild(ctx, b); err != nil {
				log.Printf("Build %s failed: %v", b.ID, err)
			} else {
				log.Printf("Build %s completed successfully", b.ID)
			}
		}(build)
	}
}
