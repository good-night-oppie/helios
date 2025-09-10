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
	"fmt"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// Diff compares two snapshots and returns Added/Changed/Deleted counts.
func (v *VST) Diff(from, to types.SnapshotID) (types.DiffStats, error) {
	fromSnap, ok := v.snaps[from]
	if !ok {
		return types.DiffStats{}, fmt.Errorf("unknown snapshot: %s", from)
	}

	toSnap, ok := v.snaps[to]
	if !ok {
		return types.DiffStats{}, fmt.Errorf("unknown snapshot: %s", to)
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
