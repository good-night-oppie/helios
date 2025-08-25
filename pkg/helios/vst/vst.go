package vst

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// Ensure VST implements the StateManager interface at compile time.
var _ types.StateManager = (*VST)(nil)

// VST is an in-memory Virtual State Tree used for fast user-space snapshots.
type VST struct {
	cur        map[string][]byte                      // current working set
	snaps      map[types.SnapshotID]map[string][]byte // snapshot store
	l1         l1cache.Cache                          // L1 cache (hot data)
	l2         objstore.Store                         // L2 persistent store
	pathToHash map[string]types.Hash                  // path -> content hash mapping for L1/L2 retrieval
}

// New returns a fresh VST.
func New() *VST {
	return &VST{
		cur:        make(map[string][]byte),
		snaps:      make(map[types.SnapshotID]map[string][]byte),
		pathToHash: make(map[string]types.Hash),
	}
}

// AttachStores attaches L1 cache and L2 object store to the VST.
func (v *VST) AttachStores(l1 l1cache.Cache, l2 objstore.Store) {
	v.l1 = l1
	v.l2 = l2
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

	// Try L1 first
	if v.l1 != nil {
		if data, ok := v.l1.Get(hash); ok {
			// Found in L1
			return data, nil
		}
		// L1 miss - will try L2
	}

	// Try L2 and promote to L1
	if v.l2 != nil {
		data, ok, err := v.l2.Get(hash)
		if err != nil {
			return nil, err
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
	if v.l2 != nil && len(blobsToStore) > 0 {
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
		entries := dirEntries[d][:0]
		entries = entries[:0]

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

	// Store the snapshot by content (keeps your existing restore/materialize/diff working)
	v.snaps[id] = snap

	metrics := types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    int64(len(snap)),
		NewBytes:      newBytes,
	}
	return id, metrics, nil
}

func depth(p string) int {
	if p == "." || p == "" || p == "/" {
		return 0
	}
	return strings.Count(filepath.Clean(p), string(os.PathSeparator))
}

// Restore replaces the current working set with the files from the given snapshot.
func (v *VST) Restore(id types.SnapshotID) error {
	base, ok := v.snaps[id]
	if !ok {
		return fmt.Errorf("unknown snapshot: %s", id)
	}
	next := make(map[string][]byte, len(base))
	pathHashes := make(map[string]types.Hash)
	for k, val := range base {
		cp := make([]byte, len(val))
		copy(cp, val)
		next[k] = cp
		// Compute and store hash for L1/L2 retrieval
		h, err := util.HashBlob(val)
		if err != nil {
			return err
		}
		pathHashes[k] = h
	}
	v.cur = next
	v.pathToHash = pathHashes
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
