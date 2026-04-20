package counter

import (
	"sync"
	"sync/atomic"
)

type DownloadCounter struct {
	counts sync.Map // string -> *atomic.Int64
}

func NewDownloadCounter() *DownloadCounter {
	return &DownloadCounter{}
}

func (c *DownloadCounter) Increment(slug string) {
	val, _ := c.counts.LoadOrStore(slug, &atomic.Int64{})
	counter := val.(*atomic.Int64)
	counter.Add(1)
}

func (c *DownloadCounter) Snapshot() map[string]int64 {
	result := make(map[string]int64)
	c.counts.Range(func(key, value any) bool {
		slug := key.(string)
		counter := value.(*atomic.Int64)
		val := counter.Load()
		if val > 0 {
			result[slug] = val
		}
		return true
	})
	return result
}

func (c *DownloadCounter) Clear() {
	c.counts.Range(func(key, _ any) bool {
		c.counts.Delete(key)
		return true
	})
}