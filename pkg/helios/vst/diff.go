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


package vst

import (
	"encoding/json"
	"fmt"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// Diff compares two snapshots and returns Added/Changed/Deleted counts.
func (v *VST) Diff(from, to types.SnapshotID) (types.DiffStats, error) {
	// Try to get both snapshots (from memory or L2)
	fromSnap, err := v.getSnapshot(from)
	if err != nil {
		return types.DiffStats{}, fmt.Errorf("failed to get 'from' snapshot %s: %w", from, err)
	}

	toSnap, err := v.getSnapshot(to)
	if err != nil {
		return types.DiffStats{}, fmt.Errorf("failed to get 'to' snapshot %s: %w", to, err)
	}

	var stats types.DiffStats

	// Check for deleted and changed files
	for path, fromContent := range fromSnap {
		if toContent, exists := toSnap[path]; !exists {
			// File exists in 'from' but not in 'to' → Deleted
			stats.Deleted++
		} else if !bytesEqual(fromContent, toContent) {
			// File exists in both but content differs → Changed
			stats.Changed++
		}
	}

	// Check for added files
	for path := range toSnap {
		if _, exists := fromSnap[path]; !exists {
			// File exists in 'to' but not in 'from' → Added
			stats.Added++
		}
	}

	return stats, nil
}

// getSnapshot retrieves a snapshot from memory or L2 store
func (v *VST) getSnapshot(id types.SnapshotID) (map[string][]byte, error) {
	// First try to get from memory
	v.mu.RLock()
	snap, ok := v.snaps[id]
	v.mu.RUnlock()

	if ok {
		return snap, nil
	}

	// If not in memory and L2 is attached, try L2
	if v.l2 != nil {
		// Get snapshot metadata
		snapshotKey := string("snapshot:" + id)
		dprintf("getSnapshot: trying to get metadata with key %s", snapshotKey)
		metadataHash := types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)}
		metadataBytes, ok, err := v.l2.Get(metadataHash)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch snapshot metadata from L2: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("snapshot not found in L2: %s", id)
		}

		// Unmarshal metadata
		var snapshotData map[string]types.Hash
		if err := json.Unmarshal(metadataBytes, &snapshotData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
		}

		// Fetch file contents from L2
		snap := make(map[string][]byte)
		for path, hash := range snapshotData {
			data, ok, err := v.l2.Get(hash)
			if err != nil {
				return nil, fmt.Errorf("failed to get file %s from L2: %w", path, err)
			}
			if !ok {
				return nil, fmt.Errorf("missing file data in L2 for %s", path)
			}
			snap[path] = data
		}

		return snap, nil
	}

	return nil, fmt.Errorf("snapshot not found: %s", id)
}
