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
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// newStaleTestL1 builds a small L1 cache for the test engines.
func newStaleTestL1(t *testing.T) l1cache.Cache {
	t.Helper()
	l1, err := l1cache.New(l1cache.Config{CapacityBytes: 8 << 20, CompressionThreshold: 256})
	if err != nil {
		t.Fatalf("l1cache.New: %v", err)
	}
	return l1
}

// openStaleTestStore opens the shared L2 object store. Pebble is single-writer,
// so each engine must Close() before the next opens the same directory.
func openStaleTestStore(t *testing.T, dir string) objstore.Store {
	t.Helper()
	l2, err := objstore.Open(dir, nil)
	if err != nil {
		t.Fatalf("objstore.Open(%s): %v", dir, err)
	}
	return l2
}

// TestRestore_DeletesStaleFiles reproduces the cold-CLI-process condition
// (snapshot lives only in L2, the agent working set is empty) and proves that
// restore makes the working tree MATCH the snapshot — i.e. it deletes files
// that are present on disk but absent from the restored snapshot.
//
// Root cause (pre-fix): RestoreForAgent seeds oldTrackedFiles from the
// in-memory agent working set (vst.go:459-465), which is always empty in a
// fresh process, so the stale-file cleanup loop (vst.go:581-589) iterates over
// nothing and sibling files leak in.
//
// The test isolates CWD into a temp dir (save/restore manually — no t.Chdir,
// go.mod pins go 1.22) so the cold-process reconcile walk is contained.
func TestRestore_DeletesStaleFiles(t *testing.T) {
	store := t.TempDir() // shared L2 object store

	// S1 = {keep.txt, gone.txt}
	vA := New()
	vA.AttachStores(newStaleTestL1(t), openStaleTestStore(t, store))
	if err := vA.WriteFile("keep.txt", []byte("K")); err != nil {
		t.Fatalf("write keep: %v", err)
	}
	if err := vA.WriteFile("gone.txt", []byte("G")); err != nil {
		t.Fatalf("write gone: %v", err)
	}
	s1, _, err := vA.Commit("s1")
	if err != nil {
		t.Fatalf("commit s1: %v", err)
	}
	if err := vA.Close(); err != nil {
		t.Fatalf("close vA: %v", err)
	}

	// S2 = {keep.txt} only, into the SAME content-addressed store.
	vB := New()
	vB.AttachStores(newStaleTestL1(t), openStaleTestStore(t, store))
	if err := vB.WriteFile("keep.txt", []byte("K")); err != nil {
		t.Fatalf("write keep (B): %v", err)
	}
	s2, _, err := vB.Commit("s2")
	if err != nil {
		t.Fatalf("commit s2: %v", err)
	}
	if err := vB.Close(); err != nil {
		t.Fatalf("close vB: %v", err)
	}

	// Cold restore: a FRESH engine with an empty working set. Materialize S1 to
	// populate the working dir (keep.txt + gone.txt), then restore S2 and expect
	// gone.txt to be pruned.
	work := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(old) }()

	vC := New()
	vC.AttachStores(newStaleTestL1(t), openStaleTestStore(t, store))
	defer vC.Close()
	if _, err := vC.Materialize(s1, work, types.MatOpts{}); err != nil {
		t.Fatalf("materialize s1: %v", err)
	}
	// Sanity: the stale file exists on disk before restore.
	if _, err := os.Stat(filepath.Join(work, "gone.txt")); err != nil {
		t.Fatalf("precondition: gone.txt should exist after materialize: %v", err)
	}

	if err := os.Chdir(work); err != nil {
		t.Fatalf("chdir work: %v", err)
	}
	if err := vC.Restore(s2); err != nil {
		t.Fatalf("restore s2: %v", err)
	}

	if _, err := os.Stat(filepath.Join(work, "keep.txt")); err != nil {
		t.Fatalf("keep.txt missing after restore: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "gone.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale gone.txt was not deleted by restore (err=%v)", err)
	}
}
