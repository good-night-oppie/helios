package vst

import (
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
)

func TestDeleteFileRemovesPathToHash(t *testing.T) {
	v := New()
	dir := t.TempDir()
	store, err := objstore.Open(filepath.Join(dir, "db"), nil)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	v.AttachStores(nil, store)

	if err := v.WriteFile("foo.txt", []byte("hi")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := v.Commit("msg"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	v.DeleteFile("foo.txt")
	if data, err := v.ReadFile("foo.txt"); err != nil || data != nil {
		t.Fatalf("expected nil after delete, got %v err %v", data, err)
	}
}
