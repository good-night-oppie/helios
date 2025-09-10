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
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/good-night-oppie/helios/pkg/helios/types"
	lru "github.com/hashicorp/golang-lru/v2"
	"lukechampine.com/blake3"
)

// writeOp represents a background write operation
type writeOp struct {
	filePath string
	content  []byte
}

// BLAKE3StoreConfig contains configuration options for BLAKE3Store
type BLAKE3StoreConfig struct {
	// Cache configuration
	CacheSize int // Maximum number of items in LRU cache (default: 10000)
	
	// Queue configuration
	WriteQueueSize int // Size of async write queue (default: 1000)
	ErrorQueueSize int // Size of error queue (default: 100)
	
	// Logger configuration
	Logger *slog.Logger // Optional structured logger (nil uses default stderr logging)
	
	// Performance configuration
	HexBufferSize int // Pre-allocated buffer size for hex encoding (default: 64)
}

// BLAKE3StoreOption is a functional option for configuring BLAKE3Store
type BLAKE3StoreOption func(*BLAKE3StoreConfig)

// WithLogger sets a custom structured logger
func WithLogger(logger *slog.Logger) BLAKE3StoreOption {
	return func(cfg *BLAKE3StoreConfig) {
		cfg.Logger = logger
	}
}

// WithCacheSize sets the LRU cache size
func WithCacheSize(size int) BLAKE3StoreOption {
	return func(cfg *BLAKE3StoreConfig) {
		cfg.CacheSize = size
	}
}

// WithQueueSizes sets the write and error queue sizes
func WithQueueSizes(writeQueue, errorQueue int) BLAKE3StoreOption {
	return func(cfg *BLAKE3StoreConfig) {
		cfg.WriteQueueSize = writeQueue
		cfg.ErrorQueueSize = errorQueue
	}
}

// BatchError accumulates multiple errors from batch operations
type BatchError struct {
	Errors []error
	Count  int
}

// Error implements the error interface
func (be *BatchError) Error() string {
	if be.Count == 0 {
		return "no errors"
	}
	if be.Count == 1 {
		return be.Errors[0].Error()
	}
	return fmt.Sprintf("batch operation failed with %d errors: first error: %v", be.Count, be.Errors[0])
}

// Add adds an error to the batch
func (be *BatchError) Add(err error) {
	if err != nil {
		be.Errors = append(be.Errors, err)
		be.Count++
	}
}

// HasErrors returns true if there are any errors
func (be *BatchError) HasErrors() bool {
	return be.Count > 0
}

// AsError returns the BatchError as an error if there are errors, nil otherwise
func (be *BatchError) AsError() error {
	if be.Count > 0 {
		return be
	}
	return nil
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
	storePath   string
	cache       *lru.Cache[string, []byte] // L1 LRU cache with bounded memory usage
	mutex       sync.RWMutex             // Protects cache for concurrent access
	
	// Performance optimizations from research phase
	hasherPool     sync.Pool         // Pool of reusable BLAKE3 hashers
	hexBufferPool  sync.Pool         // Pool of reusable hex encoding buffers
	keyCache       map[string]string // Pre-computed hex keys for hot paths
	keyMutex       sync.RWMutex      // Protects key cache
	
	// Production-grade logging and monitoring
	logger         *slog.Logger      // Structured logger for production observability
	
	// Ultra-performance optimizations for <70μs VST targets
	memoryMode  bool              // Skip disk I/O for maximum performance
	writeQueue  chan writeOp      // Async write queue for background persistence
	errorQueue  chan error        // Channel for background write errors
	wg          sync.WaitGroup    // Wait group for background writes
	closed      int32             // Atomic flag to track if store is closed (0=open, 1=closed)
	done        chan struct{}     // Signal channel for graceful shutdown coordination
	shutdownMu  sync.RWMutex      // Protects against shutdown vs work addition race
}

// defaultConfig returns the default configuration for BLAKE3Store
func defaultConfig() *BLAKE3StoreConfig {
	return &BLAKE3StoreConfig{
		CacheSize:      10000, // 10K items default LRU cache size
		WriteQueueSize: 1000,  // 1K async write queue
		ErrorQueueSize: 100,   // 100 error queue
		Logger:         nil,   // Use default stderr logging
		HexBufferSize:  64,    // 64 bytes for hex encoding buffer
	}
}

// NewBLAKE3Store creates a new BLAKE3Store with optional configuration
func NewBLAKE3Store(storePath string, opts ...BLAKE3StoreOption) (*BLAKE3Store, error) {
	// Apply configuration options
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	
	// Ensure storage directory exists
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	// Create LRU cache with configured size
	cache, err := lru.New[string, []byte](cfg.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	
	// Set up logger - use provided logger or create default
	logger := cfg.Logger
	if logger == nil {
		// Create default structured logger that writes to stderr
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn, // Only warn and above by default
		}))
	}
	
	store := &BLAKE3Store{
		storePath:  storePath,
		cache:      cache,
		keyCache:   make(map[string]string),
		logger:     logger,
		memoryMode: false,                           // Default to persistent mode
		writeQueue: make(chan writeOp, cfg.WriteQueueSize),
		errorQueue: make(chan error, cfg.ErrorQueueSize),
		closed:     0,                // Store is initially open
		done:       make(chan struct{}), // Done channel for shutdown coordination
	}
	
	// Initialize hasher pool for zero-allocation hashing
	store.hasherPool = sync.Pool{
		New: func() interface{} {
			return blake3.New(32, nil) // Pre-configured 256-bit unkeyed hasher
		},
	}
	
	// Initialize hex buffer pool for race-free hex encoding
	store.hexBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, cfg.HexBufferSize) // Pre-allocated hex buffer
		},
	}
	
	// Start background writer goroutine
	go store.backgroundWriter()
	
	// Start error handler goroutine
	go store.errorHandler()
	
	return store, nil
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
		// Use structured logging for production-grade error reporting
		s.logger.Error("BLAKE3Store background write failed",
			"error", err,
			"component", "blake3_store",
			"operation", "background_write",
			"store_path", s.storePath,
			"timestamp", time.Now().UTC(),
		)
		
		// TODO: Future enhancements for production:
		// 1. Implement retry logic with exponential backoff
		// 2. Send metrics/alerts for monitoring integration
		// 3. Consider circuit breaker pattern for persistent failures
		// 4. Add structured context fields for better observability
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

	// Pre-compute hash key using optimized hex encoding
	hashKey := s.hexEncode(digest)
	
	// Check if already exists to avoid duplicate work
	s.mutex.RLock()
	if _, exists := s.cache.Get(hashKey); exists {
		s.mutex.RUnlock()
		return hash, nil // Content already stored
	}
	s.mutex.RUnlock()

	// Store in LRU cache for fast access with bounded memory usage
	s.mutex.Lock()
	cachedContent := make([]byte, len(content))
	copy(cachedContent, content)
	s.cache.Add(hashKey, cachedContent) // LRU will evict oldest if at capacity
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
		
		// Use async writes for better performance with graceful shutdown protection
		select {
		case <-s.done:
			// Store is shutting down, decrement WaitGroup and fallback to sync write
			s.wg.Done()
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return hash, fmt.Errorf("failed to write content to disk during shutdown: %w", err)
			}
		case s.writeQueue <- writeOp{filePath: filePath, content: content}:
			// Successfully queued for background write
		default:
			// Queue full, must call Done() since we already called Add()
			s.wg.Done()
			// Write synchronously as fallback
			if err := os.WriteFile(filePath, content, 0644); err != nil {
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
	
	// Initialize batch error tracking
	batchErrors := &BatchError{}
	
	// Batch cache operations (single lock acquisition)
	s.mutex.Lock()
	for i, content := range contents {
		hashKey := hashKeys[i]
		if _, exists := s.cache.Get(hashKey); !exists {
			cachedContent := make([]byte, len(content))
			copy(cachedContent, content)
			s.cache.Add(hashKey, cachedContent) // LRU handles eviction
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
				// Store is closed, write synchronously with error tracking
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					batchErrors.Add(fmt.Errorf("failed to write %s during shutdown: %w", hashKeys[i], err))
				}
				continue
			}
			
			// Critical: Add to WaitGroup BEFORE attempting to send to channel
			// We hold shutdownMu.RLock so Close() cannot proceed until we're done
			s.wg.Add(1)
			
			select {
			case <-s.done:
				// Store is shutting down, decrement WaitGroup and fallback to sync write
				s.wg.Done()
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					batchErrors.Add(fmt.Errorf("failed to write %s during shutdown: %w", hashKeys[i], err))
				}
			case s.writeQueue <- writeOp{filePath: filePath, content: content}:
				// Successfully queued
			default:
				// Queue full, must call Done() since we already called Add()
				s.wg.Done()
				// Fallback to sync write with error tracking
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					batchErrors.Add(fmt.Errorf("failed to write %s (queue full): %w", hashKeys[i], err))
				}
			}
		}
	}
	
	// Return accumulated errors if any occurred
	if batchError := batchErrors.AsError(); batchError != nil {
		s.logger.Warn("StoreBatch completed with errors",
			"error_count", batchErrors.Count,
			"total_items", len(contents),
			"component", "blake3_store",
			"operation", "store_batch",
		)
		return hashes, batchError
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

	// Try LRU cache first (L1 cache hit)
	s.mutex.RLock()
	if content, exists := s.cache.Get(hashKey); exists {
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

	// Cache for future access in LRU cache
	s.mutex.Lock()
	cachedContent := make([]byte, len(content))
	copy(cachedContent, content)
	s.cache.Add(hashKey, cachedContent) // LRU handles eviction automatically
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

	// Check LRU cache first (fastest path) - should be <50μs
	s.mutex.RLock()
	_, exists := s.cache.Get(hashKey)
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
	s.cache.Purge() // Clear LRU cache
	s.mutex.Unlock()
	
	s.keyMutex.Lock()
	s.keyCache = make(map[string]string)
	s.keyMutex.Unlock()
	
	// Log successful shutdown
	s.logger.Info("BLAKE3Store shutdown completed",
		"component", "blake3_store",
		"operation", "shutdown",
		"store_path", s.storePath,
	)
	
	return nil
}

// hexEncode optimizes hex encoding using pooled buffer for race-free performance
func (s *BLAKE3Store) hexEncode(data []byte) string {
	// Get buffer from pool for race-free encoding
	hexBuffer := s.hexBufferPool.Get().([]byte)
	defer s.hexBufferPool.Put(hexBuffer)
	
	// For small digests, use pooled buffer for better performance
	if len(data)*2 <= len(hexBuffer) {
		n := hex.Encode(hexBuffer, data)
		return string(hexBuffer[:n])
	}
	// Fall back to standard encoding for larger data
	return hex.EncodeToString(data)
}