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
	"sort"
	"strings"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// HashBlob computes the content-addressed hash of a file blob.
func HashBlob(content []byte) (types.Hash, error) {
	return HashContent(content, types.BLAKE3)
}

// HashTree computes a deterministic Merkle hash for a directory.
// entries: list of "name:type:hexChildHash" (already stable & normalized).
// We hash the joined string to get the tree hash.
func HashTree(entries []string) (types.Hash, error) {
	sort.Strings(entries)                            // deterministic order
	joined := strings.Join(entries, "\n")            // stable join
	return HashContent([]byte(joined), types.BLAKE3) // merkle over entries
}
