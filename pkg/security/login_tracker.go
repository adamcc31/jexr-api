package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-recruitment-backend/pkg/redis"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// LoginTrackerConfig holds configuration for login tracking
type LoginTrackerConfig struct {
	MaxAttempts   int           // Maximum failed attempts before block (default: 5)
	AttemptWindow time.Duration // Time window for tracking attempts (default: 15min)
	BlockDuration time.Duration // How long to block after max attempts (default: 15min)
	UseIPTracking bool          // Also track by IP address (default: true)
}

// DefaultLoginTrackerConfig returns sensible defaults
func DefaultLoginTrackerConfig() LoginTrackerConfig {
	return LoginTrackerConfig{
		MaxAttempts:   5,
		AttemptWindow: 15 * time.Minute,
		BlockDuration: 15 * time.Minute,
		UseIPTracking: true,
	}
}

// LoginTracker tracks failed login attempts and enforces blocks
type LoginTracker struct {
	config LoginTrackerConfig
	logger *SecurityLogger
}

// NewLoginTracker creates a new login tracker with the given config
func NewLoginTracker(config LoginTrackerConfig) *LoginTracker {
	return &LoginTracker{
		config: config,
		logger: DefaultLogger(),
	}
}

// Redis key patterns
const (
	failLoginUserPrefix    = "fail:login:user:"
	failLoginIPPrefix      = "fail:login:ip:"
	blockedLoginUserPrefix = "blocked:login:user:"
	blockedLoginIPPrefix   = "blocked:login:ip:"
)

// Lua script for atomic increment with TTL on first set
// KEYS[1] = counter key
// ARGV[1] = TTL in seconds
// Returns: current count after increment
const incrWithTTLScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`

// IsBlocked checks if the given email or IP is currently blocked
// Returns (blocked, error)
func (lt *LoginTracker) IsBlocked(ctx context.Context, email, ip string) (bool, error) {
	client := redis.Client()
	if client == nil {
		// Redis not available - fail open (don't block)
		// In production, consider fail-closed behavior
		return false, nil
	}

	// Check user block
	userKey := blockedLoginUserPrefix + email
	exists, err := client.Exists(ctx, userKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check user block: %w", err)
	}
	if exists > 0 {
		return true, nil
	}

	// Check IP block if enabled
	if lt.config.UseIPTracking && ip != "" {
		ipKey := blockedLoginIPPrefix + ip
		exists, err := client.Exists(ctx, ipKey).Result()
		if err != nil {
			return false, fmt.Errorf("failed to check IP block: %w", err)
		}
		if exists > 0 {
			return true, nil
		}
	}

	return false, nil
}

// RecordFailedAttempt records a failed login attempt and returns whether the user should be blocked
// Returns (shouldBlock, currentAttempts, error)
func (lt *LoginTracker) RecordFailedAttempt(ctx context.Context, email, ip, userAgent, requestID string) (bool, int, error) {
	client := redis.Client()
	if client == nil {
		// Redis not available - can't track attempts
		return false, 0, errors.New("redis not available for login tracking")
	}

	ttlSeconds := int(lt.config.AttemptWindow.Seconds())

	// Increment user counter atomically
	userKey := failLoginUserPrefix + email
	userCount, err := lt.atomicIncrement(ctx, client, userKey, ttlSeconds)
	if err != nil {
		return false, 0, fmt.Errorf("failed to increment user counter: %w", err)
	}

	// Also track by IP if enabled
	if lt.config.UseIPTracking && ip != "" {
		ipKey := failLoginIPPrefix + ip
		_, _ = lt.atomicIncrement(ctx, client, ipKey, ttlSeconds) // Best effort
	}

	// Log the failed attempt
	lt.logger.LogLoginFailed(ctx, email, ip, userAgent, requestID, "invalid_credentials")

	// Check if we should create a block
	if userCount >= lt.config.MaxAttempts {
		if err := lt.createBlock(ctx, email, ip, requestID); err != nil {
			return true, userCount, fmt.Errorf("failed to create block: %w", err)
		}
		return true, userCount, nil
	}

	return false, userCount, nil
}

// atomicIncrement performs an atomic increment with TTL using Lua script
func (lt *LoginTracker) atomicIncrement(ctx context.Context, client *goredis.Client, key string, ttlSeconds int) (int, error) {
	result, err := client.Eval(ctx, incrWithTTLScript, []string{key}, ttlSeconds).Result()
	if err != nil {
		return 0, err
	}
	count, ok := result.(int64)
	if !ok {
		return 0, errors.New("unexpected result type from Lua script")
	}
	return int(count), nil
}

// createBlock creates a temporary block for the user (and optionally IP)
func (lt *LoginTracker) createBlock(ctx context.Context, email, ip, requestID string) error {
	client := redis.Client()
	if client == nil {
		return errors.New("redis not available")
	}

	blockTTL := lt.config.BlockDuration

	// Block user
	userBlockKey := blockedLoginUserPrefix + email
	if err := client.Set(ctx, userBlockKey, "1", blockTTL).Err(); err != nil {
		return fmt.Errorf("failed to set user block: %w", err)
	}

	// Block IP if enabled
	if lt.config.UseIPTracking && ip != "" {
		ipBlockKey := blockedLoginIPPrefix + ip
		if err := client.Set(ctx, ipBlockKey, "1", blockTTL).Err(); err != nil {
			// Log but don't fail - user is already blocked
			lt.logger.zapLogger.Warn("failed to set IP block",
				zap.Error(err),
			)
		}
	}

	// Log block creation
	lt.logger.LogBlockCreated(ctx, "email", email, ip, requestID, int(blockTTL.Minutes()))

	return nil
}

// ClearAttempts clears failed login attempts on successful login
func (lt *LoginTracker) ClearAttempts(ctx context.Context, email, ip string) error {
	client := redis.Client()
	if client == nil {
		return nil // Nothing to clear if Redis not available
	}

	// Clear user counter
	userKey := failLoginUserPrefix + email
	if err := client.Del(ctx, userKey).Err(); err != nil {
		return fmt.Errorf("failed to clear user attempts: %w", err)
	}

	// Clear IP counter if applicable
	if lt.config.UseIPTracking && ip != "" {
		ipKey := failLoginIPPrefix + ip
		_ = client.Del(ctx, ipKey).Err() // Best effort
	}

	return nil
}

// GetRemainingAttempts returns how many attempts remain before a block
// Returns (remaining, error)
func (lt *LoginTracker) GetRemainingAttempts(ctx context.Context, email string) (int, error) {
	client := redis.Client()
	if client == nil {
		return lt.config.MaxAttempts, nil // If no Redis, return max
	}

	userKey := failLoginUserPrefix + email
	count, err := client.Get(ctx, userKey).Int()
	if err == goredis.Nil {
		return lt.config.MaxAttempts, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get attempt count: %w", err)
	}

	remaining := lt.config.MaxAttempts - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// GetBlockTTL returns how long until the block expires
// Returns (duration, blocked, error)
func (lt *LoginTracker) GetBlockTTL(ctx context.Context, email string) (time.Duration, bool, error) {
	client := redis.Client()
	if client == nil {
		return 0, false, nil
	}

	userBlockKey := blockedLoginUserPrefix + email
	ttl, err := client.TTL(ctx, userBlockKey).Result()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get block TTL: %w", err)
	}

	if ttl < 0 {
		return 0, false, nil // Not blocked
	}

	return ttl, true, nil
}
