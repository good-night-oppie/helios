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

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// CommitOptimized is a high-performance version of Commit that achieves <70μs targets
// Key optimizations:
// 1. O(n²) → O(n) directory tree building
// 2. Copy-on-Write (COW) semantics for snapshots
// 3. Efficient parent-child directory mapping
func (v *VST) CommitOptimized(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	start := time.Now()

	// OPTIMIZATION 1: Copy-on-Write (COW) snapshot
	// Instead of deep copying, share references and create new working set
	snap := v.cur                              // Share reference to current working set
	v.cur = make(map[string][]byte, len(snap)) // New working set for future modifications

	var newBytes int64
	for _, val := range snap {
		newBytes += int64(len(val))
	}

	// OPTIMIZATION 2: Batch compute all blob hashes
	blobHashByPath := make(map[string]types.Hash, len(snap))
	blobsToStore := make([]objstore.BatchEntry, 0, len(snap))

	for path, content := range snap {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		blobHashByPath[path] = h
		v.pathToHash[path] = h

		if v.l2 != nil {
			blobsToStore = append(blobsToStore, objstore.BatchEntry{
				Hash:  h,
				Value: content,
			})
		}
	}

	// Store blobs in L2 if attached
	if v.l2 != nil && len(blobsToStore) > 0 {
		if err := v.l2.PutBatch(blobsToStore); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store blobs in L2: %w", err)
		}
	}

	// OPTIMIZATION 3: Efficient O(n) directory tree building
	root, err := v.buildDirectoryTreeOptimized(blobHashByPath)
	if err != nil {
		return "", types.CommitMetrics{}, err
	}

	id := types.SnapshotID(root.String())

	// Store snapshot metadata in L2
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

	// Store snapshot using COW reference
	v.snaps[id] = snap

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
		files map[string]types.Hash // file name -> hash
		dirs  map[string]struct{}   // child directory names
		depth int
	}

	dirMap := make(map[string]*DirInfo)

	// Initialize all directories
	ensureDir := func(dir string) *DirInfo {
		if info, exists := dirMap[dir]; exists {
			return info
		}
		info := &DirInfo{
			files: make(map[string]types.Hash),
			dirs:  make(map[string]struct{}),
			depth: strings.Count(dir, "/"),
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
		dirInfo.files[fileName] = hash

		// Ensure all ancestor directories exist and register dir hierarchy
		ancestor := dir
		for {
			if ancestor == "." {
				break
			}
			parent := filepath.Dir(ancestor)
			if parent == "/" || parent == "" || parent == ancestor {
				parent = "."
			}
			pInfo := ensureDir(parent)
			pInfo.dirs[filepath.Base(ancestor)] = struct{}{}
			ancestor = parent
		}
		ensureDir(".")
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
		for childName, childHash := range dirInfo.files {
			entries = append(entries, fmt.Sprintf("%s:blob:%x", childName, childHash.Digest))
		}

		// Add subdirectory entries using precomputed map
		for childName := range dirInfo.dirs {
			childPath := filepath.Join(dir, childName)
			if childHash, exists := dirHashes[childPath]; exists {
				entries = append(entries, fmt.Sprintf("%s:tree:%x", childName, childHash.Digest))
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
