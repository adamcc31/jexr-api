package middleware

import (
	"net/http"
	"sync"
	"time"

	"go-recruitment-backend/internal/delivery/http/response"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Requests per window
	Limit int
	// Time window duration
	Window time.Duration
	// Custom key extractor (default: IP-based)
	KeyFunc func(*gin.Context) string
}

// rateLimitEntry tracks request count for a key
type rateLimitEntry struct {
	count   int
	resetAt time.Time
	mu      sync.Mutex
}

// inMemoryStore for rate limiting
// TODO: Replace with Redis in production for distributed rate limiting
var (
	rateLimitStore = sync.Map{}
	cleanupOnce    sync.Once
)

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
		Limit:  100,             // 100 requests
		Window: 1 * time.Minute, // per minute
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// AuthRateLimitConfig returns strict config for authentication endpoints
func AuthRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:  5,               // 5 requests
		Window: 1 * time.Minute, // per minute
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// UploadRateLimitConfig returns config for file upload endpoints
func UploadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:  10,              // 10 uploads
		Window: 1 * time.Minute, // per minute
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}
}

// RateLimitMiddleware creates a rate limiting middleware with the given config
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	// Start cleanup goroutine once
	cleanupOnce.Do(startCleanup)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		now := time.Now()

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

		// Check limit
		if entry.count >= config.Limit {
			retryAfter := int(entry.resetAt.Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Header("X-RateLimit-Limit", string(rune(config.Limit)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", entry.resetAt.Format(time.RFC3339))
			c.Header("Retry-After", string(rune(retryAfter)))

			response.Error(c, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", nil)
			c.Abort()
			return
		}

		// Increment counter
		entry.count++

		// Add rate limit headers
		remaining := config.Limit - entry.count
		c.Header("X-RateLimit-Limit", string(rune(config.Limit)))
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))
		c.Header("X-RateLimit-Reset", entry.resetAt.Format(time.RFC3339))

		c.Next()
	}
}

// GlobalRateLimitMiddleware applies default rate limiting to all routes
func GlobalRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(DefaultRateLimitConfig())
}
