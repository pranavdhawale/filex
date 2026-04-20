package ratelimit

import (
	"hash/fnv"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastTime time.Time
}

type shard struct {
	mu      sync.Mutex
	buckets map[uint64]*bucket
}

type RateLimiter struct {
	shards []shard
	mask   uint64
}

func NewRateLimiter(numShards uint64) *RateLimiter {
	if numShards == 0 {
		numShards = 64
	}
	s := uint64(1)
	for s < numShards {
		s <<= 1
	}
	rl := &RateLimiter{
		shards: make([]shard, s),
		mask:   s - 1,
	}
	for i := range rl.shards {
		rl.shards[i].buckets = make(map[uint64]*bucket)
	}
	return rl
}

func (r *RateLimiter) getShard(key uint64) *shard {
	return &r.shards[key&r.mask]
}

func (r *RateLimiter) Allow(key uint64, limit int, window time.Duration) bool {
	s := r.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	b, ok := s.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(limit), lastTime: now}
		s.buckets[key] = b
	}

	elapsed := now.Sub(b.lastTime)
	refill := float64(elapsed) / float64(window) * float64(limit)
	b.tokens += refill
	if b.tokens > float64(limit) {
		b.tokens = float64(limit)
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func HashIP(ip string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(ip))
	return h.Sum64()
}