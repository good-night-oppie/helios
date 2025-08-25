package objstore_test

import (
	"bytes"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/internal/util"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

func hOf(t *testing.T, b []byte) types.Hash {
	t.Helper()
	h, err := util.HashContent(b, types.BLAKE3)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}

func TestPutBatch_Atomicity_PreflightFail(t *testing.T) {
	dir := t.TempDir()
	db, err := objstore.Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	a := []byte("alpha")
	b := []byte("beta")
	ha := hOf(t, a)
	hb := hOf(t, b)

	// One value is nil → preflight must fail and nothing is written.
	err = db.PutBatch([]objstore.BatchEntry{
		{Hash: ha, Value: a},
		{Hash: hb, Value: nil},
	})
	if err == nil {
		t.Fatalf("expected error on nil value")
	}

	if _, ok, _ := db.Get(ha); ok {
		t.Fatalf("atomicity violated: ha should not exist after failed batch")
	}
	if _, ok, _ := db.Get(hb); ok {
		t.Fatalf("atomicity violated: hb should not exist after failed batch")
	}
}

func TestPutGet_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	db, err := objstore.Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	payload := []byte("roundtrip")
	h := hOf(t, payload)

	if err := db.PutBatch([]objstore.BatchEntry{{Hash: h, Value: payload}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := db.Get(h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch")
	}
}

func TestLargePayload(t *testing.T) {
	dir := t.TempDir()
	db, err := objstore.Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	raw := make([]byte, 5<<20) // 5 MiB
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	h := hOf(t, raw)

	if err := db.PutBatch([]objstore.BatchEntry{{Hash: h, Value: raw}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := db.Get(h)
	if err != nil || !ok {
		t.Fatalf("expected ok=true, err=nil, got ok=%v err=%v", ok, err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("large payload mismatch")
	}
}

func TestMissingKey(t *testing.T) {
	dir := t.TempDir()
	db, err := objstore.Open(filepath.Join(dir, "rocks"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	missing := hOf(t, []byte("missing"))
	_, ok, err := db.Get(missing)
	if err != nil {
		t.Fatalf("expected err=nil")
	}
	if ok {
		t.Fatalf("expected ok=false for missing key")
	}
}
