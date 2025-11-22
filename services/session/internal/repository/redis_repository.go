package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourorg/collab/services/session/internal/models"
)

// RedisRepository handles Redis operations for sessions
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// CacheSession caches a session
func (r *RedisRepository) CacheSession(ctx context.Context, session *models.Session, ttl time.Duration) error {
	key := fmt.Sprintf("session:collab:%s", session.ID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to cache session: %w", err)
	}

	return nil
}

// GetCachedSession retrieves a cached session
func (r *RedisRepository) GetCachedSession(ctx context.Context, sessionID string) (*models.Session, error) {
	key := fmt.Sprintf("session:collab:%s", sessionID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached session: %w", err)
	}

	var session models.Session
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// InvalidateSession invalidates a cached session
func (r *RedisRepository) InvalidateSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:collab:%s", sessionID)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}
	return nil
}

// TrackConnection tracks a WebSocket connection
func (r *RedisRepository) TrackConnection(ctx context.Context, sessionID, userID, connectionID string) error {
	key := fmt.Sprintf("ws:connections:%s", sessionID)
	err := r.client.HSet(ctx, key, userID, connectionID).Err()
	if err != nil {
		return fmt.Errorf("failed to track connection: %w", err)
	}
	return nil
}

// UntrackConnection removes a WebSocket connection
func (r *RedisRepository) UntrackConnection(ctx context.Context, sessionID, userID string) error {
	key := fmt.Sprintf("ws:connections:%s", sessionID)
	err := r.client.HDel(ctx, key, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to untrack connection: %w", err)
	}
	return nil
}

// GetConnections gets all connections for a session
func (r *RedisRepository) GetConnections(ctx context.Context, sessionID string) (map[string]string, error) {
	key := fmt.Sprintf("ws:connections:%s", sessionID)
	connections, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}
	return connections, nil
}

// UpdatePresenceCache updates presence in cache
func (r *RedisRepository) UpdatePresenceCache(ctx context.Context, sessionID, userID string, presence map[string]interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("presence:%s:%s", sessionID, userID)

	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to marshal presence: %w", err)
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to update presence cache: %w", err)
	}

	return nil
}

// GetPresenceCache gets presence from cache
func (r *RedisRepository) GetPresenceCache(ctx context.Context, sessionID, userID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("presence:%s:%s", sessionID, userID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get presence cache: %w", err)
	}

	var presence map[string]interface{}
	err = json.Unmarshal(data, &presence)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal presence: %w", err)
	}

	return presence, nil
}

// PublishSessionEvent publishes a session event
func (r *RedisRepository) PublishSessionEvent(ctx context.Context, sessionID string, event interface{}) error {
	channel := fmt.Sprintf("session:events:%s", sessionID)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = r.client.Publish(ctx, channel, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// SubscribeToSessionEvents subscribes to session events
func (r *RedisRepository) SubscribeToSessionEvents(ctx context.Context, sessionID string) *redis.PubSub {
	channel := fmt.Sprintf("session:events:%s", sessionID)
	return r.client.Subscribe(ctx, channel)
}
