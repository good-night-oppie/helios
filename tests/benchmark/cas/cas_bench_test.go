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

// SPDX-License-Identifier: Apache-2.0

package cas_test

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/cas"
	"github.com/stretchr/testify/require"
)

// BenchmarkBLAKE3Store_Operations provides performance benchmarks
func BenchmarkBLAKE3Store_Operations(b *testing.B) {
	tempDir := b.TempDir()
	store, err := cas.NewBLAKE3Store(tempDir)
	require.NoError(b, err)
	defer store.Close()

	// Test data sizes typical for code files
	sizes := []int{100, 1000, 10000, 100000}
	
	for _, size := range sizes {
		content := make([]byte, size)
		rand.Read(content)
		
		b.Run(fmt.Sprintf("Store_%d_bytes", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := store.Store(content)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		
		// Store once for load benchmark
		hash, err := store.Store(content)
		require.NoError(b, err)
		
		b.Run(fmt.Sprintf("Load_%d_bytes", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := store.Load(hash)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		
		b.Run(fmt.Sprintf("Exists_%d_bytes", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				exists := store.Exists(hash)
				if !exists {
					b.Fatal("Hash should exist")
				}
			}
		})
	}
}

// BenchmarkBLAKE3Store_BatchOperations benchmarks batch performance for VST scenarios
func BenchmarkBLAKE3Store_BatchOperations(b *testing.B) {
	tempDir := b.TempDir()
	store, err := cas.NewBLAKE3Store(tempDir)
	require.NoError(b, err)
	defer store.Close()

	// Test various batch sizes typical for VST commits
	batchSizes := []int{10, 50, 100, 500}
	
	for _, batchSize := range batchSizes {
		// Generate test files
		files := make([][]byte, batchSize)
		for i := range files {
			files[i] = []byte(fmt.Sprintf(`package main

import "fmt"

func function%d() {
	fmt.Println("Function %d implementation")
	// Some typical code content...
	for i := 0; i < 10; i++ {
		fmt.Printf("Iteration: %%d\n", i)
	}
}
`, i, i))
		}
		
		b.Run(fmt.Sprintf("BatchStore_%d_files", batchSize), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := store.StoreBatch(files)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}