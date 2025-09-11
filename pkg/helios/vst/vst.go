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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/good-night-oppie/helios-engine/internal/metrics"
	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

var heliosDebug = os.Getenv("HELIOS_DEBUG") != ""

func dprintf(format string, a ...any) {
	if heliosDebug {
		fmt.Fprintf(os.Stderr, "helios-debug: "+format+"\n", a...)
	}
}

// Ensure VST implements the StateManager interface at compile time.
var _ types.StateManager = (*VST)(nil)

// VST is an in-memory Virtual State Tree used for fast user-space snapshots.
type VST struct {
	cur        map[string][]byte                      // current working set
	snaps      map[types.SnapshotID]map[string][]byte // snapshot store
	l1         l1cache.Cache                          // L1 cache (hot data)
	l2         objstore.Store                         // L2 persistent store
	pathToHash map[string]types.Hash                  // path -> content hash mapping for L1/L2 retrieval
	em         *metrics.EngineMetrics                 // engine metrics collector
}

// New returns a fresh VST.
func New() *VST {
	return &VST{
		cur:        make(map[string][]byte),
		snaps:      make(map[types.SnapshotID]map[string][]byte),
		pathToHash: make(map[string]types.Hash),
		em:         metrics.NewEngineMetrics(),
	}
}

// AttachStores attaches L1 cache and L2 object store to the VST.
func (v *VST) AttachStores(l1 l1cache.Cache, l2 objstore.Store) {
	v.l1 = l1
	v.l2 = l2
	if l1 != nil {
		dprintf("attached L1 cache: %+v", l1.Stats())
	}
}

// WriteFile writes/overwrites a file in the current working set (in memory).
func (v *VST) WriteFile(path string, content []byte) error {
	cp := make([]byte, len(content))
	copy(cp, content)
	v.cur[path] = cp
	return nil
}

// DeleteFile removes a file from the current working set.
func (v *VST) DeleteFile(path string) {
	delete(v.cur, path)
	delete(v.pathToHash, path)
}

// ReadFile reads a file from the current working set (copy returned).
// If the file is not in memory but we have stores attached, it tries L1 then L2.
func (v *VST) ReadFile(path string) ([]byte, error) {
	// First check current working set
	b, ok := v.cur[path]
	if ok {
		cp := make([]byte, len(b))
		copy(cp, b)
		return cp, nil
	}

	// Try to get from L1/L2 using stored hash
	hash, hasHash := v.pathToHash[path]
	if !hasHash {
		return nil, nil // File doesn't exist
	}

	// Always try L1 first to ensure miss is recorded
	l1Hit := false
	var l1Data []byte
	if v.l1 != nil {
		l1Data, l1Hit = v.l1.Get(hash)
	}
	if l1Hit {
		return l1Data, nil
	}

	// On L1 miss, try L2 store
	if v.l2 != nil {
		data, ok, err := v.l2.Get(hash)
		if err != nil {
			return nil, err // Return L2 errors without affecting cache stats
		}
		if ok {
			// Found in L2, promote to L1 if available
			if v.l1 != nil {
				v.l1.Put(hash, data)
			}
			return data, nil
		}
	}

	return nil, nil // Not found anywhere
}

// Commit creates a snapshot and returns a content-addressed SnapshotID (Merkle root).
func (v *VST) Commit(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	start := time.Now()

	// Deep copy current working set for the stored snapshot (restore/materialize rely on this).
	snap := make(map[string][]byte, len(v.cur))
	var newBytes int64
	for k, val := range v.cur {
		cp := make([]byte, len(val))
		copy(cp, val)
		snap[k] = cp
		newBytes += int64(len(cp))
	}

	// Compute Merkle root over the current working set.
	// Algorithm:
	//  1) For each file path -> hash blob(content)
	//  2) Aggregate bottom-up by directory: "name:type:childHash"
	//  3) The root (".") tree hash becomes SnapshotID
	blobHashByPath := make(map[string]types.Hash, len(v.cur))
	blobsToStore := make([]objstore.BatchEntry, 0, len(v.cur))
	for path, content := range v.cur {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		blobHashByPath[path] = h
		// Store path->hash mapping for L1/L2 retrieval
		v.pathToHash[path] = h

		// Prepare for L2 storage if attached
		if v.l2 != nil {
			blobsToStore = append(blobsToStore, objstore.BatchEntry{
				Hash:  h,
				Value: content,
			})
		}
	}

	// Store blobs in L2 if attached
	dprintf("commit: l2-attached=%v, blobsToStore=%d", v.l2 != nil, len(blobsToStore))
	if heliosDebug && len(blobsToStore) > 0 {
		// Print first few blobs for debugging
		for i := 0; i < len(blobsToStore) && i < 5; i++ {
			dprintf("commit: blob[%d]=%s size=%d", i, blobsToStore[i].Hash.String(), len(blobsToStore[i].Value))
		}
	}
	if v.l2 != nil && len(blobsToStore) > 0 {
		// Store all blobs first
		if err := v.l2.PutBatch(blobsToStore); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store blobs in L2: %w", err)
		}
	}

	// Build directory -> entries list
	dirEntries := map[string][]string{} // dir -> []"name:type:childHex"
	addEntry := func(dir, name, typ string, child types.Hash) {
		dirEntries[dir] = append(dirEntries[dir], fmt.Sprintf("%s:%s:%x", name, typ, child.Digest))
	}

	// For every file, register at its parent dir
	for path, h := range blobHashByPath {
		dir := filepath.Dir(path)
		if dir == "." || dir == "/" {
			dir = "."
		}
		name := filepath.Base(path)
		addEntry(dir, name, "blob", h)

		// Ensure all ancestor dirs exist in the map
		anc := dir
		for anc != "." {
			addEntry(anc, "", ".__ensure__", types.Hash{}) // placeholder to ensure key
			anc = filepath.Dir(anc)
			if anc == "" || anc == "/" {
				anc = "."
			}
		}
		// Also ensure root exists
		_ = dirEntries["."]
	}

	// Topologically fold directories bottom-up to compute tree hashes.
	// We do this by sorting all dirs by depth (deepest first).
	allDirs := make([]string, 0, len(dirEntries))
	for d := range dirEntries {
		allDirs = append(allDirs, d)
	}
	sort.Slice(allDirs, func(i, j int) bool {
		di, dj := depth(allDirs[i]), depth(allDirs[j])
		if di == dj {
			return allDirs[i] > allDirs[j] // stable
		}
		return di > dj // deeper first
	})

	treeHash := map[string]types.Hash{}
	for _, d := range allDirs {
		// Collect children entries: files we already have; dirs we must reference if present.
		var entries []string

		// Rebuild entries: include file entries already accumulated,
		// and if directory has subdirectories, we will add them when their hash becomes available.
		// The map currently holds only "file" entries and placeholders.
		for _, raw := range dirEntries[d] {
			// filter out ensure placeholders
			if strings.Contains(raw, ":.__ensure__:") {
				continue
			}
			entries = append(entries, raw)
		}

		// Add child directories that have this dir as parent
		// (We detect by scanning all paths that start with d + "/something")
		// Simpler: derive child dirs by scanning allDirs again.
		for _, maybe := range allDirs {
			if maybe == d {
				continue
			}
			// parent detection: filepath.Dir(maybe) == d
			if filepath.Dir(maybe) == d {
				if h, ok := treeHash[maybe]; ok {
					name := filepath.Base(maybe)
					if d == "." && name == "." {
						continue
					}
					entries = append(entries, fmt.Sprintf("%s:%s:%x", name, "tree", h.Digest))
				}
			}
		}

		// Deterministic hash for this directory
		h, err := util.HashTree(entries)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		treeHash[d] = h
	}

	root, hasRoot := treeHash["."]
	if !hasRoot || len(root.Digest) == 0 {
		// Empty tree: define as hash of empty entries
		h, err := util.HashTree(nil)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		root = h
	}

	id := types.SnapshotID(root.String())

	// Store snapshot metadata in L2 before keeping in memory
	if v.l2 != nil {
		// Store snapshot metadata (file list and hashes) alongside the blobs
		snapshotData := make(map[string]types.Hash)
		for path, hash := range blobHashByPath {
			snapshotData[path] = hash
		}

		// Marshal snapshot metadata
		metadataBytes, err := json.Marshal(snapshotData)
		if err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to marshal snapshot metadata: %w", err)
		}

		// Store snapshot metadata with special prefix
		snapshotKey := string("snapshot:" + id)
		dprintf("commit: storing snapshot metadata with key %s", snapshotKey)
		snapshotMetadata := []objstore.BatchEntry{{
			Hash:  types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)},
			Value: metadataBytes,
		}}

		if err := v.l2.PutBatch(snapshotMetadata); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store snapshot metadata: %w", err)
		}
	}

	// Store the snapshot by content (keeps your existing restore/materialize/diff working)
	v.snaps[id] = snap

	commitMetrics := types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    int64(len(snap)),
		NewBytes:      newBytes,
	}

	// Record metrics if collector is available
	if v.em != nil {
		v.em.ObserveCommitLatency(commitMetrics.CommitLatency)
		v.em.AddNewObjects(uint64(commitMetrics.NewObjects))
		v.em.AddNewBytes(uint64(commitMetrics.NewBytes))
	}

	return id, commitMetrics, nil
}

func depth(p string) int {
	if p == "." || p == "" || p == "/" {
		return 0
	}
	return strings.Count(filepath.Clean(p), string(os.PathSeparator))
}

// Restore replaces the current working set with the files from the given snapshot.
func (v *VST) Restore(id types.SnapshotID) error {
	dprintf("starting restore of snapshot %s (in-memory snapshots=%+v)", id, v.snaps)
	base, ok := v.snaps[id]
	if !ok && v.l2 == nil {
		return fmt.Errorf("unknown snapshot: %s", id)
	}

	// If snapshot is not in memory but L2 is available, try to restore from L2
	if !ok {
		// Try to get snapshot metadata from L2
		if v.l2 != nil {
			dprintf("restore: trying L2 restore for %s", id)

			// Get snapshot metadata
			snapshotKey := string("snapshot:" + id)
			dprintf("restore: trying to get metadata with key %s", snapshotKey)
			metadataHash := types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)}
			metadataBytes, ok, err := v.l2.Get(metadataHash)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("unknown snapshot in L2: %s", id)
			}

			// Unmarshal metadata
			var snapshotData map[string]types.Hash
			if err := json.Unmarshal(metadataBytes, &snapshotData); err != nil {
				return fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
			}
			dprintf("restore: got snapshot metadata with %d files", len(snapshotData))

			// Reset working state and use snapshot metadata as path→hash mapping
			v.cur = make(map[string][]byte)
			v.pathToHash = snapshotData
		}
	}
	// Copy in-memory snapshot to working set if not restoring from L2
	if len(base) > 0 {
		next := make(map[string][]byte, len(base))
		pathHashes := make(map[string]types.Hash, len(base))
		for k, val := range base {
			cp := make([]byte, len(val))
			copy(cp, val)
			next[k] = cp

			// Always compute hash for in-memory snapshot files
			h, err := util.HashBlob(val)
			if err != nil {
				return err
			}
			pathHashes[k] = h
		}
		v.cur = next
		v.pathToHash = pathHashes
	}
	return nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// L1Stats returns L1 cache statistics if L1 is attached.
func (v *VST) L1Stats() l1cache.CacheStats {
	var stats l1cache.CacheStats
	if v.l1 != nil {
		stats = v.l1.Stats()
	}
	return stats
}

// EngineMetricsSnapshot exposes current metrics for CLI stats.
func (v *VST) EngineMetricsSnapshot() metrics.Snapshot {
	if v.em == nil {
		return metrics.Snapshot{}
	}
	return v.em.Snapshot()
}
