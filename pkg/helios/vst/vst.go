package vst

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// Commit creates a new snapshot from the current working set.
func (v *VST) Commit(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	start := time.Now()

	v.seq++
	id := types.SnapshotID(fmt.Sprintf("snap-%06d", v.seq))

	snap := make(map[string][]byte, len(v.cur))
	var newBytes int64
	for k, val := range v.cur {
		cp := make([]byte, len(val))
		copy(cp, val)
		snap[k] = cp
		newBytes += int64(len(cp))
	}
	v.snaps[id] = snap

	return id, types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    int64(len(snap)),
		NewBytes:      newBytes,
	}, nil
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
