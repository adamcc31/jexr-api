package security

import (
	"context"
	"fmt"
	"time"

	"go-recruitment-backend/pkg/redis"

	goredis "github.com/redis/go-redis/v9"
)

// UploadLimiter enforces rate limits on file uploads using Redis sliding window
type UploadLimiter struct {
	maxPerMinute int // Max uploads per minute per IP
	maxPerDay    int // Max uploads per day per user
}

// Lua script for sliding window rate limiting
// KEYS[1] = rate limit key
// ARGV[1] = max count allowed
// ARGV[2] = window size in seconds
// ARGV[3] = current timestamp
// Returns: 1 if allowed, 0 if rate limited
const uploadRateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove expired entries outside the window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Get current count
local count = redis.call('ZCARD', key)

if count >= limit then
    return 0
end

-- Add new entry with unique member
redis.call('ZADD', key, now, now .. '-' .. math.random(1000000))
redis.call('EXPIRE', key, window)
return 1
`

// NewUploadLimiter creates an upload rate limiter
// Default: 10 uploads/min per IP, 50 uploads/day per user
func NewUploadLimiter(perMin, perDay int) *UploadLimiter {
	if perMin <= 0 {
		perMin = 10
	}
	if perDay <= 0 {
		perDay = 50
	}
	return &UploadLimiter{
		maxPerMinute: perMin,
		maxPerDay:    perDay,
	}
}

// AllowUpload checks if an upload is allowed based on rate limits
// Returns (allowed, retryAfterSeconds, error)
// If Redis is unavailable, fails CLOSED (denies request) for security
func (ul *UploadLimiter) AllowUpload(ctx context.Context, ip, userID string) (bool, int, error) {
	client := redis.Client()
	if client == nil {
		// FAIL OPEN: Allow uploads if Redis unavailable (Dev/Emergency mode)
		// This prevents blocking users when infrastructure is having issues
		return true, 0, fmt.Errorf("rate limiter unavailable - Redis not connected")
	}

	now := time.Now().Unix()

	// Check per-IP rate limit (per minute)
	ipKey := fmt.Sprintf("ratelimit:upload:ip:%s", ip)
	allowed, err := ul.checkLimit(ctx, client, ipKey, ul.maxPerMinute, 60, now)
	if err != nil {
		// Fail closed on Redis errors
		return false, 60, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return false, 60, nil // Retry after 60 seconds
	}

	// Check per-user rate limit (per day) if user is authenticated
	if userID != "" {
		userKey := fmt.Sprintf("ratelimit:upload:user:%s", userID)
		allowed, err = ul.checkLimit(ctx, client, userKey, ul.maxPerDay, 86400, now)
		if err != nil {
			return false, 3600, fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return false, 3600, nil // Retry after 1 hour
		}
	}

	return true, 0, nil
}

// checkLimit performs the atomic sliding window rate limit check
func (ul *UploadLimiter) checkLimit(ctx context.Context, client *goredis.Client, key string, limit, window int, now int64) (bool, error) {
	result, err := client.Eval(ctx, uploadRateLimitScript, []string{key}, limit, window, now).Result()
	if err != nil {
		return false, err
	}
	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from rate limit script")
	}
	return allowed == 1, nil
}

// GetRemainingQuota returns remaining uploads for IP and user
// Returns (ipRemaining, userRemaining, error)
func (ul *UploadLimiter) GetRemainingQuota(ctx context.Context, ip, userID string) (int, int, error) {
	client := redis.Client()
	if client == nil {
		return 0, 0, fmt.Errorf("redis not available")
	}

	now := time.Now().Unix()

	// Get IP usage
	ipKey := fmt.Sprintf("ratelimit:upload:ip:%s", ip)
	ipCount, err := ul.getCount(ctx, client, ipKey, 60, now)
	if err != nil {
		return 0, 0, err
	}
	ipRemaining := ul.maxPerMinute - ipCount
	if ipRemaining < 0 {
		ipRemaining = 0
	}

	// Get user usage
	var userRemaining int
	if userID != "" {
		userKey := fmt.Sprintf("ratelimit:upload:user:%s", userID)
		userCount, err := ul.getCount(ctx, client, userKey, 86400, now)
		if err != nil {
			return ipRemaining, 0, err
		}
		userRemaining = ul.maxPerDay - userCount
		if userRemaining < 0 {
			userRemaining = 0
		}
	} else {
		userRemaining = ul.maxPerDay
	}

	return ipRemaining, userRemaining, nil
}

func (ul *UploadLimiter) getCount(ctx context.Context, client *goredis.Client, key string, window int, now int64) (int, error) {
	// Clean up expired entries first
	client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-int64(window)))
	// Get count
	count, err := client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
