package vst

import (
	"bytes"
	"math/rand"
	"strconv"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// Benchmark: measure Commit() and ReadFile() fast paths.
// NOTE: This is a micro-bench; it won't verify correctness, just timing.
func BenchmarkCommitAndRead(b *testing.B) {
	eng := New() // use your ctor
	// Preload working-set with some files
	for i := 0; i < 100; i++ {
		_ = eng.WriteFile("file_"+strconv.Itoa(i)+".txt", []byte("seed"))
	}
	_, _, _ = eng.Commit("initial commit")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Write one file + commit
		key := "file_" + strconv.Itoa(rand.Intn(100)) + ".txt"
		_ = eng.WriteFile(key, []byte("payload-"+strconv.Itoa(i)))
		_, _, _ = eng.Commit("bench commit")

		// Read a hot path
		_, _ = eng.ReadFile(key)

		// Read a cold-ish path
		_, _ = eng.ReadFile("file_" + strconv.Itoa((i+17)%100) + ".txt")
	}
}

func BenchmarkMaterializeSmall(b *testing.B) {
	eng := New()
	// generate small files
	for i := 0; i < 50; i++ {
		buf := bytes.Repeat([]byte("A"), 512)
		_ = eng.WriteFile("sm/"+strconv.Itoa(i), buf)
	}
	snapID, _, _ := eng.Commit("benchmark snapshot")
	out := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Materialize full tree (small)
		_, err := eng.Materialize(snapID, out, types.MatOpts{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
