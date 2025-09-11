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


package util

import (
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/types"
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
