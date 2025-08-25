package objstore

import (
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/internal/util"
)

func TestGet_MissingOrCorruptDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// random hash
	h, _ := util.HashBlob([]byte("x"))
	// missing key
	if _, ok, err := s.Get(h); err != nil || ok {
		t.Fatalf("missing should be ok=false, err=nil: ok=%v err=%v", ok, err)
	}

	// corrupt path (simulate by PutBatch nil value preflight)
	err = s.PutBatch([]BatchEntry{{Hash: h, Value: nil}})
	if err == nil {
		t.Fatalf("expected error on nil value")
	}
}

func TestPutBatch_AllOrNothing(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "rocks"), nil)
	defer s.Close()

	h1, _ := util.HashBlob([]byte("a"))
	h2, _ := util.HashBlob([]byte("b"))

	err := s.PutBatch([]BatchEntry{
		{Hash: h1, Value: []byte("a")},
		{Hash: h2, Value: nil}, // force fail
	})
	if err == nil {
		t.Fatalf("want error on preflight")
	}
	// both should be absent
	if _, ok, _ := s.Get(h1); ok {
		t.Fatalf("atomicity violated: h1 found after failed batch")
	}
}

func TestConcurrentAccess_Safety(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Test concurrent writes don't corrupt store
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			data := []byte("data" + string(rune(id)))
			h, _ := util.HashBlob(data)
			_ = s.PutBatch([]BatchEntry{{Hash: h, Value: data}})
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify store is still functional
	testData := []byte("final")
	h, _ := util.HashBlob(testData)
	if err := s.PutBatch([]BatchEntry{{Hash: h, Value: testData}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Get(h)
	if err != nil || !ok || string(got) != "final" {
		t.Fatalf("store corrupted after concurrent access")
	}
}
