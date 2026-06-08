package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis caches generated thumbnail bytes (never originals).
type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedis builds a Redis cache.
func NewRedis(addr, password string, db int, ttl time.Duration) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db}),
		ttl:    ttl,
	}
}

// Ping verifies connectivity.
func (r *Redis) Ping(ctx context.Context) error { return r.client.Ping(ctx).Err() }

// Get returns the cached bytes for key.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, bool) {
	b, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return b, true
}

// Set stores bytes for key with the configured TTL.
func (r *Redis) Set(ctx context.Context, key string, data []byte) {
	_ = r.client.Set(ctx, key, data, r.ttl).Err()
}

// Close closes the underlying client.
func (r *Redis) Close() error { return r.client.Close() }
