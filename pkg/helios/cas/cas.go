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

// SPDX-License-Identifier: Apache-2.0

package cas

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/good-night-oppie/helios/pkg/helios/types"
	"lukechampine.com/blake3"
)

// writeOp represents a background write operation
type writeOp struct {
	filePath string
	content  []byte
}

// ContentAddressableStore defines the interface for content-addressable storage
// implementing the core CAS operations with performance guarantees from research
type ContentAddressableStore interface {
	// Store saves content and returns its content-addressable hash
	// Performance target: <1ms for typical code file sizes
	Store(content []byte) (types.Hash, error)

	// Load retrieves content by its hash
	// Performance target: <5ms for typical code file sizes
	Load(hash types.Hash) ([]byte, error)

	// Exists checks if content with given hash exists
	// Performance target: <50μs for existence checks (relaxed during implementation)
	Exists(hash types.Hash) bool

	// Close releases resources
	Close() error
}

// BLAKE3Store implements ContentAddressableStore using BLAKE3 hashing
// Based on research findings showing BLAKE3's superior performance characteristics:
// - ~15 GB/s throughput with AVX2
// - <100ns latency for small inputs
// - SIMD acceleration support
type BLAKE3Store struct {
	storePath string
	cache     map[string][]byte // L1 cache for hot content
	mutex     sync.RWMutex      // Protects cache for concurrent access

	// Performance optimizations from research phase
	hasherPool sync.Pool         // Pool of reusable BLAKE3 hashers
	keyCache   map[string]string // Pre-computed hex keys for hot paths
	keyMutex   sync.RWMutex      // Protects key cache

	// Ultra-performance optimizations for <70μs VST targets
	memoryMode    bool           // Skip disk I/O for maximum performance
	writeQueue    chan writeOp   // Async write queue for background persistence
	errorQueue    chan error     // Channel for background write errors
	wg            sync.WaitGroup // Wait group for background writes
	closed        int32          // Atomic flag to track if store is closed (0=open, 1=closed)
	done          chan struct{}  // Signal channel for graceful shutdown coordination
	shutdownMu    sync.RWMutex   // Protects against shutdown vs work addition race
	hexBufferPool sync.Pool      // Thread-safe hex encoding buffer pool
}

// NewBLAKE3Store creates a new BLAKE3-based content-addressable store
// storePath: directory for persistent storage
func NewBLAKE3Store(storePath string) (*BLAKE3Store, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	store := &BLAKE3Store{
		storePath:  storePath,
		cache:      make(map[string][]byte),
		keyCache:   make(map[string]string),
		memoryMode: false,                    // Default to persistent mode
		writeQueue: make(chan writeOp, 1000), // Buffered channel for async writes
		errorQueue: make(chan error, 100),    // Buffered channel for errors
		closed:     0,                        // Store is initially open
		done:       make(chan struct{}),      // Done channel for shutdown coordination
	}

	// Initialize hasher pool for zero-allocation hashing
	store.hasherPool = sync.Pool{
		New: func() interface{} {
			return blake3.New(32, nil) // Pre-configured 256-bit unkeyed hasher
		},
	}

	// Initialize hex buffer pool for thread-safe hex encoding
	store.hexBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64) // 32-byte hash * 2 for hex encoding
		},
	}

	// Start background writer goroutine
	go store.backgroundWriter()

	// Start error handler goroutine
	go store.errorHandler()

	return store, nil
}

// hexEncode provides thread-safe hex encoding using pooled buffers
func (s *BLAKE3Store) hexEncode(data []byte) string {
	buf := s.hexBufferPool.Get().([]byte)
	defer s.hexBufferPool.Put(buf)
	n := hex.Encode(buf, data)
	return string(buf[:n])
}

// backgroundWriter handles async disk writes for maximum performance
func (s *BLAKE3Store) backgroundWriter() {
	for writeOp := range s.writeQueue {
		if err := os.WriteFile(writeOp.filePath, writeOp.content, 0644); err != nil {
			// Send error to error handler for logging/retry
			select {
			case s.errorQueue <- fmt.Errorf("background write failed for %s: %w", writeOp.filePath, err):
			default:
				// Error queue full, error will be dropped but write continues
			}
		}
		s.wg.Done()
	}
}

// errorHandler processes background write errors
func (s *BLAKE3Store) errorHandler() {
	for err := range s.errorQueue {
		// Log error to stderr for proper error handling
		// Using fmt.Fprintf to stderr instead of stdout for errors
		fmt.Fprintf(os.Stderr, "[ERROR] BLAKE3Store background write failed: %v\n", err)

		// TODO: In production, this should:
		// 1. Log to proper logging system (slog, logrus, zap)
		// 2. Implement retry logic with exponential backoff
		// 3. Send metrics/alerts for monitoring
		// 4. Consider circuit breaker pattern for persistent failures
	}
}

// EnableMemoryMode switches to memory-only mode for maximum performance (<70μs targets)
func (s *BLAKE3Store) EnableMemoryMode() {
	s.memoryMode = true
}

// Store saves content and returns its BLAKE3 hash
func (s *BLAKE3Store) Store(content []byte) (types.Hash, error) {
	// Quick atomic check if store is closed (race-free performance check)
	if atomic.LoadInt32(&s.closed) != 0 {
		return types.Hash{}, fmt.Errorf("store is closed")
	}
	// Get hasher from pool for zero allocation
	hasher := s.hasherPool.Get().(*blake3.Hasher)
	defer func() {
		hasher.Reset() // Reset for reuse
		s.hasherPool.Put(hasher)
	}()

	// Compute BLAKE3 hash using pooled hasher
	hasher.Write(content)
	digest := hasher.Sum(nil)

	hash := types.Hash{
		Algorithm: types.BLAKE3,
		Digest:    digest,
	}

	// Pre-compute hash key once using thread-safe hex encoding
	hashKey := s.hexEncode(digest)

	// Check if already exists to avoid duplicate work
	s.mutex.RLock()
	if _, exists := s.cache[hashKey]; exists {
		s.mutex.RUnlock()
		return hash, nil // Content already stored
	}
	s.mutex.RUnlock()

	// Store in cache for fast access (zero-copy when possible)
	s.mutex.Lock()
	s.cache[hashKey] = make([]byte, len(content))
	copy(s.cache[hashKey], content)
	s.mutex.Unlock()

	// Cache the hex key for future lookups
	s.keyMutex.Lock()
	s.keyCache[string(digest)] = hashKey
	s.keyMutex.Unlock()

	// Store persistently to disk - async for performance or skip in memory mode
	if !s.memoryMode {
		filePath := filepath.Join(s.storePath, hashKey)

		// Use read lock to prevent race between shutdown and work addition
		s.shutdownMu.RLock()
		defer s.shutdownMu.RUnlock()

		// Check if store is closed after acquiring lock
		if atomic.LoadInt32(&s.closed) != 0 {
			// Store is closed, write synchronously
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return hash, fmt.Errorf("failed to write content to disk (store closed): %w", err)
			}
			return hash, nil
		}

		// Critical: Add to WaitGroup BEFORE attempting to send to channel
		// to prevent race condition where Done() is called before Add()
		// We hold shutdownMu.RLock so Close() cannot proceed until we're done
		s.wg.Add(1)

		// Make defensive copy before queuing to prevent caller mutations
		dataCopy := make([]byte, len(content))
		copy(dataCopy, content)

		// Use async writes for better performance with graceful shutdown protection
		select {
		case <-s.done:
			// Store is shutting down, decrement WaitGroup and fallback to sync write
			s.wg.Done()
			if err := os.WriteFile(filePath, dataCopy, 0644); err != nil {
				return hash, fmt.Errorf("failed to write content to disk during shutdown: %w", err)
			}
		case s.writeQueue <- writeOp{filePath: filePath, content: dataCopy}:
			// Successfully queued for background write
		default:
			// Queue full, must call Done() since we already called Add()
			s.wg.Done()
			// Write synchronously as fallback
			if err := os.WriteFile(filePath, dataCopy, 0644); err != nil {
				return hash, fmt.Errorf("failed to write content to disk: %w", err)
			}
		}
	}

	return hash, nil
}

// StoreBatch processes multiple content items in a single optimized operation
// Designed for VST commit scenarios requiring <70μs for 50 files
func (s *BLAKE3Store) StoreBatch(contents [][]byte) ([]types.Hash, error) {
	hashes := make([]types.Hash, len(contents))

	// Pre-allocate maps to avoid repeated allocations
	hashKeys := make([]string, len(contents))

	// Process all hashes first (CPU-bound, can be optimized)
	for i, content := range contents {
		// Get hasher from pool for zero allocation
		hasher := s.hasherPool.Get().(*blake3.Hasher)
		hasher.Write(content)
		digest := hasher.Sum(nil)
		hasher.Reset()
		s.hasherPool.Put(hasher)

		hashes[i] = types.Hash{
			Algorithm: types.BLAKE3,
			Digest:    digest,
		}
		hashKeys[i] = s.hexEncode(digest)
	}

	// Batch cache operations (single lock acquisition)
	s.mutex.Lock()
	for i, content := range contents {
		hashKey := hashKeys[i]
		if _, exists := s.cache[hashKey]; !exists {
			s.cache[hashKey] = make([]byte, len(content))
			copy(s.cache[hashKey], content)
		}
	}
	s.mutex.Unlock()

	// Batch key cache operations
	s.keyMutex.Lock()
	for i, hash := range hashes {
		digestStr := string(hash.Digest)
		s.keyCache[digestStr] = hashKeys[i]
	}
	s.keyMutex.Unlock()

	// Handle persistence based on mode
	if !s.memoryMode {
		// Use read lock to prevent race between shutdown and work addition
		s.shutdownMu.RLock()
		defer s.shutdownMu.RUnlock()

		// Queue all writes together for better batching
		for i, content := range contents {
			filePath := filepath.Join(s.storePath, hashKeys[i])

			// Check if store is closed after acquiring lock
			if atomic.LoadInt32(&s.closed) != 0 {
				// Store is closed, write synchronously
				dataCopy := make([]byte, len(content))
				copy(dataCopy, content)
				os.WriteFile(filePath, dataCopy, 0644) // Ignore error in batch mode
				continue
			}

			// Make defensive copy before queuing to prevent caller mutations
			dataCopy := make([]byte, len(content))
			copy(dataCopy, content)

			// Critical: Add to WaitGroup BEFORE attempting to send to channel
			// We hold shutdownMu.RLock so Close() cannot proceed until we're done
			s.wg.Add(1)

			select {
			case <-s.done:
				// Store is shutting down, decrement WaitGroup and fallback to sync write
				s.wg.Done()
				os.WriteFile(filePath, dataCopy, 0644) // Ignore error in batch mode during shutdown
			case s.writeQueue <- writeOp{filePath: filePath, content: dataCopy}:
				// Successfully queued
			default:
				// Queue full, must call Done() since we already called Add()
				s.wg.Done()
				// Fallback to sync write
				os.WriteFile(filePath, dataCopy, 0644) // Ignore error in batch mode
			}
		}
	}

	return hashes, nil
}

// Load retrieves content by its hash
func (s *BLAKE3Store) Load(hash types.Hash) ([]byte, error) {
	// Validate hash algorithm
	if hash.Algorithm != types.BLAKE3 {
		return nil, fmt.Errorf("unsupported hash algorithm: %s", hash.Algorithm)
	}

	// Fast path: check key cache first
	digestStr := string(hash.Digest)
	s.keyMutex.RLock()
	hashKey, keyExists := s.keyCache[digestStr]
	s.keyMutex.RUnlock()

	if !keyExists {
		hashKey = s.hexEncode(hash.Digest)
		// Cache the key for future lookups
		s.keyMutex.Lock()
		s.keyCache[digestStr] = hashKey
		s.keyMutex.Unlock()
	}

	// Try cache first (L1 cache hit)
	s.mutex.RLock()
	if content, exists := s.cache[hashKey]; exists {
		// Return copy to prevent external modification
		result := make([]byte, len(content))
		copy(result, content)
		s.mutex.RUnlock()
		return result, nil
	}
	s.mutex.RUnlock()

	// Load from disk (L2 storage)
	filePath := filepath.Join(s.storePath, hashKey)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("content not found for hash %s", hashKey)
		}
		return nil, fmt.Errorf("failed to read content from disk: %w", err)
	}

	// Cache for future access
	s.mutex.Lock()
	s.cache[hashKey] = make([]byte, len(content))
	copy(s.cache[hashKey], content)
	s.mutex.Unlock()

	return content, nil
}

// Exists checks if content with given hash exists
func (s *BLAKE3Store) Exists(hash types.Hash) bool {
	if hash.Algorithm != types.BLAKE3 {
		return false
	}

	// Fast path: check key cache first for microsecond performance
	digestStr := string(hash.Digest)
	s.keyMutex.RLock()
	hashKey, keyExists := s.keyCache[digestStr]
	s.keyMutex.RUnlock()

	if !keyExists {
		hashKey = s.hexEncode(hash.Digest)
		// Cache for future lookups
		s.keyMutex.Lock()
		s.keyCache[digestStr] = hashKey
		s.keyMutex.Unlock()
	}

	// Check cache first (fastest path) - should be <50μs
	s.mutex.RLock()
	_, exists := s.cache[hashKey]
	s.mutex.RUnlock()
	if exists {
		return true
	}

	// Check disk storage
	filePath := filepath.Join(s.storePath, hashKey)
	_, err := os.Stat(filePath)
	return err == nil
}

// Close releases resources and clears cache using graceful shutdown pattern
func (s *BLAKE3Store) Close() error {
	// Use atomic compare-and-swap to prevent double-close race conditions
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil // Already closed
	}

	// Step 1: Signal shutdown to prevent new operations from starting
	// This must be done BEFORE waiting for existing operations
	close(s.done)

	// Step 2: Acquire write lock to prevent new work from being added to WaitGroup
	// All Store operations must complete their WaitGroup.Add before we can proceed
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	// Step 3: Wait for any in-flight operations to complete
	// Background goroutines will see done channel closed and exit cleanly
	s.wg.Wait()

	// Step 3: Now safe to close channels - no more sends will occur
	close(s.writeQueue)
	close(s.errorQueue)

	// Step 4: Clear caches to release memory
	s.mutex.Lock()
	s.cache = make(map[string][]byte)
	s.mutex.Unlock()

	s.keyMutex.Lock()
	s.keyCache = make(map[string]string)
	s.keyMutex.Unlock()

	return nil
}
