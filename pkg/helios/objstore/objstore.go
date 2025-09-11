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


package objstore

import (
	"errors"

	"github.com/cockroachdb/pebble"
	"github.com/good-night-oppie/helios/pkg/helios/types"
)

type Options struct {
	ReadOnly bool
}

// BatchEntry represents a single key-value pair for batch operations
type BatchEntry struct {
	Hash  types.Hash
	Value []byte
}

type Store interface {
	PutBatch(batch []BatchEntry) error
	Get(h types.Hash) (value []byte, ok bool, err error)
	Close() error
}

type pebbleStore struct {
	db *pebble.DB
}

func Open(path string, opts *Options) (Store, error) {
	pebbleOpts := &pebble.Options{
		// Optimize for write-heavy workload (AI agents commit frequently)
		MemTableSize:             64 << 20, // 64MB
		MemTableStopWritesThreshold: 4,
		L0CompactionThreshold:    4,
		L0StopWritesThreshold:    12,
		LBaseMaxBytes:            64 << 20, // 64MB
		MaxConcurrentCompactions: func() int { return 3 },
		// Enable WAL for durability
		DisableWAL: false,
	}

	if opts != nil && opts.ReadOnly {
		pebbleOpts.ReadOnly = true
	}

	db, err := pebble.Open(path, pebbleOpts)
	if err != nil {
		return nil, err
	}

	return &pebbleStore{db: db}, nil
}

func (s *pebbleStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// PutBatch writes all entries atomically. Preflight rejects any nil value.
func (s *pebbleStore) PutBatch(batch []BatchEntry) error {
	// First validate all values
	for _, entry := range batch {
		if entry.Value == nil {
			return errors.New("nil value in batch")
		}
	}

	// Use Pebble batch for atomic writes
	b := s.db.NewBatch()
	defer b.Close()

	// Iterate through batch and write
	for _, entry := range batch {
		// Use only the digest part as key, since String() includes the algorithm prefix
		k := entry.Hash.Digest
		if err := b.Set(k, entry.Value, pebble.Sync); err != nil {
			return err
		}
	}

	return b.Commit(pebble.Sync)
}

// Get returns (value, ok, err). ok=false when key is missing, err on PebbleDB error.
func (s *pebbleStore) Get(h types.Hash) ([]byte, bool, error) {
	// Use only the digest part as key, since String() includes the algorithm prefix
	k := h.Digest
	val, closer, err := s.db.Get(k)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()
	
	// Copy the data since it's only valid until closer.Close()
	data := make([]byte, len(val))
	copy(data, val)
	return data, true, nil
}
