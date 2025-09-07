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
