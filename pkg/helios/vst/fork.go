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

// Package vst — VFSFork overlay primitive (ionq Vector B core).
//
// A Fork is a copy-on-write overlay over an immutable base SnapshotID. Writes
// stay in fork-local RAM (no RocksDB I/O) until MergeInto succeeds via an
// atomic compare-and-swap on a branch head. Multiple forks may diverge from
// the same base in parallel without contending on the base snapshot.
//
// Lifecycle: Fork → Write/Read/Diff → MergeInto OR Discard.
// Discard is mandatory; runtime.SetFinalizer is wired as a safety net only.

package vst

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/good-night-oppie/helios/internal/util"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// BranchID names a mutable head pointer that fork merges advance.
type BranchID string

// ChangeKind classifies the kind of divergence a Fork has from its base.
type ChangeKind uint8

const (
	// ChangeAdded — path did not exist in base, present in fork.
	ChangeAdded ChangeKind = iota + 1
	// ChangeModified — path exists in both base and fork with different content.
	ChangeModified
	// ChangeDeleted — path exists in base, tombstoned in fork.
	ChangeDeleted
)

func (k ChangeKind) String() string {
	switch k {
	case ChangeAdded:
		return "added"
	case ChangeModified:
		return "modified"
	case ChangeDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// Change describes a single path divergence between fork and base.
type Change struct {
	Path    string
	Kind    ChangeKind
	OldHash types.Hash // zero for ChangeAdded
	NewHash types.Hash // zero for ChangeDeleted
}

// Sentinel errors used by Fork / branch operations.
var (
	// ErrForkDiscarded — operation attempted on a Discarded fork.
	ErrForkDiscarded = errors.New("vst: fork already discarded")
	// ErrForkMerged — operation attempted on an already-merged fork.
	ErrForkMerged = errors.New("vst: fork already merged")
	// ErrBranchStale — MergeInto found branch head no longer equals fork.base.
	ErrBranchStale = errors.New("vst: branch head moved since fork (stale merge)")
	// ErrUnknownBranch — branch was never registered via CreateBranch.
	ErrUnknownBranch = errors.New("vst: unknown branch")
	// ErrBranchExists — CreateBranch called with an already-registered name.
	ErrBranchExists = errors.New("vst: branch already exists")
	// ErrUnknownSnapshot — supplied SnapshotID does not exist in VST.
	ErrUnknownSnapshot = errors.New("vst: unknown snapshot")
)

// Fork state flags (atomic).
const (
	forkOpen      int32 = 0
	forkMerged    int32 = 1
	forkDiscarded int32 = 2
)

// Fork is a goroutine-safe copy-on-write overlay on top of a base snapshot.
type Fork struct {
	base       types.SnapshotID
	baseSnap   map[string][]byte // shared read-only ref into v.snaps[base]
	overlay    map[string]types.Hash
	tombstones map[string]struct{}
	content    map[string][]byte // hash-digest (string) -> content blob (RAM only)
	generation uint64
	state      int32 // atomic forkOpen | forkMerged | forkDiscarded
	vst        *VST
	mu         sync.RWMutex
}

// nextForkGen is the monotonically increasing source for Fork.generation.
var nextForkGen uint64

// Fork creates a new copy-on-write overlay rooted at base.
// The base snapshot must exist either in-memory or in the attached L2 store.
// Multiple Forks from the same base may be used concurrently.
//
// Discard MUST be called when the fork is no longer needed; a runtime finalizer
// is wired as a defensive safety net but should not be relied upon.
func (v *VST) Fork(base types.SnapshotID) (*Fork, error) {
	v.mu.RLock()
	snap, ok := v.snaps[base]
	v.mu.RUnlock()
	if !ok {
		// Allow forking off an L2-resident snapshot too.
		loaded, err := v.getSnapshot(base)
		if err != nil {
			return nil, fmt.Errorf("vst.Fork: %w (base=%s)", ErrUnknownSnapshot, base)
		}
		snap = loaded
	}

	f := &Fork{
		base:       base,
		baseSnap:   snap, // immutable reference; safe to share across forks
		overlay:    make(map[string]types.Hash),
		tombstones: make(map[string]struct{}),
		content:    make(map[string][]byte),
		generation: atomic.AddUint64(&nextForkGen, 1),
		vst:        v,
	}
	runtime.SetFinalizer(f, func(g *Fork) { g.Discard() })
	return f, nil
}

// Base returns the immutable base snapshot id this fork was created from.
func (f *Fork) Base() types.SnapshotID { return f.base }

// Generation returns a process-unique monotonically increasing fork id.
func (f *Fork) Generation() uint64 { return f.generation }

func (f *Fork) ensureOpen() error {
	switch atomic.LoadInt32(&f.state) {
	case forkMerged:
		return ErrForkMerged
	case forkDiscarded:
		return ErrForkDiscarded
	}
	return nil
}

// Write installs content at path in the fork overlay.
// Content is hashed (BLAKE3) and held in fork-local RAM; no RocksDB I/O.
//
// The hash compute is done outside the lock (pure CPU, no shared state) but
// the ensureOpen() state check happens inside the lock to close the TOCTOU
// race with a concurrent Discard: if Discard wins the state CAS after our
// check but before our Lock, Discard would nil the overlay/content maps
// underneath us. With the check held under the same lock Discard later
// acquires, that ordering can no longer produce a nil-map panic.
func (f *Fork) Write(path string, content []byte) error {
	cp := make([]byte, len(content))
	copy(cp, content)
	h, err := util.HashBlob(cp)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.ensureOpen(); err != nil {
		return err
	}
	f.overlay[path] = h
	delete(f.tombstones, path)
	f.content[string(h.Digest)] = cp
	return nil
}

// Delete tombstones a path in the fork. Read will return nil for that path
// even if it exists in base. The tombstone is honoured at MergeInto time.
// ensureOpen() is held inside the lock; see Write for the TOCTOU rationale.
func (f *Fork) Delete(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.ensureOpen(); err != nil {
		return err
	}
	delete(f.overlay, path)
	f.tombstones[path] = struct{}{}
	return nil
}

// Read returns the content of path as visible through the fork:
// overlay → tombstone (nil) → base. Returns (nil, nil) if path is absent.
// The returned slice is a copy and may be modified by the caller.
//
// The RLock excludes Discard's write Lock, so a concurrent Discard cannot
// nil the maps mid-read. ensureOpen() is checked inside the RLock to close
// the TOCTOU window between an outside-the-lock check and lock acquisition.
func (f *Fork) Read(path string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if err := f.ensureOpen(); err != nil {
		return nil, err
	}
	if _, deleted := f.tombstones[path]; deleted {
		return nil, nil
	}
	if h, ok := f.overlay[path]; ok {
		if c, ok2 := f.content[string(h.Digest)]; ok2 {
			cp := make([]byte, len(c))
			copy(cp, c)
			return cp, nil
		}
	}
	if f.baseSnap == nil {
		return nil, nil
	}
	if v, ok := f.baseSnap[path]; ok {
		cp := make([]byte, len(v))
		copy(cp, v)
		return cp, nil
	}
	return nil, nil
}

// Diff returns the set of path-level changes between the fork and its base,
// sorted by path for deterministic output. Safe to call concurrently.
// ensureOpen() is checked inside the RLock; see Read for the rationale.
func (f *Fork) Diff() []Change {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if err := f.ensureOpen(); err != nil {
		return nil
	}

	changes := make([]Change, 0, len(f.overlay)+len(f.tombstones))
	for path, h := range f.overlay {
		baseContent, exists := f.baseSnap[path]
		if !exists {
			changes = append(changes, Change{Path: path, Kind: ChangeAdded, NewHash: h})
			continue
		}
		baseHash, err := util.HashBlob(baseContent)
		if err != nil {
			continue // skip silently; hashing a known-good blob shouldn't fail
		}
		if !hashEqual(baseHash, h) {
			changes = append(changes, Change{
				Path:    path,
				Kind:    ChangeModified,
				OldHash: baseHash,
				NewHash: h,
			})
		}
	}
	for path := range f.tombstones {
		baseContent, exists := f.baseSnap[path]
		if !exists {
			continue // tombstoning a non-existent path is a no-op
		}
		baseHash, err := util.HashBlob(baseContent)
		if err != nil {
			continue
		}
		changes = append(changes, Change{
			Path:    path,
			Kind:    ChangeDeleted,
			OldHash: baseHash,
		})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}

// Discard releases fork-held resources and marks the fork unusable.
// Safe to call multiple times. Safe to call from a runtime finalizer.
// Idempotent: a fork already merged will not be flipped to discarded.
func (f *Fork) Discard() {
	if !atomic.CompareAndSwapInt32(&f.state, forkOpen, forkDiscarded) {
		return
	}
	f.mu.Lock()
	f.overlay = nil
	f.tombstones = nil
	f.content = nil
	f.baseSnap = nil
	f.mu.Unlock()
	runtime.SetFinalizer(f, nil)
}

// MergeInto atomically advances the named branch from f.base to a new snapshot
// composed of base + overlay - tombstones. Compare-and-swap semantics: if the
// branch head no longer equals f.base, the merge is rejected with ErrBranchStale
// and the fork remains usable (caller may rebase + retry).
//
// On success the new snapshot is materialised into VST (and L2 if attached),
// the branch head is advanced, and the fork is transitioned to merged state.
func (f *Fork) MergeInto(branch BranchID) (types.SnapshotID, error) {
	v := f.vst

	// --- Phase 1: snapshot fork state under read lock (no VST lock yet) ---
	// ensureOpen() is checked inside the RLock so a concurrent Discard cannot
	// nil baseSnap/overlay/content/tombstones between the check and the read.
	f.mu.RLock()
	if err := f.ensureOpen(); err != nil {
		f.mu.RUnlock()
		return "", err
	}
	composed := make(map[string][]byte, len(f.baseSnap)+len(f.overlay))
	for k, val := range f.baseSnap {
		composed[k] = val // share immutable base bytes; we deep-copy at store time
	}
	for path, h := range f.overlay {
		composed[path] = f.content[string(h.Digest)]
	}
	for path := range f.tombstones {
		delete(composed, path)
	}
	overlayCopy := make(map[string]types.Hash, len(f.overlay))
	for k, h := range f.overlay {
		overlayCopy[k] = h
	}
	contentCopy := make(map[string][]byte, len(f.content))
	for k, c := range f.content {
		contentCopy[k] = c
	}
	f.mu.RUnlock()

	// --- Phase 2: compute Merkle root (no locks held) ---
	blobHashByPath := make(map[string]types.Hash, len(composed))
	for path, content := range composed {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", err
		}
		blobHashByPath[path] = h
	}
	root, err := v.buildDirectoryTreeOptimized(blobHashByPath)
	if err != nil {
		return "", err
	}
	newID := types.SnapshotID(root.String())

	// --- Phase 3: atomic CAS on branch head + install snapshot in VST ---
	v.mu.Lock()
	cur, ok := v.branches[branch]
	if !ok {
		v.mu.Unlock()
		return "", ErrUnknownBranch
	}
	if cur != f.base {
		v.mu.Unlock()
		return "", ErrBranchStale
	}
	storedSnap := make(map[string][]byte, len(composed))
	for k, val := range composed {
		c := make([]byte, len(val))
		copy(c, val)
		storedSnap[k] = c
	}
	v.snaps[newID] = storedSnap
	for path, h := range overlayCopy {
		v.pathToHash[path] = h
	}
	v.branches[branch] = newID
	l2 := v.l2
	v.mu.Unlock()

	// --- Phase 4: persist new blobs + snapshot metadata to L2 (outside lock) ---
	if l2 != nil {
		entries := make([]objstore.BatchEntry, 0, len(overlayCopy)+1)
		for _, h := range overlayCopy {
			c, ok := contentCopy[string(h.Digest)]
			if !ok {
				continue
			}
			entries = append(entries, objstore.BatchEntry{Hash: h, Value: c})
		}
		if len(entries) > 0 {
			if err := l2.PutBatch(entries); err != nil {
				return newID, fmt.Errorf("vst.MergeInto: L2 blob persist failed: %w", err)
			}
		}
		metaBytes, err := json.Marshal(blobHashByPath)
		if err != nil {
			return newID, fmt.Errorf("vst.MergeInto: marshal metadata: %w", err)
		}
		snapKey := "snapshot:" + string(newID)
		metaEntry := objstore.BatchEntry{
			Hash:  types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapKey)},
			Value: metaBytes,
		}
		if err := l2.PutBatch([]objstore.BatchEntry{metaEntry}); err != nil {
			return newID, fmt.Errorf("vst.MergeInto: L2 metadata persist failed: %w", err)
		}
	}

	// --- Phase 5: transition fork to merged (no Discard rollback path) ---
	atomic.StoreInt32(&f.state, forkMerged)
	runtime.SetFinalizer(f, nil)
	f.mu.Lock()
	f.overlay = nil
	f.tombstones = nil
	f.content = nil
	f.baseSnap = nil
	f.mu.Unlock()
	return newID, nil
}

// hashEqual is a small helper to avoid importing bytes only for this check.
func hashEqual(a, b types.Hash) bool {
	if a.Algorithm != b.Algorithm || len(a.Digest) != len(b.Digest) {
		return false
	}
	for i := range a.Digest {
		if a.Digest[i] != b.Digest[i] {
			return false
		}
	}
	return true
}
