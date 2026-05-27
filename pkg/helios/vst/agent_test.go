// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package vst

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// TestVST_AgentPathIsolation: two agents writing the same path must read
// back their own content. Cross-agent reads of unwritten paths return
// (nil, nil).
func TestVST_AgentPathIsolation(t *testing.T) {
	v := New()
	const a, b AgentId = "alpha", "beta"

	if err := v.WriteFileForAgent(a, "x.txt", []byte("A")); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := v.WriteFileForAgent(b, "x.txt", []byte("B")); err != nil {
		t.Fatalf("write beta: %v", err)
	}

	got, err := v.ReadFileForAgent(a, "x.txt")
	if err != nil || string(got) != "A" {
		t.Fatalf("alpha read: got=%q err=%v want=A", got, err)
	}
	got, err = v.ReadFileForAgent(b, "x.txt")
	if err != nil || string(got) != "B" {
		t.Fatalf("beta read: got=%q err=%v want=B", got, err)
	}

	// Cross-agent read of a path written only by the other agent: NotFound.
	got, err = v.ReadFileForAgent("gamma", "x.txt")
	if err != nil || got != nil {
		t.Fatalf("gamma read of unwritten path: got=%q err=%v want=nil,nil", got, err)
	}
}

// TestVST_DefaultAgentPassthrough: the legacy single-tenant API resolves
// to AgentDefault. WriteFile then ReadFileForAgent(AgentDefault, ...) sees
// the value; symmetric direction also works.
func TestVST_DefaultAgentPassthrough(t *testing.T) {
	v := New()
	if err := v.WriteFile("p", []byte("v")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := v.ReadFile("p")
	if err != nil || string(got) != "v" {
		t.Fatalf("ReadFile: got=%q err=%v want=v", got, err)
	}
	got, err = v.ReadFileForAgent(AgentDefault, "p")
	if err != nil || string(got) != "v" {
		t.Fatalf("ReadFileForAgent(default): got=%q err=%v want=v", got, err)
	}
	if err := v.WriteFileForAgent(AgentDefault, "p", []byte("v2")); err != nil {
		t.Fatalf("WriteFileForAgent(default): %v", err)
	}
	got, err = v.ReadFile("p")
	if err != nil || string(got) != "v2" {
		t.Fatalf("ReadFile after for-agent overwrite: got=%q err=%v want=v2", got, err)
	}

	// Empty-string agent normalises to AgentDefault.
	got, err = v.ReadFileForAgent("", "p")
	if err != nil || string(got) != "v2" {
		t.Fatalf("ReadFileForAgent(\"\"): got=%q err=%v want=v2", got, err)
	}
}

// TestVST_AgentBranchIsolation: two agents may register the same branch
// name pointing at different snapshots; lookups never cross agents.
func TestVST_AgentBranchIsolation(t *testing.T) {
	v := New()
	const a, b AgentId = "alpha", "beta"

	// Commit a snapshot under each agent so we have valid heads.
	if err := v.WriteFileForAgent(a, "x", []byte("a1")); err != nil {
		t.Fatalf("write a: %v", err)
	}
	sa, _, err := v.CommitForAgent(a, "alpha-1")
	if err != nil {
		t.Fatalf("commit a: %v", err)
	}
	if err := v.WriteFileForAgent(b, "x", []byte("b1")); err != nil {
		t.Fatalf("write b: %v", err)
	}
	sb, _, err := v.CommitForAgent(b, "beta-1")
	if err != nil {
		t.Fatalf("commit b: %v", err)
	}

	if err := v.CreateBranchForAgent(a, "main", sa); err != nil {
		t.Fatalf("create branch a: %v", err)
	}
	if err := v.CreateBranchForAgent(b, "main", sb); err != nil {
		t.Fatalf("create branch b: %v", err)
	}

	gotA, okA := v.BranchHeadForAgent(a, "main")
	if !okA || gotA != sa {
		t.Fatalf("alpha/main: got=%s ok=%v want=%s", gotA, okA, sa)
	}
	gotB, okB := v.BranchHeadForAgent(b, "main")
	if !okB || gotB != sb {
		t.Fatalf("beta/main: got=%s ok=%v want=%s", gotB, okB, sb)
	}
	if gotA == gotB {
		t.Fatalf("expected distinct branch heads across agents; both=%s", gotA)
	}

	// Default-agent has no branch named "main"; not visible.
	if _, ok := v.BranchHead("main"); ok {
		t.Fatalf("default agent unexpectedly sees branch main")
	}

	// DeleteBranchForAgent removes only the targeted agent's entry.
	if err := v.DeleteBranchForAgent(a, "main"); err != nil {
		t.Fatalf("delete alpha/main: %v", err)
	}
	if _, ok := v.BranchHeadForAgent(a, "main"); ok {
		t.Fatalf("alpha/main still present after delete")
	}
	if _, ok := v.BranchHeadForAgent(b, "main"); !ok {
		t.Fatalf("beta/main vanished after deleting alpha/main")
	}
}

// TestFork_AgentScopedMergeInto: a fork opened under agent_a only advances
// branches under agent_a. Identically-named branches under agent_b are
// untouched.
func TestFork_AgentScopedMergeInto(t *testing.T) {
	v := New()
	const a, b AgentId = "alpha", "beta"

	// Seed: each agent commits and registers main pointing at its own snapshot.
	if err := v.WriteFileForAgent(a, "x", []byte("a0")); err != nil {
		t.Fatal(err)
	}
	sa0, _, err := v.CommitForAgent(a, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.WriteFileForAgent(b, "x", []byte("b0")); err != nil {
		t.Fatal(err)
	}
	sb0, _, err := v.CommitForAgent(b, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.CreateBranchForAgent(a, "main", sa0); err != nil {
		t.Fatal(err)
	}
	if err := v.CreateBranchForAgent(b, "main", sb0); err != nil {
		t.Fatal(err)
	}

	// Open a fork under alpha, mutate, merge into alpha/main.
	fa, err := v.ForkForAgent(a, sa0)
	if err != nil {
		t.Fatalf("fork a: %v", err)
	}
	if fa.Agent() != a {
		t.Fatalf("fork.Agent: got=%s want=%s", fa.Agent(), a)
	}
	if err := fa.Write("x", []byte("a1")); err != nil {
		t.Fatal(err)
	}
	sa1, err := fa.MergeInto("main")
	if err != nil {
		t.Fatalf("merge alpha: %v", err)
	}
	if sa1 == sa0 {
		t.Fatalf("alpha/main did not advance: still %s", sa0)
	}
	if got, _ := v.BranchHeadForAgent(a, "main"); got != sa1 {
		t.Fatalf("alpha/main head: got=%s want=%s", got, sa1)
	}
	// beta/main untouched.
	if got, _ := v.BranchHeadForAgent(b, "main"); got != sb0 {
		t.Fatalf("beta/main moved: got=%s want=%s", got, sb0)
	}

	// Merging an alpha-scoped fork into a branch that exists only under beta
	// must return ErrUnknownBranch (the fork can't reach beta's namespace).
	if err := v.DeleteBranchForAgent(a, "main"); err != nil {
		t.Fatal(err)
	}
	fa2, err := v.ForkForAgent(a, sa1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fa2.MergeInto("main"); !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("merge into unknown branch: got=%v want=ErrUnknownBranch", err)
	}
	fa2.Discard()
}

// TestVST_AgentCommitSharedSnapshot: identical content from different
// agents produces the same SnapshotID. The snapshot is content-addressed
// globally; the per-agent dimension is on working state, not the snap store.
func TestVST_AgentCommitSharedSnapshot(t *testing.T) {
	v := New()
	const a, b AgentId = "alpha", "beta"
	payload := []byte("hello")

	if err := v.WriteFileForAgent(a, "p", payload); err != nil {
		t.Fatal(err)
	}
	if err := v.WriteFileForAgent(b, "p", payload); err != nil {
		t.Fatal(err)
	}

	sa, _, err := v.CommitForAgent(a, "")
	if err != nil {
		t.Fatal(err)
	}
	sb, _, err := v.CommitForAgent(b, "")
	if err != nil {
		t.Fatal(err)
	}
	if sa != sb {
		t.Fatalf("expected same SnapshotID for identical content; alpha=%s beta=%s", sa, sb)
	}
}

// TestVST_ConcurrentAgentsWrites: K goroutines each writing N paths under
// a distinct agent. Under -race, no data race must fire and each agent's
// final state must contain its full N paths.
func TestVST_ConcurrentAgentsWrites(t *testing.T) {
	const (
		K = 8
		N = 200
	)
	v := New()

	var wg sync.WaitGroup
	wg.Add(K)
	for k := 0; k < K; k++ {
		go func(k int) {
			defer wg.Done()
			agent := AgentId(fmt.Sprintf("a%d", k))
			for i := 0; i < N; i++ {
				path := fmt.Sprintf("p%d", i)
				val := fmt.Sprintf("a%d-v%d", k, i)
				if err := v.WriteFileForAgent(agent, path, []byte(val)); err != nil {
					t.Errorf("write k=%d i=%d: %v", k, i, err)
					return
				}
			}
		}(k)
	}
	wg.Wait()

	// Verify each agent sees its own N paths with correct content.
	for k := 0; k < K; k++ {
		agent := AgentId(fmt.Sprintf("a%d", k))
		for i := 0; i < N; i++ {
			path := fmt.Sprintf("p%d", i)
			want := fmt.Sprintf("a%d-v%d", k, i)
			got, err := v.ReadFileForAgent(agent, path)
			if err != nil {
				t.Fatalf("read k=%d i=%d: %v", k, i, err)
			}
			if string(got) != want {
				t.Fatalf("read k=%d i=%d: got=%q want=%q", k, i, got, want)
			}
		}
	}
}

// TestVST_CommitOptimizedForAgent_ConcurrentRace fires K goroutines that
// concurrently Write + CommitOptimizedForAgent + Write under distinct
// AgentIds. Pre-fix, the unlocked agentRW + s.cur swap + v.snaps write
// would race the agents-map insert from another goroutine and either
// panic ("fatal: concurrent map writes") or corrupt state. Under -race
// this exercise must run cleanly and each agent's post-commit
// working-set state must be empty (COW swap installed a fresh empty
// map) until the next round of writes populates it again.
func TestVST_CommitOptimizedForAgent_ConcurrentRace(t *testing.T) {
	const (
		K      = 8
		Rounds = 50
	)
	v := New()

	var wg sync.WaitGroup
	wg.Add(K)
	for k := 0; k < K; k++ {
		go func(k int) {
			defer wg.Done()
			agent := AgentId(fmt.Sprintf("commit-agent-%d", k))
			for r := 0; r < Rounds; r++ {
				path := fmt.Sprintf("r%d.txt", r)
				if err := v.WriteFileForAgent(agent, path, []byte(fmt.Sprintf("k%d-r%d", k, r))); err != nil {
					t.Errorf("write k=%d r=%d: %v", k, r, err)
					return
				}
				if _, _, err := v.CommitOptimizedForAgent(agent, ""); err != nil {
					t.Errorf("commit-optimized k=%d r=%d: %v", k, r, err)
					return
				}
			}
		}(k)
	}
	wg.Wait()

	// Every goroutine's inner loop is Write -> Commit (no trailing Write),
	// so the final post-loop state for each agent must have a freshly
	// COW-swapped empty cur. With L1/L2 unattached, ReadFileForAgent
	// falls through pathToHash and returns (nil, nil). Asserting that
	// got is nil for the last-written path is the externally observable
	// signal that the COW swap actually happened on the final Commit;
	// pre-fix, the racy commit could leave cur unchanged and ReadFile
	// would return the last-written value, silently regressing the
	// concurrency fix without this assertion.
	for k := 0; k < K; k++ {
		agent := AgentId(fmt.Sprintf("commit-agent-%d", k))
		lastPath := fmt.Sprintf("r%d.txt", Rounds-1)
		got, err := v.ReadFileForAgent(agent, lastPath)
		if err != nil {
			t.Fatalf("post-loop read agent=%s path=%s: %v", agent, lastPath, err)
		}
		if got != nil {
			t.Errorf("post-loop read agent=%s path=%s: expected nil (cur swapped empty by final Commit, no L1/L2 attached), got=%q",
				agent, lastPath, got)
		}
	}
}

// TestVST_ForAgent_RejectsInvalidUTF8 pins the AgentId contract from agent.go:
// error-returning *ForAgent methods must surface ErrInvalidAgent for non-UTF-8
// IDs; void / bool methods must silently no-op so a corrupt ID never reaches
// the internal v.agents map (which would otherwise pollute iteration and
// serialization downstream).
func TestVST_ForAgent_RejectsInvalidUTF8(t *testing.T) {
	v := New()
	bad := AgentId("\xff\xfe-not-utf8")

	if err := v.WriteFileForAgent(bad, "p", []byte("x")); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("WriteFileForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if _, err := v.ReadFileForAgent(bad, "p"); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("ReadFileForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if _, _, err := v.CommitForAgent(bad, ""); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("CommitForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if _, _, err := v.CommitOptimizedForAgent(bad, ""); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("CommitOptimizedForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if err := v.RestoreForAgent(bad, types.SnapshotID(""), types.RestoreOpts{DryRun: true}); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("RestoreForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if _, err := v.ForkForAgent(bad, types.SnapshotID("")); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("ForkForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if err := v.CreateBranchForAgent(bad, "b", types.SnapshotID("")); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("CreateBranchForAgent: got err=%v, want ErrInvalidAgent", err)
	}
	if err := v.DeleteBranchForAgent(bad, "b"); !errors.Is(err, ErrInvalidAgent) {
		t.Errorf("DeleteBranchForAgent: got err=%v, want ErrInvalidAgent", err)
	}

	// Void / bool methods silently no-op; key invariant is that the invalid
	// ID never lands in v.agents (would corrupt iteration / serialisation).
	v.DeleteFileForAgent(bad, "p")
	if _, ok := v.BranchHeadForAgent(bad, "b"); ok {
		t.Errorf("BranchHeadForAgent: invalid agent returned ok=true")
	}
	if m := v.BranchesForAgent(bad); len(m) != 0 {
		t.Errorf("BranchesForAgent: invalid agent returned %d branches, want 0", len(m))
	}
	v.mu.RLock()
	_, polluted := v.agents[bad]
	v.mu.RUnlock()
	if polluted {
		t.Errorf("v.agents leaked invalid AgentId %q despite validation gate", bad)
	}
}

// TestVST_RestoreForAgent_Isolation: restoring a snapshot into agent_a
// must not perturb agent_b's working state. Cross-tenant isolation
// invariant.
func TestVST_RestoreForAgent_Isolation(t *testing.T) {
	v := New()
	const a, b AgentId = "alpha", "beta"

	// Seed both agents with distinct content.
	if err := v.WriteFileForAgent(a, "p", []byte("a-original")); err != nil {
		t.Fatal(err)
	}
	if err := v.WriteFileForAgent(b, "p", []byte("b-original")); err != nil {
		t.Fatal(err)
	}

	// Commit + over-write alpha so its cur diverges from the committed snap.
	saSeed, _, err := v.CommitForAgent(a, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.WriteFileForAgent(a, "p", []byte("a-mutated")); err != nil {
		t.Fatal(err)
	}

	// Restore alpha back to its seeded snapshot. Restore writes to
	// filesystem by default, so use DryRun=true to scope this to memory.
	if err := v.RestoreForAgent(a, saSeed, types.RestoreOpts{DryRun: true}); err != nil {
		t.Fatalf("restore alpha: %v", err)
	}

	// Alpha's read returns the restored content.
	gotA, err := v.ReadFileForAgent(a, "p")
	if err != nil {
		t.Fatal(err)
	}
	if string(gotA) != "a-original" {
		t.Fatalf("alpha post-restore: got=%q want=a-original", gotA)
	}

	// Beta's working set is untouched.
	gotB, err := v.ReadFileForAgent(b, "p")
	if err != nil {
		t.Fatal(err)
	}
	if string(gotB) != "b-original" {
		t.Fatalf("beta post-alpha-restore: got=%q want=b-original (cross-tenant leak)", gotB)
	}
}

// TestVST_DeleteFileForAgent_ClearsPathToHash pins working-set delete
// semantics. Pre-fix, DeleteFileForAgent removed only s.cur[path] but
// left s.pathToHash[path] intact. After a Commit, cur is COW-swapped
// empty but pathToHash retains every committed path; a subsequent
// Delete-then-Read would still resolve via the pathToHash fallback
// (L1/L2 lookup in production wiring), so a "deleted" file could be
// silently resurrected from cache/store.
//
// White-box: directly inspect s.pathToHash. This is enough to pin the
// invariant without dragging in pebble/objstore wiring for an L1/L2
// end-to-end round-trip.
func TestVST_DeleteFileForAgent_ClearsPathToHash(t *testing.T) {
	v := New()
	const a AgentId = "delete-agent"
	const p = "doomed.txt"

	if err := v.WriteFileForAgent(a, p, []byte("payload")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := v.CommitForAgent(a, ""); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// After Commit, cur is empty but pathToHash holds the path -> hash entry.
	v.mu.RLock()
	s, ok := v.agentRO(a)
	if !ok {
		v.mu.RUnlock()
		t.Fatal("agent state missing post-commit")
	}
	if _, hasHash := s.pathToHash[p]; !hasHash {
		v.mu.RUnlock()
		t.Fatal("pathToHash missing entry post-commit (commit did not register path)")
	}
	v.mu.RUnlock()

	v.DeleteFileForAgent(a, p)

	// Both maps must be free of p after Delete; otherwise ReadFileForAgent
	// would fall through to pathToHash and return the L1/L2-resident value.
	v.mu.RLock()
	if _, stillInCur := s.cur[p]; stillInCur {
		t.Errorf("DeleteFileForAgent: cur[%q] not cleared", p)
	}
	if _, stillInPath := s.pathToHash[p]; stillInPath {
		t.Errorf("DeleteFileForAgent: pathToHash[%q] not cleared — deleted file would resurrect via L1/L2 fallback", p)
	}
	v.mu.RUnlock()

	// Externally observable: with no L1/L2 attached, Read returns nil
	// either way; this gives a defensive end-to-end check that the
	// fast path also reports the file as absent.
	got, err := v.ReadFileForAgent(a, p)
	if err != nil {
		t.Fatalf("post-delete read: %v", err)
	}
	if got != nil {
		t.Errorf("post-delete read: got=%q want=nil", got)
	}
}
