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
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBLAKE3Store tests basic store initialization
func TestNewBLAKE3Store(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewBLAKE3Store(tempDir)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	// Test basic functionality
	content := []byte("test content")
	hash, err := store.Store(content)
	require.NoError(t, err)
	assert.Equal(t, types.BLAKE3, hash.Algorithm)

	retrieved, err := store.Load(hash)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)

	assert.True(t, store.Exists(hash))
}

// TestBLAKE3Store_Close tests store closure
func TestBLAKE3Store_Close(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewBLAKE3Store(tempDir)
	require.NoError(t, err)

	// Store should work before close
	content := []byte("test content")
	_, err = store.Store(content)
	require.NoError(t, err)

	// Close should work
	err = store.Close()
	require.NoError(t, err)

	// Operations after close should fail gracefully
	_, err = store.Store([]byte("after close"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store is closed")
}