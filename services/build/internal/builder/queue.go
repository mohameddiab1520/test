package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yourorg/collab/services/build/internal/models"
)

const (
	buildQueueKey     = "build:queue"
	buildProcessingKey = "build:processing"
)

// BuildQueue manages build job queue using Redis
type BuildQueue struct {
	client *redis.Client
}

// NewBuildQueue creates a new build queue
func NewBuildQueue(redisAddr, password string) (*BuildQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &BuildQueue{client: client}, nil
}

// Enqueue adds a build to the queue
func (q *BuildQueue) Enqueue(ctx context.Context, build *models.Build) error {
	data, err := json.Marshal(build)
	if err != nil {
		return fmt.Errorf("failed to marshal build: %w", err)
	}

	if err := q.client.RPush(ctx, buildQueueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue build: %w", err)
	}

	return nil
}

// Dequeue retrieves the next build from the queue
func (q *BuildQueue) Dequeue(ctx context.Context) (*models.Build, error) {
	// Move build from queue to processing
	result, err := q.client.BLMove(ctx, buildQueueKey, buildProcessingKey, "LEFT", "RIGHT", 0).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No builds in queue
		}
		return nil, fmt.Errorf("failed to dequeue build: %w", err)
	}

	var build models.Build
	if err := json.Unmarshal([]byte(result), &build); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build: %w", err)
	}

	return &build, nil
}

// Complete marks a build as completed and removes from processing
func (q *BuildQueue) Complete(ctx context.Context, buildID string) error {
	// Remove from processing queue
	builds, err := q.client.LRange(ctx, buildProcessingKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get processing builds: %w", err)
	}

	for _, data := range builds {
		var build models.Build
		if err := json.Unmarshal([]byte(data), &build); err != nil {
			continue
		}

		if build.ID == buildID {
			if err := q.client.LRem(ctx, buildProcessingKey, 1, data).Err(); err != nil {
				return fmt.Errorf("failed to remove build from processing: %w", err)
			}
			break
		}
	}

	return nil
}

// GetQueueSize returns the number of builds in queue
func (q *BuildQueue) GetQueueSize(ctx context.Context) (int64, error) {
	size, err := q.client.LLen(ctx, buildQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return size, nil
}

// GetProcessingCount returns the number of builds currently processing
func (q *BuildQueue) GetProcessingCount(ctx context.Context) (int64, error) {
	count, err := q.client.LLen(ctx, buildProcessingKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get processing count: %w", err)
	}
	return count, nil
}

// Close closes the Redis connection
func (q *BuildQueue) Close() error {
	return q.client.Close()
}
