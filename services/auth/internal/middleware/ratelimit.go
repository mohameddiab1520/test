package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/yourorg/collab/services/auth/internal/repository"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(redisRepo *repository.RedisRepository, limit int, window time.Duration, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use user ID if authenticated, otherwise use IP
		var identifier string
		if userID, exists := c.Get("userID"); exists {
			identifier = fmt.Sprintf("user:%s", userID)
		} else {
			identifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		// Check rate limit
		allowed, err := redisRepo.CheckRateLimit(c.Request.Context(), identifier, limit, window)
		if err != nil {
			logger.Error("Failed to check rate limit", zap.Error(err))
			// Continue on error to avoid blocking legitimate requests
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d requests per %s", limit, window),
			})
			c.Abort()
			return
		}

		// Get remaining requests
		remaining, err := redisRepo.GetRateLimitRemaining(c.Request.Context(), identifier, limit)
		if err != nil {
			logger.Warn("Failed to get rate limit remaining", zap.Error(err))
		} else {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		}

		c.Next()
	}
}
