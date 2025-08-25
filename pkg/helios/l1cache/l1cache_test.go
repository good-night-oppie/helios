package l1cache_test

import (
	"bytes"
	"testing"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

func hOf(t *testing.T, b []byte) types.Hash {
	t.Helper()
	h, err := util.HashContent(b, types.BLAKE3)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}

func TestPutGet_HitAndMiss(t *testing.T) {
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        1 << 20, // 1MiB
		CompressionThreshold: 256,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("hello world")
	h := hOf(t, raw)

	stored, compressed := c.Put(h, raw)
	if stored == 0 {
		t.Fatalf("expected store > 0")
	}
	got, ok := c.Get(h)
	if !ok || !bytes.Equal(got, raw) {
		t.Fatalf("cache get mismatch: ok=%v", ok)
	}
	// miss
	other := hOf(t, []byte("other"))
	if _, ok := c.Get(other); !ok {
		// ok: miss increments
	} else {
		t.Fatalf("expected miss")
	}

	s := c.Stats()
	if s.Hits != 1 || s.Misses != 1 {
		t.Fatalf("stats mismatch hits=%d misses=%d", s.Hits, s.Misses)
	}
	_ = compressed // ensure API compiles
}

func TestCapacityAndEviction_FIFO(t *testing.T) {
	c, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        200,    // 小容量触发淘汰
		CompressionThreshold: 100000, // 实质关闭压缩，便于直观看字节
	})

	a := bytes.Repeat([]byte("A"), 120)
	b := bytes.Repeat([]byte("B"), 120)
	ha := hOf(t, a)
	hb := hOf(t, b)

	_, _ = c.Put(ha, a) // 占约120
	_, _ = c.Put(hb, b) // 插入时需要淘汰 A（FIFO）

	if _, ok := c.Get(ha); ok {
		t.Fatalf("expected A evicted")
	}
	if got, ok := c.Get(hb); !ok || !bytes.Equal(got, b) {
		t.Fatalf("B should exist")
	}
	s := c.Stats()
	if s.Evictions < 1 {
		t.Fatalf("expect at least 1 eviction, got %d", s.Evictions)
	}
	if s.Items != 1 {
		t.Fatalf("items=1 after eviction, got %d", s.Items)
	}
}

func TestCompressionThreshold(t *testing.T) {
	c, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        4 << 20,
		CompressionThreshold: 256,
	})
	// 小对象：不压缩
	small := []byte("tiny-object")
	hs := hOf(t, small)
	storedSmall, compressedSmall := c.Put(hs, small)
	if compressedSmall {
		t.Fatalf("small should not be compressed")
	}
	if storedSmall != len(small) {
		t.Fatalf("storedSmall=%d != raw=%d", storedSmall, len(small))
	}
	// 大且可压缩：应压缩节省空间
	large := bytes.Repeat([]byte("Z"), 4096)
	hl := hOf(t, large)
	storedLarge, compressedLarge := c.Put(hl, large)
	if !compressedLarge {
		t.Fatalf("large should be compressed")
	}
	if storedLarge >= len(large) {
		t.Fatalf("compressed size should be smaller; stored=%d raw=%d", storedLarge, len(large))
	}
}

func TestStatsFields(t *testing.T) {
	c, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        1 << 20,
		CompressionThreshold: 0,
	})
	d1 := []byte("d1")
	h1 := hOf(t, d1)
	c.Put(h1, d1)
	c.Get(h1)                     // hit
	c.Get(hOf(t, []byte("miss"))) // miss
	st := c.Stats()
	if st.Hits != 1 || st.Misses != 1 || st.Items != 1 || st.SizeBytes == 0 {
		t.Fatalf("stats unexpected: %+v", st)
	}
}

func TestDisabledCache(t *testing.T) {
	// Test with CapacityBytes = 0 (disabled cache)
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        0,
		CompressionThreshold: 256,
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte("test data")
	h := hOf(t, raw)

	// Put should return 0 for disabled cache
	stored, compressed := c.Put(h, raw)
	if stored != 0 || compressed {
		t.Fatalf("disabled cache should not store: stored=%d, compressed=%v", stored, compressed)
	}

	// Get should return false for disabled cache
	_, ok := c.Get(h)
	if ok {
		t.Fatalf("disabled cache should not have data")
	}

	stats := c.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Items != 0 {
		t.Fatalf("disabled cache should have zero stats: %+v", stats)
	}
}

func TestNegativeCapacity(t *testing.T) {
	// Test with negative capacity (should be treated as 0)
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        -100,
		CompressionThreshold: 256,
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte("test")
	h := hOf(t, raw)
	stored, _ := c.Put(h, raw)
	if stored != 0 {
		t.Fatalf("negative capacity should be treated as disabled")
	}
}

func TestReplaceExistingEntry(t *testing.T) {
	// Test updating an existing entry (covers deleteFromOrder)
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        1000,
		CompressionThreshold: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}

	raw1 := []byte("first version")
	raw2 := []byte("second version updated")
	h := hOf(t, raw1) // same hash for testing replace

	// Put first version
	stored1, _ := c.Put(h, raw1)
	if stored1 == 0 {
		t.Fatal("should store first version")
	}

	// Replace with second version (same key)
	stored2, _ := c.Put(h, raw2)
	if stored2 == 0 {
		t.Fatal("should store second version")
	}

	// Verify we get the second version
	got, ok := c.Get(h)
	if !ok || !bytes.Equal(got, raw2) {
		t.Fatalf("should get second version: ok=%v, got=%s", ok, got)
	}

	// Stats should show only 1 item
	stats := c.Stats()
	if stats.Items != 1 {
		t.Fatalf("should have 1 item after replace, got %d", stats.Items)
	}
}

func TestObjectLargerThanCapacity(t *testing.T) {
	// Test when object is larger than entire cache capacity
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        100,
		CompressionThreshold: 10000, // disable compression
	})
	if err != nil {
		t.Fatal(err)
	}

	huge := bytes.Repeat([]byte("X"), 200) // larger than capacity
	h := hOf(t, huge)

	stored, compressed := c.Put(h, huge)
	if stored != 0 || compressed {
		t.Fatalf("should not cache object larger than capacity: stored=%d", stored)
	}

	_, ok := c.Get(h)
	if ok {
		t.Fatal("should not find huge object")
	}
}

func TestAlwaysCompress(t *testing.T) {
	// Test with CompressionThreshold <= 0 (always compress)
	c, err := l1cache.New(l1cache.Config{
		CapacityBytes:        1 << 20,
		CompressionThreshold: -1, // always compress
	})
	if err != nil {
		t.Fatal(err)
	}

	// Even tiny data should be attempted for compression
	tiny := []byte("x")
	h := hOf(t, tiny)

	stored, _ := c.Put(h, tiny)
	if stored == 0 {
		t.Fatal("should store tiny data")
	}
	// Note: compression may not actually compress tiny data smaller,
	// but the attempt should be made

	got, ok := c.Get(h)
	if !ok || !bytes.Equal(got, tiny) {
		t.Fatalf("should retrieve tiny data: ok=%v", ok)
	}
}
