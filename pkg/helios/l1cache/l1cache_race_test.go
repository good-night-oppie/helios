package l1cache_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
)

// TestConcurrentAccess ensures cache operations are safe for concurrent use.
func TestConcurrentAccess(t *testing.T) {
	c, err := l1cache.New(l1cache.Config{CapacityBytes: 1 << 20, CompressionThreshold: -1})
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	raw := bytes.Repeat([]byte("x"), 1024)
	h := hOf(t, raw)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			c.Put(h, raw)
			wg.Done()
		}()
		go func() {
			c.Get(h)
			wg.Done()
		}()
	}
	wg.Wait()
}
