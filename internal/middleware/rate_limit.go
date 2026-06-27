package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter enforces per-tenant/IP request rate limits using Redis.
type RateLimiter struct {
	rdb       *redis.Client
	defaultRL int // requests per second
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rdb *redis.Client, defaultRL int) *RateLimiter {
	return &RateLimiter{
		rdb:       rdb,
		defaultRL: defaultRL,
	}
}

// Middleware returns a Gin handler that applies sliding-window rate limiting.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Key on tenant_id when available, fall back to client IP
		var key string
		if tid, ok := c.Get("tenant_id"); ok {
			key = fmt.Sprintf("rl:tenant:%s", tid)
		} else {
			key = fmt.Sprintf("rl:ip:%s", c.ClientIP())
		}

		now := time.Now().Unix()
		windowStart := now - 1

		// Remove entries outside the 1-second window
		if err := rl.rdb.ZRemRangeByScore(context.Background(), key, "0", strconv.FormatInt(windowStart, 10)).Err(); err != nil {
			// Log and continue — prefer availability over strict limiting on Redis failure
			fmt.Printf("Rate limit error: %v\n", err)
		}

		count, err := rl.rdb.ZCard(context.Background(), key).Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Rate limit error: %v\n", err)
		}

		if count >= int64(rl.defaultRL) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":       "RATE_LIMITED",
					"message":    "Too many requests",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		if err := rl.rdb.ZAdd(context.Background(), key, redis.Z{Score: float64(now), Member: now}).Err(); err != nil {
			fmt.Printf("Rate limit error: %v\n", err)
		}

		// Set TTL slightly beyond window so Redis cleans up stale keys
		if err := rl.rdb.Expire(context.Background(), key, 2*time.Second).Err(); err != nil {
			fmt.Printf("Rate limit error: %v\n", err)
		}

		c.Next()
	}
}
