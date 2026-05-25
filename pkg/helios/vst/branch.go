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

// Branch registry — minimal named-head primitive required by Fork.MergeInto's
// compare-and-swap. Intentionally narrow: this is not a full reference-tracking
// store, only the mutable pointer that VFSFork advances atomically.

package vst

import (
	"fmt"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// CreateBranch registers a new named branch pointing at head.
// Use the empty SnapshotID to create an orphan branch (Fork from "" allowed
// only if an empty-tree snapshot has been committed; otherwise callers must
// commit first and then CreateBranch with the resulting id).
func (v *VST) CreateBranch(name BranchID, head types.SnapshotID) error {
	if name == "" {
		return fmt.Errorf("vst.CreateBranch: empty branch name")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.branches[name]; ok {
		return ErrBranchExists
	}
	if head != "" {
		if _, ok := v.snaps[head]; !ok {
			return fmt.Errorf("vst.CreateBranch: %w (head=%s)", ErrUnknownSnapshot, head)
		}
	}
	v.branches[name] = head
	return nil
}

// BranchHead returns the current head SnapshotID for the named branch, with
// ok=false if the branch is not registered.
func (v *VST) BranchHead(name BranchID) (types.SnapshotID, bool) {
	v.mu.RLock()
	id, ok := v.branches[name]
	v.mu.RUnlock()
	return id, ok
}

// Branches returns a snapshot of all branch-name → head mappings.
// The returned map is a defensive copy; mutating it does not affect VST state.
func (v *VST) Branches() map[BranchID]types.SnapshotID {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make(map[BranchID]types.SnapshotID, len(v.branches))
	for k, v := range v.branches {
		out[k] = v
	}
	return out
}

// DeleteBranch removes a registered branch. Returns ErrUnknownBranch if
// the name is not registered.
func (v *VST) DeleteBranch(name BranchID) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.branches[name]; !ok {
		return ErrUnknownBranch
	}
	delete(v.branches, name)
	return nil
}
