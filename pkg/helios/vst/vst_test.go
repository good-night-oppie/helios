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
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// TestVST_CommitRestoreDiffMaterialize checks the basic lifecycle of VST.
// At this stage VST is not implemented yet, so compilation will fail (RED phase).
func TestVST_CommitRestoreDiffMaterialize(t *testing.T) {
	v := New() // not implemented yet

	// Write two files and do the first commit
	_ = v.WriteFile("hello.txt", []byte("hi"))
	_ = v.WriteFile("dir/a.txt", []byte("A"))
	id1, m1, err := v.Commit("init")
	if err != nil || id1 == "" {
		t.Fatalf("commit1 err=%v id=%s", err, id1)
	}
	if m1.NewObjects == 0 {
		t.Fatalf("expect new objects on first commit")
	}

	// Modify hello.txt and commit again
	_ = v.WriteFile("hello.txt", []byte("hello"))
	id2, m2, err := v.Commit("update")
	if err != nil || id2 == "" || id2 == id1 {
		t.Fatalf("commit2 issue: err=%v id1=%s id2=%s", err, id1, id2)
	}
	if m2.NewObjects == 0 {
		t.Fatalf("expect some new objects on second commit")
	}

	// Diff should detect one changed file
	diff, err := v.Diff(id1, id2)
	if err != nil {
		t.Fatalf("diff err=%v", err)
	}
	if diff.Changed < 1 {
		t.Fatalf("want Changed>=1, got %+v", diff)
	}

	// Restore to id1 and check that hello.txt content goes back to "hi"
	if err := v.Restore(id1); err != nil {
		t.Fatalf("restore err=%v", err)
	}
	got, _ := v.ReadFile("hello.txt")
	if string(got) != "hi" {
		t.Fatalf("after restore want 'hi', got %q", string(got))
	}

	// Materialize snapshot id2 into a temp dir and check file content
	out, _ := os.MkdirTemp("", "helios-mat-*")
	defer os.RemoveAll(out)
	if _, err := v.Materialize(id2, out, types.MatOpts{}); err != nil {
		t.Fatalf("materialize err=%v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "hello.txt"))
	if err != nil {
		t.Fatalf("read materialized file: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("materialized content want 'hello', got %q", string(b))
	}
}

func TestVST_L1Stats(t *testing.T) {
	v := New()

	// Test with no L1 attached
	stats := v.L1Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("expected zero stats with no L1, got %+v", stats)
	}
}

func TestVST_ReadFileNotFound(t *testing.T) {
	v := New()
	data, err := v.ReadFile("nonexistent.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil data for non-existent file, got %v", data)
	}
}

func TestVST_DeleteFile(t *testing.T) {
	v := New()
	_ = v.WriteFile("to_be_deleted.txt", []byte("delete me"))
	v.DeleteFile("to_be_deleted.txt")
	data, err := v.ReadFile("to_be_deleted.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil data for deleted file, got %v", data)
	}
}

func TestMatchGlob_SpecialPatterns(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"src/main.go", "src/**", true},
		{"src/deep/nested/file.go", "src/**", true},
		{"other/file.go", "src/**", false},
		{"test.go", "*.go", true},
		{"test.txt", "*.go", false},
		{"prefix_file", "prefix**", true},
		{"other_file", "prefix**", false},
	}

	for _, tt := range tests {
		got := matchGlob(tt.path, tt.pattern)
		if got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestVST_ReadFile_WithCache(t *testing.T) {
	// Setup VST with L1 and L2 stores
	v := New()
	l1, err := l1cache.New(l1cache.Config{CapacityBytes: 1024})
	if err != nil {
		t.Fatalf("failed to create l1 cache: %v", err)
	}
	l2dir := t.TempDir()
	l2, err := objstore.Open(filepath.Join(l2dir, "obj"), nil)
	if err != nil {
		t.Fatalf("failed to create l2 store: %v", err)
	}
	defer l2.Close()
	v.AttachStores(l1, l2)

	// Test data
	filePath := "cached_file.txt"
	fileContent := []byte("this content will be cached")
	fileHash, err := util.HashBlob(fileContent)
	if err != nil {
		t.Fatalf("failed to hash content: %v", err)
	}

	// Manually set the path->hash mapping to simulate a state where the file
	// is not in the working set (`v.cur`) but is referenced.
	v.pathToHash[filePath] = fileHash

	// Scenario 1: L1 miss, L2 hit, then L1 hit
	// 1a. Put data in L2 only
	if err := l2.PutBatch([]objstore.BatchEntry{{Hash: fileHash, Value: fileContent}}); err != nil {
		t.Fatalf("failed to put data in L2: %v", err)
	}

	// 1b. Read the file - should be an L1 miss and L2 hit
	data, err := v.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile (L2 hit) failed: %v", err)
	}
	if !bytes.Equal(data, fileContent) {
		t.Fatalf("ReadFile (L2 hit) returned wrong data: got %q, want %q", data, fileContent)
	}

	// 1c. Check L1 stats for one miss
	stats := v.L1Stats()
	if stats.Hits != 0 || stats.Misses != 1 {
		t.Errorf("Expected 0 hits and 1 miss, got %d hits and %d misses", stats.Hits, stats.Misses)
	}

	// 1d. Read the file again - should be an L1 hit now
	data, err = v.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile (L1 hit) failed: %v", err)
	}
	if !bytes.Equal(data, fileContent) {
		t.Fatalf("ReadFile (L1 hit) returned wrong data: got %q, want %q", data, fileContent)
	}

	// 1e. Check L1 stats for one hit and one miss
	stats = v.L1Stats()
	if stats.Hits != 1 || stats.Misses != 1 {
		t.Errorf("Expected 1 hit and 1 miss, got %d hits and %d misses", stats.Hits, stats.Misses)
	}

	// Scenario 2: File not in L1 or L2 (dangling reference)
	otherPath := "other_file.txt"
	otherContent := []byte("other content")
	otherHash, err := util.HashBlob(otherContent)
	if err != nil {
		t.Fatalf("failed to hash other content: %v", err)
	}
	v.pathToHash[otherPath] = otherHash // path is known, but hash is not in stores

	data, err = v.ReadFile(otherPath)
	if err != nil {
		t.Fatalf("ReadFile (dangling) failed: %v", err)
	}
	if data != nil {
		t.Fatalf("ReadFile (dangling) should return nil data, got: %q", data)
	}
}

func TestVST_CommitRestore_WithL2(t *testing.T) {
	// Setup VST with L1 and L2 stores
	l1, err := l1cache.New(l1cache.Config{CapacityBytes: 1024})
	if err != nil {
		t.Fatalf("failed to create l1 cache: %v", err)
	}
	l2dir := t.TempDir()
	l2, err := objstore.Open(filepath.Join(l2dir, "obj"), nil)
	if err != nil {
		t.Fatalf("failed to create l2 store: %v", err)
	}
	defer l2.Close()

	// Create and configure the first VST instance
	v1 := New()
	v1.AttachStores(l1, l2)

	// Write a file and commit
	filePath := "persistent_file.txt"
	fileContent := []byte("this content should persist in L2")
	_ = v1.WriteFile(filePath, fileContent)
	snapID, _, err := v1.Commit("first commit")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Create a new VST instance with the same L2 store but no in-memory snapshots
	v2 := New()
	v2.AttachStores(nil, l2) // No L1 for simplicity, just testing L2 restore

	// Restore the snapshot in the new VST instance
	if err := v2.Restore(snapID); err != nil {
		t.Fatalf("Restore from L2 failed: %v", err)
	}

	// Verify the file is restored correctly
	data, err := v2.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile after L2 restore failed: %v", err)
	}
	if !bytes.Equal(data, fileContent) {
		t.Fatalf("Restored data mismatch: got %q, want %q", data, fileContent)
	}
}

func TestVST_EngineMetricsSnapshot_Nil(t *testing.T) {
	v := New()
	v.em = nil // Manually set to nil to test robustness
	snapshot := v.EngineMetricsSnapshot()
	if snapshot.NewObjects != 0 {
		t.Errorf("expected 0 new objects, got %d", snapshot.NewObjects)
	}
}

func TestDepth(t *testing.T) {
	testCases := []struct {
		path string
		want int
	}{
		{".", 0},
		{"/", 0},
		{"", 0},
		{"a", 0},
		{"a/b", 1},
		{"a/b/c", 2},
		{"a//b", 1},
	}

	for _, tc := range testCases {
		got := depth(tc.path)
		if got != tc.want {
			t.Errorf("depth(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}
