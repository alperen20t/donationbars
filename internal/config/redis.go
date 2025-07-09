package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client  *redis.Client
	Enabled bool
}

// InitRedis initializes Redis connection based on configuration
func InitRedis(cfg RedisConfig, timeout time.Duration) (*RedisClient, error) {
	if !cfg.Enabled {
		slog.Info("Redis is disabled, using fallback rate limiting")
		return &RedisClient{
			Client:  nil,
			Enabled: false,
		}, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("Successfully connected to Redis",
		"addr", cfg.Addr,
		"db", cfg.DB)

	return &RedisClient{
		Client:  rdb,
		Enabled: true,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// IsEnabled returns whether Redis is enabled and available
func (r *RedisClient) IsEnabled() bool {
	return r.Enabled && r.Client != nil
}
