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

package vst

import (
	"fmt"
	"testing"
)

// BenchmarkFork_WriteDiscard measures the cost of a propose-only loop:
// Fork → Write 1 small file → Discard. Acceptance target: ≤ 50μs on
// AMD EPYC class hardware. Use `go test -run=^$ -bench=BenchmarkFork_WriteDiscard
// -benchmem ./pkg/helios/vst/...` to validate.
func BenchmarkFork_WriteDiscard(b *testing.B) {
	v := New()
	if err := v.WriteFile("seed.txt", []byte("seed")); err != nil {
		b.Fatal(err)
	}
	id, _, err := v.Commit("seed")
	if err != nil {
		b.Fatal(err)
	}
	payload := []byte("payload")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := v.Fork(id)
		if err != nil {
			b.Fatal(err)
		}
		if err := f.Write("p.txt", payload); err != nil {
			b.Fatal(err)
		}
		f.Discard()
	}
}

// BenchmarkFork_MergeInto measures the validate-then-commit path:
// Fork → Write → MergeInto on a fresh branch each iter. Acceptance target:
// ≤ 100μs (excludes RocksDB flush; in-memory only here).
func BenchmarkFork_MergeInto(b *testing.B) {
	v := New()
	if err := v.WriteFile("seed.txt", []byte("seed")); err != nil {
		b.Fatal(err)
	}
	id, _, err := v.Commit("seed")
	if err != nil {
		b.Fatal(err)
	}
	payload := []byte("payload")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		branch := BranchID(fmt.Sprintf("b%d", i))
		if err := v.CreateBranch(branch, id); err != nil {
			b.Fatal(err)
		}
		f, err := v.Fork(id)
		if err != nil {
			b.Fatal(err)
		}
		if err := f.Write("p.txt", payload); err != nil {
			b.Fatal(err)
		}
		if _, err := f.MergeInto(branch); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFork_ReadPassthrough measures overlay-miss → base passthrough.
func BenchmarkFork_ReadPassthrough(b *testing.B) {
	v := New()
	for i := 0; i < 64; i++ {
		_ = v.WriteFile(fmt.Sprintf("f%02d.txt", i), []byte("content"))
	}
	id, _, err := v.Commit("seed")
	if err != nil {
		b.Fatal(err)
	}
	f, err := v.Fork(id)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Discard()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := f.Read("f00.txt"); err != nil {
			b.Fatal(err)
		}
	}
}
