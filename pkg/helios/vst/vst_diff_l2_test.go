package vst

import (
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
)

func TestDiffLoadsSnapshotsFromL2(t *testing.T) {
	v := New()
	dir := t.TempDir()
	store, err := objstore.Open(filepath.Join(dir, "db"), nil)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	v.AttachStores(nil, store)

	if err := v.WriteFile("a.txt", []byte("old")); err != nil {
		t.Fatalf("write: %v", err)
	}
	id1, _, err := v.Commit("c1")
	if err != nil {
		t.Fatalf("commit1: %v", err)
	}
	if err := v.WriteFile("a.txt", []byte("new")); err != nil {
		t.Fatalf("write2: %v", err)
	}
	id2, _, err := v.Commit("c2")
	if err != nil {
		t.Fatalf("commit2: %v", err)
	}
	delete(v.snaps, id1)

	diff, err := v.Diff(id1, id2)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if diff.Changed != 1 {
		t.Fatalf("expected 1 changed file, got %+v", diff)
	}
}
