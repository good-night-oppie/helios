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
	"testing"
)

// RED: committing identical content twice should yield identical root-hash SnapshotIDs
// once VST uses content-addressed objects (Blob/Tree/Commit).
func TestVST_ObjectizedCommit_IsDeterministic(t *testing.T) {
	v := New()

	_ = v.WriteFile("a.txt", []byte("A"))
	_ = v.WriteFile("b.txt", []byte("B"))
	id1, _, err := v.Commit("first")
	if err != nil {
		t.Fatalf("commit 1: %v", err)
	}

	// Restore and rewrite same bytes in different order – same semantic state.
	if err := v.Restore(id1); err != nil {
		t.Fatalf("restore: %v", err)
	}
	_ = v.WriteFile("b.txt", []byte("B"))
	_ = v.WriteFile("a.txt", []byte("A"))
	id2, _, err := v.Commit("second")
	if err != nil {
		t.Fatalf("commit 2: %v", err)
	}

	// RED now: SnapshotIDs should be stable and equal once we switch to content-addressed root.
	if id1 != id2 {
		t.Fatalf("want deterministic SnapshotID (root hash), got %s vs %s", id1, id2)
	}
}

// RED: changing one file must change the root hash, but leave others shared (implicit).
func TestVST_ObjectizedCommit_ChangesAffectRootHash(t *testing.T) {
	v := New()

	_ = v.WriteFile("a.txt", []byte("A"))
	_ = v.WriteFile("b.txt", []byte("B"))
	id1, _, err := v.Commit("base")
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	// Change a single file
	_ = v.WriteFile("b.txt", []byte("B2"))
	id2, _, err := v.Commit("delta")
	if err != nil {
		t.Fatalf("commit delta: %v", err)
	}

	if id1 == id2 {
		t.Fatalf("root hash should change when content changes; got id1==id2 %s", id1)
	}
}
