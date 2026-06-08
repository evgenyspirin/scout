package cache

import (
	"context"
	"strings"

	"scout/internal/application/thumbapp"
)

// TwoLevel combines a hot in-memory LRU with a Redis byte cache.
// It satisfies thumbapp.Cache.
type TwoLevel struct {
	lru   *LRU
	redis *Redis
}

// NewTwoLevel builds the two-level cache.
func NewTwoLevel(lru *LRU, redis *Redis) *TwoLevel {
	return &TwoLevel{lru: lru, redis: redis}
}

// contentTypeFromKey derives the content type implied by the cache key format.
func contentTypeFromKey(key string) string {
	if strings.HasSuffix(key, "fmtwebp") {
		return "image/webp"
	}
	return "image/jpeg"
}

// Get checks the LRU first, then Redis (promoting hits into the LRU).
func (c *TwoLevel) Get(ctx context.Context, key string) (thumbapp.Thumbnail, string, bool) {
	if t, ok := c.lru.Get(key); ok {
		return t, "lru", true
	}
	if b, ok := c.redis.Get(ctx, key); ok {
		t := thumbapp.Thumbnail{Data: b, ContentType: contentTypeFromKey(key)}
		c.lru.Set(key, t)
		return t, "redis", true
	}
	return thumbapp.Thumbnail{}, "", false
}

// Set writes through to both levels.
func (c *TwoLevel) Set(ctx context.Context, key string, t thumbapp.Thumbnail) {
	c.lru.Set(key, t)
	c.redis.Set(ctx, key, t.Data)
}
