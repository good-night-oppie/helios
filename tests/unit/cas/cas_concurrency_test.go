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

package cas_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/good-night-oppie/helios/pkg/helios/cas"
	"github.com/good-night-oppie/helios/pkg/helios/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBLAKE3Store_Concurrency tests concurrent operations
func TestBLAKE3Store_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	store, err := cas.NewBLAKE3Store(tempDir)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Concurrent_Store_Operations", func(t *testing.T) {
		concurrency := 10
		contentCount := 100
		
		// Generate unique content for each goroutine
		allContent := make([][]byte, concurrency*contentCount)
		for i := range allContent {
			allContent[i] = []byte(fmt.Sprintf("concurrent_content_%d", i))
		}
		
		// Channel to collect results
		results := make(chan types.Hash, len(allContent))
		errors := make(chan error, len(allContent))
		
		// Launch concurrent store operations
		for i := 0; i < concurrency; i++ {
			go func(startIdx int) {
				for j := 0; j < contentCount; j++ {
					idx := startIdx*contentCount + j
					hash, err := store.Store(allContent[idx])
					if err != nil {
						errors <- err
						return
					}
					results <- hash
				}
			}(i)
		}
		
		// Collect results - map hash to original content for verification
		hashToContent := make(map[string][]byte)
		var hashes []types.Hash
		
		for i := 0; i < len(allContent); i++ {
			select {
			case hash := <-results:
				hashes = append(hashes, hash)
			case err := <-errors:
				t.Fatalf("Concurrent store operation failed: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}
		
		assert.Len(t, hashes, len(allContent))
		
		// Create expected content map by storing each content piece
		for _, content := range allContent {
			tempHash, err := store.Store(content)
			require.NoError(t, err)
			hashKey := fmt.Sprintf("%x", tempHash.Digest)
			hashToContent[hashKey] = content
		}
		
		// Verify all stored content can be retrieved correctly
		for _, hash := range hashes {
			retrieved, err := store.Load(hash)
			require.NoError(t, err)
			
			hashKey := fmt.Sprintf("%x", hash.Digest)
			expectedContent, exists := hashToContent[hashKey]
			require.True(t, exists, "Hash should exist in expected content map")
			assert.Equal(t, expectedContent, retrieved)
		}
	})
}

// TestRaceConditionAndShutdown tests the critical race conditions identified in PR #14 comments
func TestRaceConditionAndShutdown(t *testing.T) {
	t.Run("Concurrent_Store_Operations_During_Shutdown", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		
		const numGoroutines = 100
		const opsPerGoroutine = 10
		
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		// Start concurrent store operations
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < opsPerGoroutine; j++ {
					content := []byte(fmt.Sprintf("content-%d-%d", id, j))
					_, err := store.Store(content)
					// Either succeeds or fails with "store is closed" - both are acceptable
					if err != nil {
						assert.Contains(t, err.Error(), "store is closed")
					}
				}
			}(i)
		}
		
		// Allow some operations to start
		time.Sleep(10 * time.Millisecond)
		
		// Close the store while operations are running
		err = store.Close()
		require.NoError(t, err)
		
		// Wait for all goroutines to complete
		wg.Wait()
		
		// Verify store is properly closed
		_, err = store.Store([]byte("should fail"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "store is closed")
	})
	
	t.Run("No_Send_On_Closed_Channel_Panic", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		
		// Fill the write queue to trigger fallback paths
		for i := 0; i < 1100; i++ { // More than buffer size (1000)
			go func(id int) {
				content := []byte(fmt.Sprintf("content-%d", id))
				store.Store(content) // Should not panic even during shutdown
			}(i)
		}
		
		// Close immediately to test shutdown race conditions
		err = store.Close()
		require.NoError(t, err)
		
		// Additional operations after close should fail gracefully, not panic
		_, err = store.Store([]byte("after close"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "store is closed")
	})
	
	t.Run("Double_Close_Protection", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		
		// First close should succeed
		err = store.Close()
		require.NoError(t, err)
		
		// Second close should not panic or error
		err = store.Close()
		require.NoError(t, err)
		
		// Third close should also be safe
		err = store.Close()
		require.NoError(t, err)
	})
	
	t.Run("Graceful_Shutdown_With_Background_Writes", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		
		// Start background operations that will queue writes
		var hashes []types.Hash
		for i := 0; i < 10; i++ {
			content := []byte(fmt.Sprintf("background-content-%d", i))
			hash, err := store.Store(content)
			require.NoError(t, err)
			hashes = append(hashes, hash)
		}
		
		// Close should wait for background writes to complete
		start := time.Now()
		err = store.Close()
		require.NoError(t, err)
		closeTime := time.Since(start)
		
		// Should not take too long (background writes should complete quickly)
		assert.Less(t, closeTime, 100*time.Millisecond, "Close took too long: %v", closeTime)
		
		// All files should be properly written
		for i, hash := range hashes {
			hashKey := fmt.Sprintf("%x", hash.Digest)
			filePath := filepath.Join(tempDir, hashKey)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("background-content-%d", i))
			assert.Equal(t, expected, content)
		}
	})
	
	t.Run("Atomic_Close_Flag_Race_Detection", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		
		const numReaders = 50
		const numWrites = 10
		
		var wg sync.WaitGroup
		
		// Start many goroutines checking if store is closed
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numWrites; j++ {
					content := []byte(fmt.Sprintf("race-test-content-%d", j))
					_, err := store.Store(content)
					// Should either succeed or fail cleanly with "store is closed"
					if err != nil {
						assert.Contains(t, err.Error(), "store is closed")
					}
					time.Sleep(time.Microsecond) // Small delay to increase chance of race
				}
			}()
		}
		
		// Close after a short delay
		time.Sleep(5 * time.Millisecond)
		err = store.Close()
		require.NoError(t, err)
		
		wg.Wait()
	})
}