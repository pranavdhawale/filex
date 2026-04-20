package counter

import (
	"sync"
	"testing"
)

func TestIncrement(t *testing.T) {
	c := NewDownloadCounter()
	c.Increment("f~abc123")
	c.Increment("f~abc123")
	c.Increment("f~xyz789")

	snapshot := c.Snapshot()
	if snapshot["f~abc123"] != 2 {
		t.Errorf("expected 2, got %d", snapshot["f~abc123"])
	}
	if snapshot["f~xyz789"] != 1 {
		t.Errorf("expected 1, got %d", snapshot["f~xyz789"])
	}
}

func TestConcurrentIncrement(t *testing.T) {
	c := NewDownloadCounter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Increment("f~test")
		}()
	}
	wg.Wait()
	snapshot := c.Snapshot()
	if snapshot["f~test"] != 100 {
		t.Errorf("expected 100, got %d", snapshot["f~test"])
	}
}

func TestClear(t *testing.T) {
	c := NewDownloadCounter()
	c.Increment("f~abc")
	c.Clear()
	snapshot := c.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected empty snapshot after clear, got %d entries", len(snapshot))
	}
}