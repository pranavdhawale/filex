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

func TestDifferentEndpointsGetIndependentBuckets(t *testing.T) {
	rl := NewRateLimiter(2)
	ipHash := HashIP("1.2.3.4")
	uploadKey := CompositeKey(ipHash, "POST /upload/")
	apiKey := CompositeKey(ipHash, "POST /api/v1/files/init")

	// Exhaust the upload bucket (limit=3)
	for i := 0; i < 3; i++ {
		rl.Allow(uploadKey, 3, time.Minute)
	}

	// Upload bucket should be exhausted
	if rl.Allow(uploadKey, 3, time.Minute) {
		t.Fatal("upload bucket should be exhausted")
	}

	// API bucket should still have its own tokens
	if !rl.Allow(apiKey, 10, time.Minute) {
		t.Fatal("api bucket should have its own independent tokens")
	}
}

func TestCompositeKey(t *testing.T) {
	ipHash := HashIP("1.2.3.4")

	key1 := CompositeKey(ipHash, "POST /upload/")
	key2 := CompositeKey(ipHash, "POST /api/v1/files/init")
	key3 := CompositeKey(HashIP("5.6.7.8"), "POST /upload/")

	if key1 == key2 {
		t.Fatal("different endpoints with same IP should produce different keys")
	}
	if key1 == key3 {
		t.Fatal("same endpoint with different IPs should produce different keys")
	}

	// Same inputs must produce the same key deterministically
	key4 := CompositeKey(ipHash, "POST /upload/")
	if key1 != key4 {
		t.Fatal("same inputs should produce the same key deterministically")
	}
}