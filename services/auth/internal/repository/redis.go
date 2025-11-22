package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourorg/collab/services/auth/internal/models"
)

// RedisRepository handles caching and session management
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// CacheUser caches user data
func (r *RedisRepository) CacheUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s", user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to cache user: %w", err)
	}

	return nil
}

// GetCachedUser retrieves cached user data
func (r *RedisRepository) GetCachedUser(ctx context.Context, userID string) (*models.User, error) {
	key := fmt.Sprintf("user:%s", userID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached user: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

// InvalidateUser invalidates user cache
func (r *RedisRepository) InvalidateUser(ctx context.Context, userID string) error {
	key := fmt.Sprintf("user:%s", userID)
	return r.client.Del(ctx, key).Err()
}

// StoreSession stores user session
func (r *RedisRepository) StoreSession(ctx context.Context, sessionID, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)

	data := map[string]interface{}{
		"userId":    userID,
		"createdAt": time.Now().Unix(),
	}

	err := r.client.HSet(ctx, key, data).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	err = r.client.Expire(ctx, key, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set session expiry: %w", err)
	}

	return nil
}

// GetSession retrieves session data
func (r *RedisRepository) GetSession(ctx context.Context, sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	userID, err := r.client.HGet(ctx, key, "userId").Result()
	if err == redis.Nil {
		return "", fmt.Errorf("session not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	return userID, nil
}

// DeleteSession deletes a session
func (r *RedisRepository) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Del(ctx, key).Err()
}

// DeleteAllUserSessions deletes all sessions for a user
func (r *RedisRepository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	pattern := "session:*"

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan sessions: %w", err)
		}

		for _, key := range keys {
			storedUserID, err := r.client.HGet(ctx, key, "userId").Result()
			if err == nil && storedUserID == userID {
				r.client.Del(ctx, key)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// CheckRateLimit checks and increments rate limit counter
func (r *RedisRepository) CheckRateLimit(ctx context.Context, identifier string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)

	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	if count == 1 {
		// First request, set expiry
		err = r.client.Expire(ctx, key, window).Err()
		if err != nil {
			return false, fmt.Errorf("failed to set rate limit expiry: %w", err)
		}
	}

	return count <= int64(limit), nil
}

// GetRateLimitRemaining gets remaining requests
func (r *RedisRepository) GetRateLimitRemaining(ctx context.Context, identifier string, limit int) (int, error) {
	key := fmt.Sprintf("ratelimit:%s", identifier)

	count, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return limit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit: %w", err)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// BlacklistToken adds a token to blacklist
func (r *RedisRepository) BlacklistToken(ctx context.Context, token string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:token:%s", token)
	return r.client.Set(ctx, key, "1", ttl).Err()
}

// IsTokenBlacklisted checks if token is blacklisted
func (r *RedisRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:token:%s", token)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}
