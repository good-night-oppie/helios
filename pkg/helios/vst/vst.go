package vst

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// Ensure VST implements the StateManager interface at compile time.
var _ types.StateManager = (*VST)(nil)

// VST is an in-memory Virtual State Tree used for fast user-space snapshots.
type VST struct {
	cur   map[string][]byte                      // current working set
	snaps map[types.SnapshotID]map[string][]byte // snapshot store
	seq   int64                                  // monotonic ID generator
}

// New returns a fresh VST.
func New() *VST {
	return &VST{
		cur:   make(map[string][]byte),
		snaps: make(map[types.SnapshotID]map[string][]byte),
	}
}

// WriteFile writes/overwrites a file in the current working set (in memory).
func (v *VST) WriteFile(path string, content []byte) {
	cp := make([]byte, len(content))
	copy(cp, content)
	v.cur[path] = cp
}

// ReadFile reads a file from the current working set (copy returned).
func (v *VST) ReadFile(path string) []byte {
	b, ok := v.cur[path]
	if !ok {
		return nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
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
	for path, content := range v.cur {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		blobHashByPath[path] = h
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
	for k, val := range base {
		cp := make([]byte, len(val))
		copy(cp, val)
		next[k] = cp
	}
	v.cur = next
	return nil
}

// Diff compares two snapshots and returns Added/Changed/Deleted counts.
func (v *VST) Diff(from, to types.SnapshotID) (types.DiffStats, error) {
	a, ok := v.snaps[from]
	b, ok2 := v.snaps[to]
	if !ok || !ok2 {
		return types.DiffStats{}, fmt.Errorf("unknown snapshot(s)")
	}
	var st types.DiffStats
	for path, aval := range a {
		if bval, ok := b[path]; !ok {
			st.Deleted++
		} else if !bytesEqual(aval, bval) {
			st.Changed++
		}
	}
	for path := range b {
		if _, ok := a[path]; !ok {
			st.Added++
		}
	}
	return st, nil
}

// Materialize writes the files from a snapshot to a real directory on disk.
func (v *VST) Materialize(id types.SnapshotID, outDir string, _ types.MatOpts) (types.CommitMetrics, error) {
	start := time.Now()
	snap, ok := v.snaps[id]
	if !ok {
		return types.CommitMetrics{}, fmt.Errorf("unknown snapshot: %s", id)
	}

	var bytesTotal int64
	for path, content := range snap {
		dst := filepath.Join(outDir, path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return types.CommitMetrics{}, err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return types.CommitMetrics{}, err
		}
		bytesTotal += int64(len(content))
	}
	return types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    int64(len(snap)),
		NewBytes:      bytesTotal,
	}, nil
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
