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
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// helper: commit a base snapshot with the given files and return a registered branch.
func setupBase(t *testing.T, files map[string]string) (*VST, BranchID, string) {
	t.Helper()
	v := New()
	for path, content := range files {
		if err := v.WriteFile(path, []byte(content)); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}
	id, _, err := v.Commit("base")
	if err != nil {
		t.Fatalf("Commit base: %v", err)
	}
	branch := BranchID("main")
	if err := v.CreateBranch(branch, id); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	return v, branch, string(id)
}

func TestFork_BasicReadWriteMerge(t *testing.T) {
	v, branch, baseID := setupBase(t, map[string]string{
		"a.txt":      "alpha",
		"dir/b.txt":  "beta",
	})

	f, err := v.Fork(v.mustHead(t, branch))
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	defer f.Discard()

	// Read from base.
	got, err := f.Read("a.txt")
	if err != nil || string(got) != "alpha" {
		t.Fatalf("Read a.txt = %q, %v; want alpha", got, err)
	}

	// Overlay write + read overrides base.
	if err := f.Write("a.txt", []byte("alpha-2")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, _ := f.Read("a.txt"); string(got) != "alpha-2" {
		t.Fatalf("overlay Read a.txt = %q; want alpha-2", got)
	}
	// Unaffected base path still readable.
	if got, _ := f.Read("dir/b.txt"); string(got) != "beta" {
		t.Fatalf("base passthrough dir/b.txt = %q; want beta", got)
	}
	// Add a brand-new path.
	if err := f.Write("c.txt", []byte("gamma")); err != nil {
		t.Fatalf("Write c.txt: %v", err)
	}

	// Diff must show modified + added.
	diff := f.Diff()
	if len(diff) != 2 {
		t.Fatalf("Diff len = %d; want 2 (%+v)", len(diff), diff)
	}
	gotKinds := map[string]ChangeKind{}
	for _, c := range diff {
		gotKinds[c.Path] = c.Kind
	}
	if gotKinds["a.txt"] != ChangeModified || gotKinds["c.txt"] != ChangeAdded {
		t.Fatalf("Diff kinds wrong: %+v", gotKinds)
	}

	// Merge advances branch head atomically.
	newID, err := f.MergeInto(branch)
	if err != nil {
		t.Fatalf("MergeInto: %v", err)
	}
	if string(newID) == baseID {
		t.Fatalf("MergeInto returned base id; expected new snapshot")
	}
	head, _ := v.BranchHead(branch)
	if head != newID {
		t.Fatalf("branch head not advanced: %s vs %s", head, newID)
	}

	// Fork is now merged; further ops must fail.
	if err := f.Write("x", []byte("x")); !errors.Is(err, ErrForkMerged) {
		t.Fatalf("post-merge Write err = %v; want ErrForkMerged", err)
	}
	// Discard after merge is a no-op.
	f.Discard()

	// Validate snapshot content via Diff between base and new head.
	stats, err := v.Diff(v.mustSnapshot(t, baseID), newID)
	if err != nil {
		t.Fatalf("v.Diff: %v", err)
	}
	if stats.Added != 1 || stats.Changed != 1 || stats.Deleted != 0 {
		t.Fatalf("stats=%+v; want added=1 changed=1 deleted=0", stats)
	}
}

func TestFork_Tombstone(t *testing.T) {
	v, branch, _ := setupBase(t, map[string]string{"keep.txt": "k", "rm.txt": "r"})
	f, err := v.Fork(v.mustHead(t, branch))
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	defer f.Discard()

	if err := f.Delete("rm.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, _ := f.Read("rm.txt"); got != nil {
		t.Fatalf("tombstoned Read = %q; want nil", got)
	}
	diff := f.Diff()
	if len(diff) != 1 || diff[0].Kind != ChangeDeleted || diff[0].Path != "rm.txt" {
		t.Fatalf("Diff = %+v; want single ChangeDeleted rm.txt", diff)
	}

	newID, err := f.MergeInto(branch)
	if err != nil {
		t.Fatalf("MergeInto: %v", err)
	}
	// keep.txt must survive; rm.txt must be gone.
	snap, err := v.getSnapshot(newID)
	if err != nil {
		t.Fatalf("getSnapshot: %v", err)
	}
	if _, ok := snap["rm.txt"]; ok {
		t.Fatalf("rm.txt not deleted in merged snapshot")
	}
	if !bytes.Equal(snap["keep.txt"], []byte("k")) {
		t.Fatalf("keep.txt content lost: %q", snap["keep.txt"])
	}
}

func TestFork_DiscardNoPersistence(t *testing.T) {
	v, branch, baseID := setupBase(t, map[string]string{"a.txt": "alpha"})
	f, err := v.Fork(v.mustHead(t, branch))
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if err := f.Write("ghost.txt", []byte("should-not-survive")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	f.Discard()

	// Branch head unchanged.
	head, _ := v.BranchHead(branch)
	if string(head) != baseID {
		t.Fatalf("branch head changed after Discard: %s vs %s", head, baseID)
	}
	// Post-discard ops fail.
	if err := f.Write("x", []byte("y")); !errors.Is(err, ErrForkDiscarded) {
		t.Fatalf("post-discard Write err = %v; want ErrForkDiscarded", err)
	}
	if _, err := f.MergeInto(branch); !errors.Is(err, ErrForkDiscarded) {
		t.Fatalf("post-discard MergeInto err = %v; want ErrForkDiscarded", err)
	}
	// Discard idempotent.
	f.Discard()
}

func TestFork_StaleBranchCAS(t *testing.T) {
	v, branch, _ := setupBase(t, map[string]string{"a.txt": "alpha"})
	base := v.mustHead(t, branch)

	fa, err := v.Fork(base)
	if err != nil {
		t.Fatalf("Fork a: %v", err)
	}
	defer fa.Discard()
	fb, err := v.Fork(base)
	if err != nil {
		t.Fatalf("Fork b: %v", err)
	}
	defer fb.Discard()

	if err := fa.Write("a.txt", []byte("alpha-A")); err != nil {
		t.Fatalf("fa.Write: %v", err)
	}
	if err := fb.Write("a.txt", []byte("alpha-B")); err != nil {
		t.Fatalf("fb.Write: %v", err)
	}

	if _, err := fa.MergeInto(branch); err != nil {
		t.Fatalf("fa.MergeInto: %v", err)
	}
	// fb is now stale.
	if _, err := fb.MergeInto(branch); !errors.Is(err, ErrBranchStale) {
		t.Fatalf("fb.MergeInto err = %v; want ErrBranchStale", err)
	}
	// fb is still usable (state == open) but Discard required.
	if err := fb.ensureOpen(); err != nil {
		t.Fatalf("fb should remain open after stale CAS: %v", err)
	}
}

func TestFork_UnknownBranchAndSnapshot(t *testing.T) {
	v := New()
	if _, err := v.Fork("nonexistent-snap"); err == nil {
		t.Fatalf("Fork unknown snap should error")
	}
	_, _, err := v.Commit("seed")
	if err != nil {
		t.Fatalf("seed commit: %v", err)
	}
	if _, ok := v.BranchHead("nope"); ok {
		t.Fatalf("BranchHead nope ok=true; want false")
	}
	if err := v.DeleteBranch("nope"); !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("DeleteBranch nope err = %v; want ErrUnknownBranch", err)
	}
}

// TestFork_ParallelDivergent runs 32 forks × 100 ops in parallel from the same
// base. Each fork writes to a disjoint path namespace then merges to its own
// branch (so all CAS succeeds) — verifies no contention races on the base.
func TestFork_ParallelDivergent(t *testing.T) {
	v, baseBranch, _ := setupBase(t, map[string]string{"root.txt": "root"})
	base := v.mustHead(t, baseBranch)

	const K = 32
	const N = 100
	branches := make([]BranchID, K)
	for i := 0; i < K; i++ {
		b := BranchID(fmt.Sprintf("worker-%02d", i))
		if err := v.CreateBranch(b, base); err != nil {
			t.Fatalf("CreateBranch %s: %v", b, err)
		}
		branches[i] = b
	}

	var wg sync.WaitGroup
	errs := make(chan error, K)
	wg.Add(K)
	for i := 0; i < K; i++ {
		go func(idx int) {
			defer wg.Done()
			f, err := v.Fork(base)
			if err != nil {
				errs <- fmt.Errorf("worker %d Fork: %w", idx, err)
				return
			}
			for j := 0; j < N; j++ {
				path := fmt.Sprintf("w%02d/file%03d.txt", idx, j)
				content := []byte(fmt.Sprintf("worker=%d op=%d", idx, j))
				if err := f.Write(path, content); err != nil {
					errs <- fmt.Errorf("worker %d Write: %w", idx, err)
					return
				}
			}
			// Sanity: base passthrough still works under parallel access.
			if got, _ := f.Read("root.txt"); string(got) != "root" {
				errs <- fmt.Errorf("worker %d base passthrough broken: %q", idx, got)
				return
			}
			if _, err := f.MergeInto(branches[idx]); err != nil {
				errs <- fmt.Errorf("worker %d MergeInto: %w", idx, err)
				return
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("%v", e)
	}

	// Every branch head must have advanced and contain N files.
	for i, b := range branches {
		head, ok := v.BranchHead(b)
		if !ok || head == base {
			t.Fatalf("branch %s not advanced (head=%s base=%s)", b, head, base)
		}
		snap, err := v.getSnapshot(head)
		if err != nil {
			t.Fatalf("getSnapshot %s: %v", b, err)
		}
		count := 0
		prefix := fmt.Sprintf("w%02d/", i)
		for k := range snap {
			if hasPrefix(k, prefix) {
				count++
			}
		}
		if count != N {
			t.Errorf("branch %s file count = %d; want %d", b, count, N)
		}
	}
}

// TestFork_ParallelSameBranchCAS — many concurrent forks racing to merge to
// the same branch. Exactly one wins per CAS round; the rest see ErrBranchStale.
// We don't require any specific winner, only that the head consistently
// advances exactly K times (with rebase-retry per loser).
func TestFork_ParallelSameBranchCAS(t *testing.T) {
	v, branch, _ := setupBase(t, map[string]string{"shared.txt": "0"})

	const K = 16
	var winners atomic.Int32
	var wg sync.WaitGroup
	wg.Add(K)
	for i := 0; i < K; i++ {
		go func(idx int) {
			defer wg.Done()
			for attempt := 0; attempt < 50; attempt++ {
				head, _ := v.BranchHead(branch)
				f, err := v.Fork(head)
				if err != nil {
					t.Errorf("worker %d Fork: %v", idx, err)
					return
				}
				if err := f.Write(fmt.Sprintf("w%02d.txt", idx), []byte("done")); err != nil {
					t.Errorf("worker %d Write: %v", idx, err)
					f.Discard()
					return
				}
				_, err = f.MergeInto(branch)
				if err == nil {
					winners.Add(1)
					return
				}
				if !errors.Is(err, ErrBranchStale) {
					t.Errorf("worker %d MergeInto unexpected err: %v", idx, err)
					f.Discard()
					return
				}
				f.Discard()
			}
			t.Errorf("worker %d exceeded retry budget", idx)
		}(i)
	}
	wg.Wait()
	if got := winners.Load(); got != K {
		t.Fatalf("winners=%d; want %d", got, K)
	}
	head, _ := v.BranchHead(branch)
	snap, _ := v.getSnapshot(head)
	if len(snap) != 1+K { // shared.txt + K workers
		t.Fatalf("final snap len=%d; want %d", len(snap), 1+K)
	}
}

func TestFork_FinalizerSafetyNet(t *testing.T) {
	v, branch, _ := setupBase(t, map[string]string{"a": "1"})
	{
		f, err := v.Fork(v.mustHead(t, branch))
		if err != nil {
			t.Fatalf("Fork: %v", err)
		}
		_ = f.Write("leak", []byte("x"))
		// Drop the only reference without calling Discard.
		_ = f
	}
	// Force GC; finalizer must Discard without panic.
	for i := 0; i < 5; i++ {
		runtime.GC()
		runtime.Gosched()
	}
	// Branch head must be unchanged.
	head, _ := v.BranchHead(branch)
	snap, _ := v.getSnapshot(head)
	if _, ok := snap["leak"]; ok {
		t.Fatalf("leak.txt persisted; finalizer should have discarded")
	}
}

// --- test helpers -------------------------------------------------------

func (v *VST) mustHead(t *testing.T, b BranchID) types.SnapshotID {
	t.Helper()
	h, ok := v.BranchHead(b)
	if !ok {
		t.Fatalf("branch %s missing", b)
	}
	return h
}

func (v *VST) mustSnapshot(t *testing.T, id string) types.SnapshotID {
	t.Helper()
	return types.SnapshotID(id)
}

func hasPrefix(s, p string) bool {
	if len(s) < len(p) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if s[i] != p[i] {
			return false
		}
	}
	return true
}

// TestFork_DiscardRaceTOCTOU stresses the TOCTOU window between an
// out-of-lock state check and lock acquisition in Fork.Write / Read /
// Delete / Diff / MergeInto. Before the fix, a concurrent Discard could
// CAS state→discarded and nil the internal maps between a method's
// ensureOpen() check and its f.mu lock, causing a nil-map write panic.
//
// Strategy: spawn many goroutines that hammer mutating ops while a single
// goroutine races to Discard the fork. Run -race so any unsynchronised
// map read/write would fire the race detector. The only acceptable per-op
// outcomes are: success, or ErrForkDiscarded. No panics, no data races.
//
// Run with: go test -v -run TestFork_DiscardRaceTOCTOU -race -count=10 ./...
func TestFork_DiscardRaceTOCTOU(t *testing.T) {
	const iters = 1000

	for i := 0; i < iters; i++ {
		v, branch, _ := setupBase(t, map[string]string{"seed.txt": "seed"})
		base := v.mustHead(t, branch)
		f, err := v.Fork(base)
		if err != nil {
			t.Fatalf("iter %d: Fork: %v", i, err)
		}

		const writers = 4
		var wg sync.WaitGroup
		wg.Add(writers + 1)

		// Writers hammer all the mutating + read paths concurrently.
		for w := 0; w < writers; w++ {
			go func(wid int) {
				defer wg.Done()
				path := fmt.Sprintf("w%d.txt", wid)
				payload := []byte(fmt.Sprintf("payload-%d", wid))
				for j := 0; j < 32; j++ {
					if err := f.Write(path, payload); err != nil && !errors.Is(err, ErrForkDiscarded) {
						t.Errorf("iter %d writer %d Write: unexpected err %v", i, wid, err)
						return
					}
					if _, err := f.Read(path); err != nil && !errors.Is(err, ErrForkDiscarded) {
						t.Errorf("iter %d writer %d Read: unexpected err %v", i, wid, err)
						return
					}
					if err := f.Delete(path); err != nil && !errors.Is(err, ErrForkDiscarded) {
						t.Errorf("iter %d writer %d Delete: unexpected err %v", i, wid, err)
						return
					}
					// Diff returns nil on discarded fork (not an error path) so
					// just exercising it is enough to catch nil-map panics.
					_ = f.Diff()
				}
			}(w)
		}

		// Single discarder racing the writers.
		go func() {
			defer wg.Done()
			// Slight stagger so the first iterations of each writer have a
			// chance to interleave with the state CAS.
			runtime.Gosched()
			f.Discard()
		}()

		wg.Wait()

		// After Discard, every op must terminate cleanly.
		if err := f.Write("post.txt", []byte("x")); !errors.Is(err, ErrForkDiscarded) {
			t.Fatalf("iter %d: post-discard Write err = %v; want ErrForkDiscarded", i, err)
		}
		if _, err := f.Read("post.txt"); !errors.Is(err, ErrForkDiscarded) {
			t.Fatalf("iter %d: post-discard Read err = %v; want ErrForkDiscarded", i, err)
		}
	}
}

// TestFork_MergeIntoVsDiscardRace exercises the analogous TOCTOU between
// MergeInto's phase-1 read snapshot and a concurrent Discard. Acceptable
// outcomes: (a) merge wins, branch advances; (b) discard wins, MergeInto
// returns ErrForkDiscarded with no branch advance. No panics, no races.
func TestFork_MergeIntoVsDiscardRace(t *testing.T) {
	const iters = 200

	for i := 0; i < iters; i++ {
		v, branch, _ := setupBase(t, map[string]string{"seed.txt": "seed"})
		base := v.mustHead(t, branch)
		f, err := v.Fork(base)
		if err != nil {
			t.Fatalf("iter %d: Fork: %v", i, err)
		}
		// Plant a few overlay writes so MergeInto has real phase-1 work.
		for k := 0; k < 4; k++ {
			path := fmt.Sprintf("k%d.txt", k)
			if err := f.Write(path, []byte("v")); err != nil {
				t.Fatalf("iter %d: Write: %v", i, err)
			}
		}

		var wg sync.WaitGroup
		wg.Add(2)
		var mergeErr error
		go func() {
			defer wg.Done()
			_, mergeErr = f.MergeInto(branch)
		}()
		go func() {
			defer wg.Done()
			runtime.Gosched()
			f.Discard()
		}()
		wg.Wait()

		head, _ := v.BranchHead(branch)
		switch {
		case mergeErr == nil:
			if head == base {
				t.Fatalf("iter %d: merge ok but head did not advance", i)
			}
		case errors.Is(mergeErr, ErrForkDiscarded):
			// Acceptable: Discard CAS'd state before phase-1 ensureOpen check.
			if head != base {
				t.Fatalf("iter %d: discard-wins case but branch advanced (head=%s)", i, head)
			}
		default:
			t.Fatalf("iter %d: unexpected MergeInto err: %v", i, mergeErr)
		}
	}
}
