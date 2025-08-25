package types

import "testing"

func TestSnapshotID(t *testing.T) {
	id := SnapshotID("test-id")
	if string(id) != "test-id" {
		t.Errorf("got %s, want test-id", id)
	}
}

func TestHashString_NotEmpty(t *testing.T) {
	h := Hash{Algorithm: BLAKE3, Digest: []byte{0x01, 0x02}}
	if s := h.String(); s == "" {
		t.Fatal("expected non-empty string")
	}
}
