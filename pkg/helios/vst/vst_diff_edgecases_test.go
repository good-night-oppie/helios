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

// Focus: add-only, delete-only, rename (= delete+add), binary content.
func TestVST_Diff_EdgeCases(t *testing.T) {
	v := New()

	// Base snapshot with two files
	_ = v.WriteFile("a.txt", []byte("A"))
	_ = v.WriteFile("b.bin", []byte{0x00, 0x01})
	id1, _, err := v.Commit("base")
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	// Case 1: add-only
	_ = v.WriteFile("c.txt", []byte("C"))
	id2, _, _ := v.Commit("add-only")
	diff, err := v.Diff(id1, id2)
	if err != nil {
		t.Fatalf("diff add-only: %v", err)
	}
	if diff.Added < 1 || diff.Changed != 0 || diff.Deleted != 0 {
		t.Fatalf("want Added>=1, Changed=0, Deleted=0, got %+v", diff)
	}

	// Case 2: delete-only
	if err := v.Restore(id1); err != nil {
		t.Fatalf("restore: %v", err)
	}
	v.DeleteFile("b.bin") // delete b.bin, keep only a.txt
	id3, _, _ := v.Commit("delete-only")
	diff, err = v.Diff(id1, id3)
	if err != nil {
		t.Fatalf("diff delete-only: %v", err)
	}
	if diff.Deleted < 1 || diff.Changed != 0 {
		t.Fatalf("want Deleted>=1, Changed=0, got %+v", diff)
	}

	// Case 3: rename simulated as delete+add
	if err := v.Restore(id1); err != nil {
		t.Fatalf("restore: %v", err)
	}
	// "rename": remove a.txt and add a_renamed.txt with same content
	v.DeleteFile("a.txt")
	_ = v.WriteFile("a_renamed.txt", []byte("A"))
	// b.bin stays the same
	id4, _, _ := v.Commit("rename-sim")
	diff, err = v.Diff(id1, id4)
	if err != nil {
		t.Fatalf("diff rename-sim: %v", err)
	}
	if diff.Added < 1 || diff.Deleted < 1 {
		t.Fatalf("want Added>=1 and Deleted>=1 for rename-sim, got %+v", diff)
	}

	// Binary change should be counted as Changed
	if err := v.Restore(id1); err != nil {
		t.Fatalf("restore: %v", err)
	}
	_ = v.WriteFile("b.bin", []byte{0x00, 0xFF})
	id5, _, _ := v.Commit("binary-change")
	diff, err = v.Diff(id1, id5)
	if err != nil {
		t.Fatalf("diff binary-change: %v", err)
	}
	if diff.Changed < 1 {
		t.Fatalf("want Changed>=1 for binary-change, got %+v", diff)
	}
}
