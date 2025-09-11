package vst

import (
	"fmt"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

func BenchmarkBuildDirectoryTreeOptimized(b *testing.B) {
	v := New()
	files := make(map[string]types.Hash)
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("dir%d/file%d.txt", i, i)
		files[path] = types.Hash{Algorithm: types.BLAKE3, Digest: []byte{byte(i)}}
	}
	for i := 0; i < b.N; i++ {
		v.buildDirectoryTreeOptimized(files)
	}
}

func TestBuildDirectoryTreeOptimizedLarge(t *testing.T) {
	v := New()
	files := make(map[string]types.Hash)
	for i := 0; i < 2000; i++ {
		path := fmt.Sprintf("dir%d/file.txt", i)
		files[path] = types.Hash{Algorithm: types.BLAKE3, Digest: []byte{byte(i % 256)}}
	}
	if _, err := v.buildDirectoryTreeOptimized(files); err != nil {
		t.Fatalf("buildDirectoryTreeOptimized: %v", err)
	}
}
