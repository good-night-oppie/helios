package l1cache

import (
	"bytes"
	"testing"

	"github.com/good-night-oppie/helios-engine/internal/util"
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

func TestEvictOnDecompressionFailure(t *testing.T) {
	cIface, err := New(Config{CapacityBytes: 1 << 20, CompressionThreshold: -1})
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	c := cIface.(*cache)
	raw := bytes.Repeat([]byte("a"), 1024)
	h := hOf(t, raw)
	c.Put(h, raw)

	// Corrupt stored data
	ck := h.String()
	c.mu.Lock()
	if ent, ok := c.entries[ck]; ok {
		ent.data[0] ^= 0xff
	}
	c.mu.Unlock()

	if _, ok := c.Get(h); ok {
		t.Fatalf("expected get to fail")
	}
	if s := c.Stats(); s.Misses != 1 || s.Items != 0 {
		t.Fatalf("unexpected stats after failure: %+v", s)
	}
	if _, ok := c.Get(h); ok {
		t.Fatalf("entry should be evicted")
	}
	if s := c.Stats(); s.Misses != 2 {
		t.Fatalf("misses should increment on subsequent miss, got %+v", s)
	}
}
