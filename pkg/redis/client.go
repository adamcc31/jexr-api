package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	client     *redis.Client
	clientOnce sync.Once
	clientErr  error
)

// Config holds Redis connection configuration
type Config struct {
	URL      string // Upstash Redis URL (redis://...) or (rediss://... for TLS)
	Password string // Upstash Redis password
}

// Client returns the singleton Redis client instance.
// Returns nil if Redis is not configured or connection failed.
func Client() *redis.Client {
	return client
}

// Initialize initializes the Redis client with the given configuration.
// This should be called once at application startup.
// Safe for concurrent calls - only first call initializes.
func Initialize(cfg Config) error {
	clientOnce.Do(func() {
		if cfg.URL == "" {
			clientErr = errors.New("redis: UPSTASH_REDIS_URL not configured")
			return
		}

		// Parse the URL to extract components
		parsedURL, err := url.Parse(cfg.URL)
		if err != nil {
			clientErr = fmt.Errorf("redis: invalid URL: %w", err)
			return
		}

		// Determine TLS requirement from scheme
		useTLS := parsedURL.Scheme == "rediss"

		// Extract host and port
		addr := parsedURL.Host
		if parsedURL.Port() == "" {
			if useTLS {
				addr = parsedURL.Host + ":6379"
			}
		}

		// Use password from config (Upstash requires password)
		password := cfg.Password
		if password == "" {
			// Try to get from URL if not provided separately
			if parsedURL.User != nil {
				password, _ = parsedURL.User.Password()
			}
		}

		opts := &redis.Options{
			Addr:         addr,
			Password:     password,
			DB:           0,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 2,
		}

		// Enable TLS for Upstash (rediss://)
		if useTLS {
			opts.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		client = redis.NewClient(opts)

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			clientErr = fmt.Errorf("redis: connection failed: %w", err)
			client = nil
			return
		}
	})

	return clientErr
}

// IsAvailable checks if Redis client is initialized and connected.
func IsAvailable() bool {
	if client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return client.Ping(ctx).Err() == nil
}

// Close closes the Redis connection gracefully.
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// HealthCheck performs a health check on the Redis connection.
// Returns nil if healthy, error otherwise.
func HealthCheck(ctx context.Context) error {
	if client == nil {
		return errors.New("redis: client not initialized")
	}
	return client.Ping(ctx).Err()
}
