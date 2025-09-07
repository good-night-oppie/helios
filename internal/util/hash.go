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
