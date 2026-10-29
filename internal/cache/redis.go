package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"ai-hub-platform/internal/model"
)

// Cache defines the caching interface for summaries.
type Cache interface {
	GetSummary(ctx context.Context, text, lang string) (*model.SummarizeResponse, error)
	SetSummary(ctx context.Context, text, lang string, resp *model.SummarizeResponse, ttl time.Duration) error
	Ping(ctx context.Context) error
	Close() error
}

type redisCache struct {
	client *redis.Client
}

// NewRedis creates a new Redis cache instance using go-redis/v9.
func NewRedis(addr, password string) Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &redisCache{client: rdb}
}

func makeCacheKey(text, lang string) string {
	hasher := sha256.New()
	hasher.Write([]byte(lang))
	hasher.Write([]byte(":"))
	hasher.Write([]byte(text))
	sum := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("summary:%s", sum)
}

// GetSummary retrieves a cached summary response if present.
func (r *redisCache) GetSummary(ctx context.Context, text, lang string) (*model.SummarizeResponse, error) {
	key := makeCacheKey(text, lang)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Cache miss
			return nil, nil
		}
		slog.Warn("redis get failed", "key", key, "error", err)
		return nil, err
	}

	var resp model.SummarizeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		slog.Warn("failed to decode cached summary", "key", key, "error", err)
		return nil, err
	}

	resp.Provider = "cache"
	return &resp, nil
}

// SetSummary stores a summary response in the cache with the given TTL.
func (r *redisCache) SetSummary(ctx context.Context, text, lang string, resp *model.SummarizeResponse, ttl time.Duration) error {
	key := makeCacheKey(text, lang)

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		slog.Warn("redis set failed", "key", key, "error", err)
		return err
	}

	return nil
}

// Ping checks connectivity with Redis server.
func (r *redisCache) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.client.Ping(pingCtx).Err()
}

// Close closes the Redis client connection.
func (r *redisCache) Close() error {
	return r.client.Close()
}
