package util

import (
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

func TestHashContent(t *testing.T) {
	tests := []struct {
		name      string
		content   []byte
		algorithm types.HashAlgorithm
		wantPref  string
		wantErr   bool
	}{
		{"blake3/hello", []byte("hello helios"), types.BLAKE3, "blake3:", false},
		{"blake3/empty", []byte{}, types.BLAKE3, "blake3:", false},
		{"sha256/hello", []byte("hello helios"), types.SHA256, "sha256:", false},
		{"unsupported", []byte("x"), types.HashAlgorithm("md5"), "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := HashContent(tc.content, tc.algorithm)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr {
				got := h.String()
				if len(got) == 0 || got[:len(tc.wantPref)] != tc.wantPref {
					t.Fatalf("hash prefix mismatch: got=%q want prefix=%q", got, tc.wantPref)
				}
			}
		})
	}
}
