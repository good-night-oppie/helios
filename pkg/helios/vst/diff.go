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
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// Diff compares two snapshots and returns Added/Changed/Deleted counts.
func (v *VST) Diff(from, to types.SnapshotID) (types.DiffStats, error) {
	fromSnap, err := v.snapshotHashes(from)
	if err != nil {
		return types.DiffStats{}, err
	}
	toSnap, err := v.snapshotHashes(to)
	if err != nil {
		return types.DiffStats{}, err
	}

	var stats types.DiffStats
	for path, fromHash := range fromSnap {
		if toHash, exists := toSnap[path]; !exists {
			stats.Deleted++
		} else if !hashEqual(fromHash, toHash) {
			stats.Changed++
		}
	}
	for path := range toSnap {
		if _, exists := fromSnap[path]; !exists {
			stats.Added++
		}
	}
	return stats, nil
}

func (v *VST) snapshotHashes(id types.SnapshotID) (map[string]types.Hash, error) {
	if snap, ok := v.snaps[id]; ok {
		res := make(map[string]types.Hash, len(snap))
		for p, c := range snap {
			h, err := util.HashBlob(c)
			if err != nil {
				return nil, err
			}
			res[p] = h
		}
		return res, nil
	}
	if v.l2 != nil {
		key := "snapshot:" + string(id)
		metaHash := types.Hash{Algorithm: types.BLAKE3, Digest: []byte(key)}
		data, ok, err := v.l2.Get(metaHash)
		if err != nil {
			return nil, fmt.Errorf("get snapshot %s from L2: %w", id, err)
		}
		if !ok {
			return nil, fmt.Errorf("snapshot %s not found in memory or L2 storage", id)
		}
		var m map[string]types.Hash
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
		}
		return m, nil
	}
	return nil, fmt.Errorf("snapshot %s not found in memory or L2 storage", id)
}

func hashEqual(a, b types.Hash) bool {
	return a.Algorithm == b.Algorithm && bytes.Equal(a.Digest, b.Digest)
}
