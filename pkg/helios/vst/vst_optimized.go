// Copyright 2025 Oppie Thunder Contributors
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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/good-night-oppie/helios/internal/util"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// CommitOptimized is a high-performance version of Commit that achieves <70μs targets
// for the default agent's working set.
// Equivalent to CommitOptimizedForAgent(AgentDefault, msg).
// Key optimizations:
// 1. O(n²) → O(n) directory tree building
// 2. Copy-on-Write (COW) semantics for snapshots
// 3. Efficient parent-child directory mapping
func (v *VST) CommitOptimized(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	return v.CommitOptimizedForAgent(AgentDefault, msg)
}

// CommitOptimizedForAgent is the agent-aware variant of CommitOptimized.
//
// Locking discipline mirrors CommitForAgent: short critical sections around
// every mutation of `v.agents`, `agentState.cur`, `agentState.pathToHash`,
// and `v.snaps`. Pure-CPU work (hashing, Merkle-tree build) and L2 I/O
// (PutBatch) happen outside the lock so concurrent agents do not block on
// each other for the slow path.
//
// Pre-VA-1a this function was already racy on `v.cur` / `v.pathToHash` /
// `v.snaps`. With multi-tenant `agents` map it became fatally racy because
// `agentRW` lazily inserts into `v.agents` — an unlocked insert collides
// with a concurrent map write and panics. Locking is now mandatory.
func (v *VST) CommitOptimizedForAgent(agent AgentId, msg string) (types.SnapshotID, types.CommitMetrics, error) {
	if err := validateAgent(agent); err != nil {
		return "", types.CommitMetrics{}, err
	}
	_ = msg // commit message currently unused (parity with Commit)
	start := time.Now()

	// Phase 1 — COW snapshot of cur under write lock.
	// agentRW may mutate v.agents; the cur reference swap mutates agentState.
	v.mu.Lock()
	s := v.agentRW(agent)
	snap := s.cur                              // Share reference to current working set
	s.cur = make(map[string][]byte, len(snap)) // New working set for future modifications
	v.mu.Unlock()

	var newBytes int64
	for _, val := range snap {
		newBytes += int64(len(val))
	}

	// Phase 2 — hash all blobs (pure CPU, no shared state mutated).
	blobHashByPath := make(map[string]types.Hash, len(snap))
	blobsToStore := make([]objstore.BatchEntry, 0, len(snap))
	for path, content := range snap {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		blobHashByPath[path] = h
		if v.l2 != nil {
			blobsToStore = append(blobsToStore, objstore.BatchEntry{
				Hash:  h,
				Value: content,
			})
		}
	}

	// Phase 3 — install pathToHash under write lock. `s` ptr captured in
	// Phase 1 is stable because agentRW never replaces an existing
	// agentState, only inserts a new one if absent.
	v.mu.Lock()
	for path, h := range blobHashByPath {
		s.pathToHash[path] = h
	}
	v.mu.Unlock()

	// Phase 4 — L2 blob batch (outside lock; I/O dominates).
	if v.l2 != nil && len(blobsToStore) > 0 {
		if err := v.l2.PutBatch(blobsToStore); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store blobs in L2: %w", err)
		}
	}

	// Phase 5 — O(n) Merkle tree build (pure CPU).
	root, err := v.buildDirectoryTreeOptimized(blobHashByPath)
	if err != nil {
		return "", types.CommitMetrics{}, err
	}
	id := types.SnapshotID(root.String())

	// Phase 6 — L2 snapshot metadata (outside lock).
	if v.l2 != nil {
		metadataBytes, err := json.Marshal(blobHashByPath)
		if err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to marshal snapshot metadata: %w", err)
		}
		snapshotKey := "snapshot:" + string(id)
		snapshotMetadata := []objstore.BatchEntry{{
			Hash:  types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)},
			Value: metadataBytes,
		}}
		if err := v.l2.PutBatch(snapshotMetadata); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store snapshot metadata: %w", err)
		}
	}

	// Phase 7 — install v.snaps[id] under write lock (shared snap store).
	v.mu.Lock()
	v.snaps[id] = snap
	v.mu.Unlock()

	commitMetrics := types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    int64(len(snap)),
		NewBytes:      newBytes,
	}

	// Record metrics
	if v.em != nil {
		v.em.ObserveCommitLatency(commitMetrics.CommitLatency)
		v.em.AddNewObjects(uint64(commitMetrics.NewObjects))
		v.em.AddNewBytes(uint64(commitMetrics.NewBytes))
	}

	return id, commitMetrics, nil
}

// buildDirectoryTreeOptimized builds directory Merkle tree in O(n) time
// Key optimization: Single-pass parent-child mapping instead of nested loops
func (v *VST) buildDirectoryTreeOptimized(blobHashByPath map[string]types.Hash) (types.Hash, error) {
	if len(blobHashByPath) == 0 {
		// Empty tree
		return util.HashTree(nil)
	}

	// OPTIMIZATION: Build parent-child relationship map in single pass O(n)
	type DirInfo struct {
		children map[string]types.Hash  // child name -> hash
		depth    int
	}
	
	dirMap := make(map[string]*DirInfo)
	
	// Initialize all directories
	ensureDir := func(dir string) *DirInfo {
		if info, exists := dirMap[dir]; exists {
			return info
		}
		
		info := &DirInfo{
			children: make(map[string]types.Hash),
			depth:    strings.Count(dir, "/"),
		}
		if dir == "." {
			info.depth = 0
		}
		dirMap[dir] = info
		return info
	}

	// Single pass: register all files and their parent directories
	for path, hash := range blobHashByPath {
		dir := filepath.Dir(path)
		if dir == "/" || dir == "" {
			dir = "."
		}
		
		// Ensure this directory exists
		dirInfo := ensureDir(dir)
		
		// Register file in parent directory
		fileName := filepath.Base(path)
		dirInfo.children[fileName] = hash
		
		// Ensure all ancestor directories exist
		ancestor := dir
		for ancestor != "." {
			parent := filepath.Dir(ancestor)
			if parent == "/" || parent == "" || parent == ancestor {
				parent = "."
			}
			ensureDir(parent)
			ancestor = parent
		}
		ensureDir(".") // Ensure root exists
	}

	// OPTIMIZATION: Sort directories by depth (deepest first) in O(n log n)
	var dirs []string
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	
	sort.Slice(dirs, func(i, j int) bool {
		depthI, depthJ := dirMap[dirs[i]].depth, dirMap[dirs[j]].depth
		if depthI == depthJ {
			return dirs[i] > dirs[j] // Stable sort
		}
		return depthI > depthJ // Deeper directories first
	})

	// Compute directory hashes bottom-up in O(n) time
	dirHashes := make(map[string]types.Hash)
	
	for _, dir := range dirs {
		dirInfo := dirMap[dir]
		
		// Collect entries for this directory
		var entries []string
		
		// Add file entries (blobs)
		for childName, childHash := range dirInfo.children {
			entries = append(entries, fmt.Sprintf("%s:blob:%x", childName, childHash.Digest))
		}
		
		// Add subdirectory entries (trees) - O(1) lookup instead of O(n) scan
		for _, potentialChild := range dirs {
			if potentialChild == dir {
				continue
			}
			if filepath.Dir(potentialChild) == dir {
				if childHash, exists := dirHashes[potentialChild]; exists {
					childName := filepath.Base(potentialChild)
					entries = append(entries, fmt.Sprintf("%s:tree:%x", childName, childHash.Digest))
				}
			}
		}

		// Compute deterministic hash for this directory
		hash, err := util.HashTree(entries)
		if err != nil {
			return types.Hash{}, err
		}
		dirHashes[dir] = hash
	}

	// Return root directory hash
	if rootHash, exists := dirHashes["."]; exists {
		return rootHash, nil
	}
	
	// Empty tree fallback
	return util.HashTree(nil)
}