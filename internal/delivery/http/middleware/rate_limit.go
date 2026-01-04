package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/pkg/redis"
	"go-recruitment-backend/pkg/security"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Requests per window
	Limit int
	// Time window duration
	Window time.Duration
	// Custom key extractor (default: IP-based)
	KeyFunc func(*gin.Context) string
	// Key prefix for Redis (default: "rl:ip:")
	KeyPrefix string
	// Whether to fail closed (reject) when Redis is unavailable
	FailClosed bool
}

// rateLimitEntry tracks request count for a key (in-memory fallback)
type rateLimitEntry struct {
	count   int
	resetAt time.Time
	mu      sync.Mutex
}

// inMemoryStore for rate limiting (fallback when Redis unavailable)
var (
	rateLimitStore = sync.Map{}
	cleanupOnce    sync.Once
)

// Lua script for atomic increment with TTL on first set
// KEYS[1] = counter key
// ARGV[1] = TTL in seconds
// ARGV[2] = Max limit (for returning remaining)
// Returns: [current_count, ttl_remaining]
const rateLimitLuaScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('TTL', KEYS[1])
return {count, ttl}
`

// startCleanup runs a background goroutine to clean up expired entries
func startCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			now := time.Now()
			rateLimitStore.Range(func(key, value interface{}) bool {
				entry := value.(*rateLimitEntry)
				entry.mu.Lock()
				if now.After(entry.resetAt) {
					rateLimitStore.Delete(key)
				}
				entry.mu.Unlock()
				return true
			})
		}
	}()
}

// DefaultRateLimitConfig returns sensible defaults for API rate limiting
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:      100,             // 100 requests
		Window:     1 * time.Minute, // per minute
		KeyPrefix:  "rl:ip:",
		FailClosed: false, // Fail open by default for availability
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// AuthRateLimitConfig returns strict config for authentication endpoints
func AuthRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:      10,              // 10 requests
		Window:     1 * time.Minute, // per minute
		KeyPrefix:  "rl:auth:",
		FailClosed: true, // Fail closed for security-sensitive endpoints
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// LoginRateLimitConfig returns strict config specifically for login endpoint
func LoginRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:      5,               // 5 login attempts
		Window:     1 * time.Minute, // per minute
		KeyPrefix:  "rl:login:",
		FailClosed: true,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// UploadRateLimitConfig returns config for file upload endpoints
func UploadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:      10,              // 10 uploads
		Window:     1 * time.Minute, // per minute
		KeyPrefix:  "rl:upload:",
		FailClosed: false,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// RateLimitMiddleware creates a rate limiting middleware with the given config
// Uses Redis when available, falls back to in-memory when not
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	// Start cleanup goroutine once (for fallback)
	cleanupOnce.Do(startCleanup)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		fullKey := config.KeyPrefix + key
		now := time.Now()

		var count int
		var resetAt time.Time
		var err error

		// Try Redis first
		redisClient := redis.Client()
		if redisClient != nil {
			count, resetAt, err = checkRateLimitRedis(c.Request.Context(), redisClient, fullKey, config)
			if err != nil {
				// Redis error - use fallback or fail based on config
				if config.FailClosed {
					logRateLimitError(c, "redis_error", err)
					response.Error(c, http.StatusServiceUnavailable, "Service temporarily unavailable. Please try again.", nil)
					c.Abort()
					return
				}
				// Fall through to in-memory
				count, resetAt = checkRateLimitInMemory(fullKey, config, now)
			}
		} else {
			// No Redis - use in-memory fallback
			count, resetAt = checkRateLimitInMemory(fullKey, config, now)
		}

		// Check if limit exceeded
		if count > config.Limit {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.Limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC3339))
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			// Log the rate limit trigger
			logRateLimitTriggered(c)

			response.Error(c, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", nil)
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		remaining := config.Limit - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC3339))

		c.Next()
	}
}

// checkRateLimitRedis checks rate limit using Redis with atomic Lua script
func checkRateLimitRedis(ctx context.Context, client *goredis.Client, key string, config RateLimitConfig) (int, time.Time, error) {
	ttlSeconds := int(config.Window.Seconds())

	result, err := client.Eval(ctx, rateLimitLuaScript, []string{key}, ttlSeconds, config.Limit).Result()
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("redis rate limit eval failed: %w", err)
	}

	// Parse result [count, ttl]
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return 0, time.Time{}, fmt.Errorf("unexpected redis result format")
	}

	count, _ := arr[0].(int64)
	ttl, _ := arr[1].(int64)

	resetAt := time.Now().Add(time.Duration(ttl) * time.Second)

	return int(count), resetAt, nil
}

// checkRateLimitInMemory checks rate limit using in-memory store (fallback)
func checkRateLimitInMemory(key string, config RateLimitConfig, now time.Time) (int, time.Time) {
	// Get or create entry
	entryI, _ := rateLimitStore.LoadOrStore(key, &rateLimitEntry{
		count:   0,
		resetAt: now.Add(config.Window),
	})
	entry := entryI.(*rateLimitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Reset if window expired
	if now.After(entry.resetAt) {
		entry.count = 0
		entry.resetAt = now.Add(config.Window)
	}

	// Increment counter
	entry.count++

	return entry.count, entry.resetAt
}

// logRateLimitTriggered logs when rate limiting is triggered
func logRateLimitTriggered(c *gin.Context) {
	logger := security.DefaultLogger()
	if logger != nil {
		requestID, _ := c.Get("RequestID")
		reqIDStr, _ := requestID.(string)
		logger.LogRateLimitTriggered(
			c.Request.Context(),
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			reqIDStr,
			c.FullPath(),
		)
	}
}

// logRateLimitError logs Redis errors
func logRateLimitError(c *gin.Context, errorType string, err error) {
	logger := security.DefaultLogger()
	if logger != nil {
		logger.Log(c.Request.Context(), security.SecurityEvent{
			Event:       security.EventRateLimitTriggered,
			SubjectType: "system",
			IP:          c.ClientIP(),
			Details: map[string]interface{}{
				"error_type": errorType,
				"error":      err.Error(),
			},
		})
	}
}

// GlobalRateLimitMiddleware applies default rate limiting to all routes
func GlobalRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(DefaultRateLimitConfig())
}

// StrictRateLimitMiddleware for auth-sensitive endpoints
func StrictRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(AuthRateLimitConfig())
}
