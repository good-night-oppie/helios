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
	"testing"
	"time"

	"github.com/good-night-oppie/helios-engine/pkg/helios/cas"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBLAKE3Store_Integration tests integration with existing Helios components
func TestBLAKE3Store_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store, err := cas.NewBLAKE3Store(tempDir)
	require.NoError(t, err)
	defer store.Close()

	t.Run("Compatible_With_Existing_Hash_Type", func(t *testing.T) {
		content := []byte("integration test content")
		
		hash, err := store.Store(content)
		require.NoError(t, err)
		
		// Verify hash is compatible with existing types.Hash
		assert.Equal(t, types.BLAKE3, hash.Algorithm)
		assert.Len(t, hash.Digest, 32)
		
		// Should be convertible to string for logging/debugging
		hashStr := hash.String()
		assert.NotEmpty(t, hashStr)
		assert.Contains(t, hashStr, "blake3:")
	})

	t.Run("Persistence_Across_Restarts", func(t *testing.T) {
		content := []byte("persistence test")
		
		// Store in first instance
		hash, err := store.Store(content)
		require.NoError(t, err)
		store.Close()
		
		// Reopen store
		store2, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		defer store2.Close()
		
		// Content should still be retrievable
		retrieved, err := store2.Load(hash)
		require.NoError(t, err)
		assert.Equal(t, content, retrieved)
		
		// Exists should still return true
		assert.True(t, store2.Exists(hash))
	})
}

// TestVSTIntegration_Performance validates VST performance targets
func TestVSTIntegration_Performance(t *testing.T) {
	// This test validates the <70μs VST commit target from requirements
	t.Run("VST_Commit_Latency_Target", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		defer store.Close()
		
		// Enable memory mode for <70μs VST performance targets
		store.EnableMemoryMode()
		
		// Simulate VST commit operations (multiple small file hashes)
		fileCount := 50 // Typical number of files in a small commit
		files := make([][]byte, fileCount)
		
		// Generate typical code file content
		for i := range files {
			files[i] = []byte(fmt.Sprintf(`package main

import "fmt"

func function%d() {
	fmt.Println("Function %d implementation")
	// Some typical code content...
	for i := 0; i < 10; i++ {
		fmt.Printf("Iteration: %%d\n", i)
	}
}
`, i, i))
		}
		
		// Measure complete VST-like operation using batch optimization
		start := time.Now()
		
		// Use batch operation for maximum performance
		hashes, err := store.StoreBatch(files)
		require.NoError(t, err)
		
		vstCommitDuration := time.Since(start)
		
		// Performance target from requirements: <70μs VST commits (original target)
		// This is the total time for all file operations in a commit
		assert.Less(t, vstCommitDuration, 70*time.Microsecond,
			"VST commit simulation took %v, should be <70μs for %d files", 
			vstCommitDuration, fileCount)
		
		// Verify all content is retrievable
		for i, hash := range hashes {
			retrieved, err := store.Load(hash)
			require.NoError(t, err)
			assert.Equal(t, files[i], retrieved)
		}
	})

	t.Run("Batch_Operations_Performance", func(t *testing.T) {
		tempDir := t.TempDir()
		store, err := cas.NewBLAKE3Store(tempDir)
		require.NoError(t, err)
		defer store.Close()
		
		// Test storing multiple small contents (simulating code file commits)
		batchSize := 100
		contents := make([][]byte, batchSize)
		hashes := make([]types.Hash, batchSize)
		
		// Generate test data
		for i := 0; i < batchSize; i++ {
			contents[i] = []byte(fmt.Sprintf("file_content_%d", i))
		}
		
		// Batch store operations
		start := time.Now()
		for i, content := range contents {
			hash, err := store.Store(content)
			require.NoError(t, err)
			hashes[i] = hash
		}
		batchStoreDuration := time.Since(start)
		
		// Average per-operation should still be fast
		avgStoreTime := batchStoreDuration / time.Duration(batchSize)
		assert.Less(t, avgStoreTime, 1*time.Millisecond,
			"Average store time in batch: %v, should be <1ms", avgStoreTime)
		
		// Batch load operations
		start = time.Now()
		for i, hash := range hashes {
			retrieved, err := store.Load(hash)
			require.NoError(t, err)
			assert.Equal(t, contents[i], retrieved)
		}
		batchLoadDuration := time.Since(start)
		
		avgLoadTime := batchLoadDuration / time.Duration(batchSize)
		assert.Less(t, avgLoadTime, 5*time.Millisecond,
			"Average load time in batch: %v, should be <5ms", avgLoadTime)
	})
}