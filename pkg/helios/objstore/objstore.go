package objstore

import (
	"errors"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/tecbot/gorocksdb"
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

type rocksStore struct {
	db *gorocksdb.DB
	ro *gorocksdb.ReadOptions
	wo *gorocksdb.WriteOptions
}

func Open(path string, _ *Options) (Store, error) {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(opts, path)
	if err != nil {
		return nil, err
	}
	return &rocksStore{
		db: db,
		ro: gorocksdb.NewDefaultReadOptions(),
		wo: gorocksdb.NewDefaultWriteOptions(),
	}, nil
}

func (s *rocksStore) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	if s.ro != nil {
		s.ro.Destroy()
	}
	if s.wo != nil {
		s.wo.Destroy()
	}
	return nil
}

// PutBatch writes all entries atomically. Preflight rejects any nil value.
func (s *rocksStore) PutBatch(batch []BatchEntry) error {
	// First validate all values
	for _, entry := range batch {
		if entry.Value == nil {
			return errors.New("nil value in batch")
		}
	}

	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()

	// Iterate through batch and write
	for _, entry := range batch {
		k := []byte(entry.Hash.String())
		wb.Put(k, entry.Value)
	}

	return s.db.Write(s.wo, wb)
}

// Get returns (value, ok, err). ok=false when key is missing, err on RocksDB error.
func (s *rocksStore) Get(h types.Hash) ([]byte, bool, error) {
	k := []byte(h.String())
	val, err := s.db.Get(s.ro, k)
	if err != nil {
		return nil, false, err
	}
	defer val.Free()
	if !val.Exists() {
		return nil, false, nil
	}
	data := make([]byte, len(val.Data()))
	copy(data, val.Data())
	return data, true, nil
}
