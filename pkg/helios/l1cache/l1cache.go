package l1cache

import (
	"sync"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/klauspost/compress/zstd"
)

type Cache interface {
	Put(hash types.Hash, raw []byte) (storedBytes int, compressed bool)
	Get(hash types.Hash) (data []byte, ok bool)
	Stats() CacheStats
}

type CacheStats struct {
	Hits      uint64
	Misses    uint64
	Evictions uint64
	SizeBytes uint64
	Items     uint64
}

type Config struct {
	CapacityBytes        int64 // ≤0 means cache is disabled
	CompressionThreshold int   // below threshold: do not compress; ≤0 means always try compress
}

type entry struct {
	k          string
	data       []byte // may be compressed
	rawSize    int
	compressed bool
}

type cache struct {
	mu        sync.Mutex
	capBytes  int64
	sizeBytes int64

	order   []string
	entries map[string]*entry

	enc       *zstd.Encoder
	dec       *zstd.Decoder
	threshold int

	stats CacheStats
}

func New(cfg Config) (Cache, error) {
	if cfg.CapacityBytes < 0 {
		cfg.CapacityBytes = 0
	}
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	return &cache{
		capBytes:  cfg.CapacityBytes,
		entries:   make(map[string]*entry),
		order:     make([]string, 0, 128),
		enc:       enc,
		dec:       dec,
		threshold: cfg.CompressionThreshold,
	}, nil
}

func (c *cache) key(h types.Hash) string { return h.String() }

func (c *cache) Put(h types.Hash, raw []byte) (int, bool) {
	if c.capBytes == 0 {
		return 0, false
	}
	k := c.key(h)
	var store []byte
	compressed := false
	tryCompress := c.threshold <= 0 || len(raw) >= c.threshold

	c.mu.Lock()
	defer c.mu.Unlock()

	if tryCompress {
		comp := c.enc.EncodeAll(raw, nil)
		if len(comp) < len(raw) {
			store = comp
			compressed = true
		}
	}
	if store == nil {
		store = make([]byte, len(raw))
		copy(store, raw)
	}
	need := int64(len(store))
	if need > c.capBytes {
		// object larger than cache capacity → skip caching
		return 0, false
	}

	// if already exists, remove old entry and free space
	if old, ok := c.entries[k]; ok {
		c.sizeBytes -= int64(len(old.data))
		c.deleteFromOrder(k)
		delete(c.entries, k)
		c.stats.Items--
	}

	// evict until there is enough space (FIFO)
	for c.sizeBytes+need > c.capBytes && len(c.order) > 0 {
		evictK := c.order[0]
		c.order = c.order[1:]
		if e := c.entries[evictK]; e != nil {
			c.sizeBytes -= int64(len(e.data))
			delete(c.entries, evictK)
			c.stats.Evictions++
			c.stats.Items--
		}
	}

	ent := &entry{k: k, data: store, rawSize: len(raw), compressed: compressed}
	c.entries[k] = ent
	c.order = append(c.order, k)
	c.sizeBytes += need
	c.stats.Items++
	c.stats.SizeBytes = uint64(c.sizeBytes)
	return len(store), compressed
}

func (c *cache) Get(h types.Hash) ([]byte, bool) {
	if c.capBytes == 0 {
		return nil, false
	}
	k := c.key(h)

	c.mu.Lock()
	defer c.mu.Unlock()

	ent, ok := c.entries[k]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	if ent.compressed {
		dec, err := c.dec.DecodeAll(ent.data, nil)
		if err != nil {
			// decompression failed, count as miss (do not panic)
			c.stats.Misses++
			return nil, false
		}
		c.stats.Hits++
		return dec, true
	}

	out := make([]byte, len(ent.data))
	copy(out, ent.data)
	c.stats.Hits++
	return out, true
}

func (c *cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.stats
	st.SizeBytes = uint64(c.sizeBytes)
	return st
}

func (c *cache) deleteFromOrder(k string) {
	for i := range c.order {
		if c.order[i] == k {
			copy(c.order[i:], c.order[i+1:])
			c.order = c.order[:len(c.order)-1]
			return
		}
	}
}
