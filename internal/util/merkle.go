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
