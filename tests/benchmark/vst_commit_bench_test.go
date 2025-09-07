// Copyright 2025 Oppie Thunder Contributors
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

package benchmark

import (
	"bytes"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/good-night-oppie/helios-engine/pkg/helios/vst"
)

// BenchmarkCommitPerformance tests VST commit performance across different scenarios
func BenchmarkCommitPerformance(b *testing.B) {
	scenarios := []struct {
		name      string
		files     int
		fileSize  int
		target    time.Duration
	}{
		{"1File_1KB", 1, 1024, 20 * time.Microsecond},
		{"10Files_1KB", 10, 1024, 50 * time.Microsecond},
		{"100Files_1KB", 100, 1024, 70 * time.Microsecond},   // Critical target
		{"1000Files_1KB", 1000, 1024, 100 * time.Microsecond}, // Stress test
	}

	for _, scenario := range scenarios {
		b.Run("Original_"+scenario.name, func(b *testing.B) {
			benchmarkCommitMethod(b, scenario.files, scenario.fileSize, scenario.target, false)
		})
		
		b.Run("Optimized_"+scenario.name, func(b *testing.B) {
			benchmarkCommitMethod(b, scenario.files, scenario.fileSize, scenario.target, true)
		})
	}
}

func benchmarkCommitMethod(b *testing.B, numFiles, fileSize int, target time.Duration, useOptimized bool) {
	eng := vst.New()

	// Pre-populate with scenario data
	payload := make([]byte, fileSize)
	for i := 0; i < len(payload); i++ {
		payload[i] = byte('A' + (i % 26))
	}

	for i := 0; i < numFiles; i++ {
		path := "perf/" + strconv.Itoa(i/100) + "/file_" + strconv.Itoa(i) + ".dat"
		err := eng.WriteFile(path, payload)
		if err != nil {
			b.Fatalf("WriteFile failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	var totalLatency time.Duration
	for i := 0; i < b.N; i++ {
		// Modify one file for this iteration
		modPath := "perf/" + strconv.Itoa((i%numFiles)/100) + "/file_" + strconv.Itoa(i%numFiles) + ".dat"
		newPayload := append(payload, byte('0'+i%10))
		err := eng.WriteFile(modPath, newPayload)
		if err != nil {
			b.Fatalf("WriteFile modification failed: %v", err)
		}

		var start time.Time
		var commitLatency time.Duration

		if useOptimized {
			start = time.Now()
			_, _, err = eng.CommitOptimized("optimized-commit-" + strconv.Itoa(i))
			commitLatency = time.Since(start)
		} else {
			start = time.Now()
			_, _, err = eng.Commit("original-commit-" + strconv.Itoa(i))
			commitLatency = time.Since(start)
		}
		
		if err != nil {
			b.Fatalf("Commit failed: %v", err)
		}

		totalLatency += commitLatency
	}

	avgLatency := totalLatency / time.Duration(b.N)
	methodName := "Original"
	if useOptimized {
		methodName = "Optimized"
	}
	
	b.ReportMetric(float64(avgLatency.Nanoseconds())/1000, "μs/commit")
	
	if avgLatency > target {
		b.Logf("%s PERFORMANCE: avg=%v vs target=%v (%.1fx slower)", 
			methodName, avgLatency, target, float64(avgLatency)/float64(target))
	} else {
		b.Logf("%s PERFORMANCE SUCCESS: avg=%v under target=%v (%.1fx faster than target)", 
			methodName, avgLatency, target, float64(target)/float64(avgLatency))
	}
}

// BenchmarkCommitAndRead measures Commit() and ReadFile() together
func BenchmarkCommitAndRead(b *testing.B) {
	eng := vst.New()
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

// BenchmarkMaterialize tests snapshot materialization performance
func BenchmarkMaterialize(b *testing.B) {
	scenarios := []struct {
		name     string
		files    int
		fileSize int
	}{
		{"Small_50x512B", 50, 512},
		{"Medium_200x1KB", 200, 1024},
		{"Large_1000x1KB", 1000, 1024},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			eng := vst.New()
			
			// Generate files
			for i := 0; i < scenario.files; i++ {
				buf := bytes.Repeat([]byte("A"), scenario.fileSize)
				_ = eng.WriteFile("mat/"+strconv.Itoa(i/100)+"/file_"+strconv.Itoa(i), buf)
			}
			snapID, _, _ := eng.Commit("benchmark snapshot")
			out := b.TempDir()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Materialize full tree
				_, err := eng.Materialize(snapID, out, types.MatOpts{})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryEfficiency tests memory usage during commits
func BenchmarkMemoryEfficiency(b *testing.B) {
	scenarios := []struct {
		name      string
		files     int
		fileSize  int
		useOptimized bool
	}{
		{"Original_1000Files", 1000, 1024, false},
		{"Optimized_1000Files", 1000, 1024, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			eng := vst.New()

			// Pre-populate with larger dataset
			for i := 0; i < scenario.files; i++ {
				// Create files with varied sizes
				size := scenario.fileSize + (i * 10) // Variable size
				payload := make([]byte, size)
				for j := range payload {
					payload[j] = byte('A' + (j % 26))
				}
				
				path := "mem_test/" + strconv.Itoa(i/100) + "/file_" + strconv.Itoa(i) + ".dat"
				err := eng.WriteFile(path, payload)
				if err != nil {
					b.Fatalf("WriteFile failed: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Modify a random subset of files
				for j := 0; j < 10; j++ {
					idx := (i*10 + j) % scenario.files
					path := "mem_test/" + strconv.Itoa(idx/100) + "/file_" + strconv.Itoa(idx) + ".dat"
					newPayload := []byte("modified-content-" + strconv.Itoa(i) + "-" + strconv.Itoa(j))
					err := eng.WriteFile(path, newPayload)
					if err != nil {
						b.Fatalf("WriteFile failed: %v", err)
					}
				}

				if scenario.useOptimized {
					_, _, err := eng.CommitOptimized("memory-efficiency-test-" + strconv.Itoa(i))
					if err != nil {
						b.Fatalf("CommitOptimized failed: %v", err)
					}
				} else {
					_, _, err := eng.Commit("memory-efficiency-test-" + strconv.Itoa(i))
					if err != nil {
						b.Fatalf("Commit failed: %v", err)
					}
				}
			}
		})
	}
}