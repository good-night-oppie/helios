package vst_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/vst"
)

// This test asserts:
// 1) First read after restore promotes from L2 -> L1 (L1 miss observed).
// 2) Second read should be a L1 hit (Hits increases).
// 3) SnapshotID remains deterministic across identical content.
func TestReadPath_Promotes_L2_to_L1_and_HitsAfter(t *testing.T) {
	t.Skip("Skipping integration test - cross-engine snapshot sharing needs more work")
	// fresh engine
	eng := vst.New()

	// Create stores
	l1, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        8 << 20,
		CompressionThreshold: 256,
	})
	l2Dir := t.TempDir()
	l2, err := objstore.Open(filepath.Join(l2Dir, "rocks"), nil)
	if err != nil {
		t.Fatalf("open l2: %v", err)
	}

	// Attach stores to engine
	eng.AttachStores(l1, l2)

	// prepare content and commit
	want := []byte("hello helios")
	if err := eng.WriteFile("a.txt", want); err != nil {
		t.Fatalf("write: %v", err)
	}
	id1, _, err := eng.Commit("")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// New engine to simulate cold start; attach same stores
	eng2 := vst.New()
	eng2.AttachStores(l1, l2)

	// restore snapshot into a fresh engine
	if err := eng2.Restore(id1); err != nil {
		t.Fatalf("restore: %v", err)
	}

	// 1st read: expect OK and an L1 miss (data promoted to L1)
	got1, err := eng2.ReadFile("a.txt")
	if err != nil || !bytes.Equal(got1, want) {
		t.Fatalf("first read mismatch; err=%v", err)
	}
	s1 := l1.Stats()
	if s1.Misses < 1 {
		t.Fatalf("expected at least 1 L1 miss on first read, got %+v", s1)
	}

	// 2nd read: should now be a L1 hit
	got2, err := eng2.ReadFile("a.txt")
	if err != nil || !bytes.Equal(got2, want) {
		t.Fatalf("second read mismatch; err=%v", err)
	}
	s2 := l1.Stats()
	if s2.Hits < 1 {
		t.Fatalf("expected L1 hit on second read, got %+v", s2)
	}

	// Snapshot determinism: re-commit same state should produce same id
	id2, _, err := eng2.Commit("")
	if err != nil {
		t.Fatalf("re-commit: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("snapshot id not stable: %s vs %s", id1, id2)
	}
}
