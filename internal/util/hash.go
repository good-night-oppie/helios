package util

import (
	"crypto/sha256"
	"errors"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"lukechampine.com/blake3"
)

// HashContent computes content hash with the given algorithm.
func HashContent(content []byte, algorithm types.HashAlgorithm) (types.Hash, error) {
	switch algorithm {
	case types.BLAKE3:
		sum := blake3.Sum256(content)
		return types.Hash{Algorithm: types.BLAKE3, Digest: sum[:]}, nil
	case types.SHA256:
		sum := sha256.Sum256(content)
		return types.Hash{Algorithm: types.SHA256, Digest: sum[:]}, nil
	default:
		return types.Hash{}, errors.New("unsupported hash algorithm")
	}
}
