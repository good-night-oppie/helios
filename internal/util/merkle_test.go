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
)

func TestHashBlob(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{"empty", []byte{}},
		{"hello", []byte("hello world")},
		{"binary", []byte{0x00, 0xFF, 0x42}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := HashBlob(tc.content)
			if err != nil {
				t.Fatalf("HashBlob error: %v", err)
			}
			if len(h.Digest) == 0 {
				t.Fatal("expected non-empty digest")
			}
			// Verify it's deterministic
			h2, err := HashBlob(tc.content)
			if err != nil {
				t.Fatalf("HashBlob error on second call: %v", err)
			}
			if h.String() != h2.String() {
				t.Fatalf("HashBlob not deterministic: %s != %s", h.String(), h2.String())
			}
		})
	}
}

func TestHashTree(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
	}{
		{"empty", nil},
		{"single", []string{"file.txt:blob:abc123"}},
		{"multiple", []string{"a.txt:blob:111", "b.txt:blob:222", "dir:tree:333"}},
		{"unsorted", []string{"z:blob:999", "a:blob:111", "m:blob:555"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := HashTree(tc.entries)
			if err != nil {
				t.Fatalf("HashTree error: %v", err)
			}
			if len(h.Digest) == 0 {
				t.Fatal("expected non-empty digest")
			}
			// Verify it's deterministic even with different order
			if tc.name == "unsorted" {
				// Test that different input order produces same hash
				reordered := []string{"a:blob:111", "m:blob:555", "z:blob:999"}
				h2, err := HashTree(reordered)
				if err != nil {
					t.Fatalf("HashTree error on reordered: %v", err)
				}
				if h.String() != h2.String() {
					t.Fatalf("HashTree not order-independent: %s != %s", h.String(), h2.String())
				}
			}
		})
	}
}
