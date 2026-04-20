package ratelimit

import (
	"testing"
	"time"
)

func TestAllowWithinLimit(t *testing.T) {
	rl := NewRateLimiter(2)
	key := uint64(12345)
	lim := 5
	window := time.Minute

	for i := 0; i < lim; i++ {
		allowed := rl.Allow(key, lim, window)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestDenyOverLimit(t *testing.T) {
	rl := NewRateLimiter(2)
	key := uint64(12345)
	lim := 3
	window := time.Minute

	for i := 0; i < lim; i++ {
		rl.Allow(key, lim, window)
	}
	allowed := rl.Allow(key, lim, window)
	if allowed {
		t.Fatal("request over limit should be denied")
	}
}

func TestDifferentKeys(t *testing.T) {
	rl := NewRateLimiter(2)
	lim := 2
	window := time.Minute

	rl.Allow(1, lim, window)
	rl.Allow(1, lim, window)

	allowed := rl.Allow(2, lim, window)
	if !allowed {
		t.Fatal("different key should have its own bucket")
	}
}