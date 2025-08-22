package types

import "fmt"

// HashAlgorithm defines supported hash algorithms
type HashAlgorithm string

const (
	BLAKE3 HashAlgorithm = "blake3"
	SHA256 HashAlgorithm = "sha256"
)

// Hash represents a content-addressable identifier
type Hash struct {
	Algorithm HashAlgorithm
	Digest    []byte
}

func (h Hash) String() string {
	return fmt.Sprintf("%s:%x", h.Algorithm, h.Digest)
}
