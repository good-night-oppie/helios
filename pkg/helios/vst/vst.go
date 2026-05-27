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
	"sync"
	"time"

	"github.com/good-night-oppie/helios/internal/metrics"
	"github.com/good-night-oppie/helios/internal/util"
	"github.com/good-night-oppie/helios/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
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
//
// Multi-tenancy: per-tenant working state lives in `agents`, keyed by AgentId.
// The legacy single-tenant API resolves to AgentDefault. Snapshots remain
// content-addressed and shared across agents via the global `snaps` table so
// that identical content from any agent yields the same SnapshotID.
type VST struct {
	agents map[AgentId]*agentState                // per-agent cur / pathToHash / branches
	snaps  map[types.SnapshotID]map[string][]byte // snapshot store — global, content-addressed
	l1     l1cache.Cache                          // L1 cache (hot data)
	l2     objstore.Store                         // L2 persistent store
	em     *metrics.EngineMetrics                 // engine metrics collector
	mu     sync.RWMutex                           // protects agents (and each agentState), snaps
}

// New returns a fresh VST with the default agent pre-created so the legacy
// single-tenant API is immediately usable without an extra allocation.
func New() *VST {
	v := &VST{
		agents: make(map[AgentId]*agentState),
		snaps:  make(map[types.SnapshotID]map[string][]byte),
		em:     metrics.NewEngineMetrics(),
	}
	v.agents[AgentDefault] = newAgentState()
	return v
}

// AttachStores attaches L1 cache and L2 object store to the VST.
func (v *VST) AttachStores(l1 l1cache.Cache, l2 objstore.Store) {
	v.l1 = l1
	v.l2 = l2
	if l1 != nil {
		dprintf("attached L1 cache: %+v", l1.Stats())
	}
}

// WriteFile writes/overwrites a file in the default agent's working set.
// Equivalent to WriteFileForAgent(AgentDefault, path, content).
func (v *VST) WriteFile(path string, content []byte) error {
	return v.WriteFileForAgent(AgentDefault, path, content)
}

// WriteFileForAgent writes/overwrites a file in the named agent's working
// set (in memory only). The agent's per-tenant state is lazily created on
// first write.
func (v *VST) WriteFileForAgent(agent AgentId, path string, content []byte) error {
	if err := validateAgent(agent); err != nil {
		return err
	}
	cp := make([]byte, len(content))
	copy(cp, content)
	v.mu.Lock()
	v.agentRW(agent).cur[path] = cp
	v.mu.Unlock()
	return nil
}

// DeleteFile removes a file from the default agent's working set.
// Equivalent to DeleteFileForAgent(AgentDefault, path).
func (v *VST) DeleteFile(path string) {
	v.DeleteFileForAgent(AgentDefault, path)
}

// DeleteFileForAgent removes a file from the named agent's working set.
// No-op for unknown agents (no state to delete from) and silently skips
// invalid (non-UTF-8) AgentIds since this method does not return an error.
func (v *VST) DeleteFileForAgent(agent AgentId, path string) {
	if validateAgent(agent) != nil {
		return
	}
	v.mu.Lock()
	if s, ok := v.agentRO(agent); ok {
		delete(s.cur, path)
	}
	v.mu.Unlock()
}

// ReadFile reads a file from the default agent's working set (copy returned).
// Equivalent to ReadFileForAgent(AgentDefault, path).
// If the file is not in memory but L1/L2 stores are attached, it tries L1 then L2.
func (v *VST) ReadFile(path string) ([]byte, error) {
	return v.ReadFileForAgent(AgentDefault, path)
}

// ReadFileForAgent reads a file from the named agent's working set.
// If the file is not in memory but L1/L2 stores are attached, it tries
// L1 then L2 via the agent's path-to-hash map.
func (v *VST) ReadFileForAgent(agent AgentId, path string) ([]byte, error) {
	if err := validateAgent(agent); err != nil {
		return nil, err
	}
	// First check current working set
	v.mu.RLock()
	var (
		b       []byte
		ok      bool
		hash    types.Hash
		hasHash bool
	)
	if s, agentOk := v.agentRO(agent); agentOk {
		b, ok = s.cur[path]
		hash, hasHash = s.pathToHash[path]
	}
	v.mu.RUnlock()

	if ok {
		cp := make([]byte, len(b))
		copy(cp, b)
		return cp, nil
	}

	// Try to get from L1/L2 using stored hash
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

// Commit creates a snapshot of the default agent's working set and returns
// a content-addressed SnapshotID (Merkle root). Equivalent to
// CommitForAgent(AgentDefault, msg).
func (v *VST) Commit(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	return v.CommitForAgent(AgentDefault, msg)
}

// CommitForAgent creates a snapshot of the named agent's working set.
// The returned SnapshotID is content-addressed: identical content from
// any agent yields the same SnapshotID and shares storage in v.snaps.
func (v *VST) CommitForAgent(agent AgentId, msg string) (types.SnapshotID, types.CommitMetrics, error) {
	if err := validateAgent(agent); err != nil {
		return "", types.CommitMetrics{}, err
	}
	_ = msg // commit message currently unused (parity with Commit)
	start := time.Now()

	// Deep copy current working set for the stored snapshot (restore/materialize rely on this).
	v.mu.RLock()
	var (
		snap     map[string][]byte
		newBytes int64
	)
	if s, ok := v.agentRO(agent); ok {
		snap = make(map[string][]byte, len(s.cur))
		for k, val := range s.cur {
			cp := make([]byte, len(val))
			copy(cp, val)
			snap[k] = cp
			newBytes += int64(len(cp))
		}
	} else {
		// Unknown agent: commit an empty working set (matches single-tenant
		// behaviour where a fresh VST commits to the empty-tree snapshot).
		snap = map[string][]byte{}
	}
	v.mu.RUnlock()

	// Compute Merkle root over the current working set.
	// Algorithm:
	//  1) For each file path -> hash blob(content)
	//  2) Aggregate bottom-up by directory: "name:type:childHash"
	//  3) The root (".") tree hash becomes SnapshotID
	blobHashByPath := make(map[string]types.Hash, len(snap))
	blobsToStore := make([]objstore.BatchEntry, 0, len(snap))
	for path, content := range snap {
		h, err := util.HashBlob(content)
		if err != nil {
			return "", types.CommitMetrics{}, err
		}
		blobHashByPath[path] = h

		// Prepare for L2 storage if attached
		if v.l2 != nil {
			blobsToStore = append(blobsToStore, objstore.BatchEntry{
				Hash:  h,
				Value: content,
			})
		}
	}

	// Store path->hash mapping for L1/L2 retrieval (must be done with lock)
	v.mu.Lock()
	{
		s := v.agentRW(agent)
		for path, h := range blobHashByPath {
			s.pathToHash[path] = h
		}
	}
	v.mu.Unlock()

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
			Hash: types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)},
			Value: metadataBytes,
		}}
		
		if err := v.l2.PutBatch(snapshotMetadata); err != nil {
			return "", types.CommitMetrics{}, fmt.Errorf("failed to store snapshot metadata: %w", err)
		}
	}

	// Store the snapshot by content (keeps your existing restore/materialize/diff working)
	v.mu.Lock()
	v.snaps[id] = snap
	v.mu.Unlock()

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

// Restore replaces the default agent's working set with the files from the given
// snapshot and writes them to the filesystem. Equivalent to RestoreWithOpts with
// WriteToFilesystem=true.
func (v *VST) Restore(id types.SnapshotID) error {
	return v.RestoreWithOpts(id, types.RestoreOpts{WriteToFilesystem: true})
}

// RestoreWithOpts replaces the default agent's working set with the snapshot's
// contents. Equivalent to RestoreForAgent(AgentDefault, id, opts).
func (v *VST) RestoreWithOpts(id types.SnapshotID, opts types.RestoreOpts) error {
	return v.RestoreForAgent(AgentDefault, id, opts)
}

// RestoreForAgent replaces the named agent's working set with the snapshot's
// contents. Behaviour mirrors RestoreWithOpts.
// - DryRun: when true, only restores to memory without filesystem writes
// - WriteToFilesystem: when true, writes restored files to disk atomically
func (v *VST) RestoreForAgent(agent AgentId, id types.SnapshotID, opts types.RestoreOpts) error {
	if err := validateAgent(agent); err != nil {
		return err
	}
	v.mu.RLock()
	base, ok := v.snaps[id]
	dprintf("starting restore of snapshot %s (in-memory snapshots=%d)", id, len(v.snaps))

	// Save the old tracked files before they get overwritten (for stale file cleanup)
	oldTrackedFiles := make(map[string]bool)
	if s, agentOk := v.agentRO(agent); agentOk {
		for path := range s.cur {
			oldTrackedFiles[path] = true
		}
	}
	v.mu.RUnlock()

	if !ok && v.l2 == nil {
		return fmt.Errorf("unknown snapshot: %s", id)
	}
	
	// Prepare the content to restore
	var restoredContent map[string][]byte
	
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
			
			// Fetch file contents from L2
			restoredContent = make(map[string][]byte)
			for path, hash := range snapshotData {
				data, ok, err := v.l2.Get(hash)
				if err != nil {
					return fmt.Errorf("failed to get file %s: %w", path, err)
				}
				if !ok {
					return fmt.Errorf("missing file data for %s", path)
				}
				restoredContent[path] = data
			}

			// Reset agent working state and use snapshot metadata as path→hash mapping
			v.mu.Lock()
			s := v.agentRW(agent)
			s.cur = restoredContent
			s.pathToHash = snapshotData
			v.mu.Unlock()
		}
	} else {
		// Copy in-memory snapshot to working set
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
		v.mu.Lock()
		s := v.agentRW(agent)
		s.cur = next
		s.pathToHash = pathHashes
		v.mu.Unlock()
		restoredContent = next
	}
	
	// Write restored content to filesystem if not in dry-run mode
	if !opts.DryRun && opts.WriteToFilesystem {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		
		// Write all restored files
		for path, content := range restoredContent {
			fullPath := filepath.Join(cwd, path)
			
			// Create directory if needed
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			
			// Write to a temporary file first for atomicity
			tempFile := fullPath + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
			if err := os.WriteFile(tempFile, content, 0644); err != nil {
				return fmt.Errorf("failed to write temp file %s: %w", tempFile, err)
			}
			
			// Atomically move temp file to final location
			if err := os.Rename(tempFile, fullPath); err != nil {
				// Clean up temp file if rename failed
				os.Remove(tempFile)
				return fmt.Errorf("failed to move file to final location %s: %w", path, err)
			}
			
			// Mark this file as processed (no longer stale)
			delete(oldTrackedFiles, path)
			
			dprintf("restored file %s (%d bytes)", path, len(content))
		}
		
		// Clean up stale files that were tracked before but are not in the restored snapshot
		for stalePath := range oldTrackedFiles {
			fullPath := filepath.Join(cwd, stalePath)
			if err := os.Remove(fullPath); err != nil {
				// Log the error but don't fail the restore operation
				dprintf("warning: failed to remove stale file %s: %v", stalePath, err)
			} else {
				dprintf("removed stale file %s", stalePath)
			}
		}
	} else if opts.DryRun || !opts.WriteToFilesystem {
		dprintf("dry-run mode: skipped writing %d files to filesystem", len(restoredContent))
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

// Close releases resources, including closing the L2 store if attached.
func (v *VST) Close() error {
	if v.l2 != nil {
		return v.l2.Close()
	}
	return nil
}
