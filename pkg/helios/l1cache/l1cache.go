// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	encMu     sync.Mutex
	decMu     sync.Mutex
	threshold int

	stats CacheStats
}

func New(cfg Config) (Cache, error) {
	if cfg.CapacityBytes < 0 {
		cfg.CapacityBytes = 0
	}
	// Note: capacity=0 is valid for a "disabled cache" that never stores anything
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

	// Compress data if appropriate
	if c.threshold <= 0 || len(raw) >= c.threshold {
		c.encMu.Lock()
		comp := c.enc.EncodeAll(raw, nil)
		c.encMu.Unlock()
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
		return 0, false // skip if too large
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing entry
	if old, ok := c.entries[k]; ok {
		c.sizeBytes -= int64(len(old.data))
		c.deleteFromOrder(k)
		delete(c.entries, k)
		c.stats.Items--
	}

	// FIFO eviction
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

	// Add new entry
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
	ent, ok := c.entries[k]
	if !ok {
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	data := make([]byte, len(ent.data))
	copy(data, ent.data)
	compressed := ent.compressed
	c.mu.Unlock()

	if compressed {
		c.decMu.Lock()
		dec, err := c.dec.DecodeAll(data, nil)
		c.decMu.Unlock()
		if err != nil {
			c.mu.Lock()
			if cur, exists := c.entries[k]; exists {
				c.sizeBytes -= int64(len(cur.data))
				c.deleteFromOrder(k)
				delete(c.entries, k)
				c.stats.Items--
				c.stats.SizeBytes = uint64(c.sizeBytes)
			}
			c.stats.Misses++
			c.mu.Unlock()
			return nil, false
		}
		c.mu.Lock()
		c.stats.Hits++
		c.mu.Unlock()
		return dec, true
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()
	return data, true
}

func (c *cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		Hits:      c.stats.Hits,
		Misses:    c.stats.Misses,
		Evictions: c.stats.Evictions,
		SizeBytes: uint64(c.sizeBytes),
		Items:     c.stats.Items,
	}
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
