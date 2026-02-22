package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

// Allow checks if a request from the given IP is allowed under the sliding window limit.
// It uses a Redis pipeline to ensure atomicity.
func (r *RateLimiter) Allow(ctx context.Context, ip string, action string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("rate:%s:%s", action, ip)
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	// Create a unique member ID for the ZSet entry
	memberID := uuid.New().String()

	pipe := r.client.Pipeline()

	// 1. Remove entries older than the window
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprint(windowStart))

	// 2. Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: memberID,
	})

	// 3. Count remaining entries
	countCmd := pipe.ZCard(ctx, key)

	// 4. Set expiry on the key to clean up inactive IPs
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	count := countCmd.Val()

	return count <= int64(limit), nil
}
