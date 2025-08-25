package vst

import (
	"os"
	"path/filepath"
	"testing"

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
