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


package stress

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/good-night-oppie/helios/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
	"github.com/good-night-oppie/helios/pkg/helios/vst"
)

// BenchmarkAlphaGoWorkload simulates AlphaGo-level MCTS workload
// Target: 1,600 simulations per move, 800-1000 parallel trees
func BenchmarkAlphaGoWorkload(b *testing.B) {
	// Setup L1 and L2 stores
	l1, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        256 << 20, // 256MB L1 cache
		CompressionThreshold: 1024,
	})
	
	l2Dir := b.TempDir()
	l2, _ := objstore.Open(l2Dir+"/rocks", nil)
	defer l2.Close()
	
	eng := vst.New()
	eng.AttachStores(l1, l2)
	
	const (
		SimulationsPerMove = 1600
		BranchingFactor    = 250  // Go board positions
		StateSize          = 361  // 19x19 board
		ParallelTrees      = 100  // Scale for demo
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		treeID := rand.Intn(ParallelTrees)
		for pb.Next() {
			// Simulate MCTS expansion phase
			for sim := 0; sim < SimulationsPerMove; sim++ {
				// Generate random board state
				state := make([]byte, StateSize)
				rand.Read(state)
				
				// Write state to tree
				path := fmt.Sprintf("tree_%d/sim_%d/board.dat", treeID, sim)
				eng.WriteFile(path, state)
				
				// Commit every 100 simulations (batch optimization)
				if sim%100 == 0 {
					eng.Commit(fmt.Sprintf("MCTS expansion %d", sim))
				}
			}
		}
	})
	
	// Report metrics
	b.ReportMetric(float64(b.N*SimulationsPerMove), "simulations")
	b.ReportMetric(float64(b.N*SimulationsPerMove)/b.Elapsed().Seconds(), "sims/sec")
}

// BenchmarkMuZeroDynamics tests MuZero-style hidden state dynamics
// Target: 800 simulations with learned model predictions
func BenchmarkMuZeroDynamics(b *testing.B) {
	l1, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        128 << 20, // 128MB
		CompressionThreshold: 256,
	})
	
	eng := vst.New()
	eng.AttachStores(l1, nil) // L1-only for speed
	
	const (
		HiddenStateSize = 256  // Dimensions
		LookaheadDepth  = 50   // Planning horizon
		ParallelEnvs    = 128  // Concurrent environments
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		envID := rand.Intn(ParallelEnvs)
		for pb.Next() {
			// Simulate dynamics rollout
			for step := 0; step < LookaheadDepth; step++ {
				// Generate hidden state representation
				hiddenState := make([]byte, HiddenStateSize*4) // float32
				rand.Read(hiddenState)
				
				path := fmt.Sprintf("env_%d/step_%d/hidden.bin", envID, step)
				eng.WriteFile(path, hiddenState)
			}
			
			// Snapshot for backpropagation
			eng.Commit(fmt.Sprintf("rollout_env_%d", envID))
		}
	})
	
	b.ReportMetric(float64(b.N*LookaheadDepth*ParallelEnvs), "state_updates")
}

// TestConcurrentMCTS tests massive parallel tree operations
// Target: 1,000+ concurrent MCTS agents, zero lock contention
func TestConcurrentMCTS(t *testing.T) {
	const (
		NumTrees        = 1000
		SimsPerTree     = 100
		TargetOpsPerSec = 10000
	)
	
	// Create shared storage
	l1, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        512 << 20, // 512MB
		CompressionThreshold: 512,
	})
	
	l2Dir := t.TempDir()
	l2, _ := objstore.Open(l2Dir+"/rocks", nil)
	defer l2.Close()
	
	// Metrics tracking
	var totalOps int64
	var totalLatency int64
	latencies := make([]int64, 0, NumTrees*SimsPerTree)
	var mu sync.Mutex
	
	start := time.Now()
	var wg sync.WaitGroup
	
	// Launch concurrent MCTS trees
	for i := 0; i < NumTrees; i++ {
		wg.Add(1)
		go func(treeID int) {
			defer wg.Done()
			
			// Each tree gets its own VST instance
			eng := vst.New()
			eng.AttachStores(l1, l2)
			
			for sim := 0; sim < SimsPerTree; sim++ {
				opStart := time.Now()
				
				// Simulate tree operation
				state := make([]byte, 1024)
				rand.Read(state)
				
				path := fmt.Sprintf("tree_%d/node_%d.dat", treeID, sim)
				eng.WriteFile(path, state)
				
				if sim%10 == 0 {
					eng.Commit(fmt.Sprintf("tree_%d_checkpoint", treeID))
				}
				
				// Track metrics
				latency := time.Since(opStart).Microseconds()
				atomic.AddInt64(&totalOps, 1)
				atomic.AddInt64(&totalLatency, latency)
				
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	elapsed := time.Since(start)
	
	// Calculate metrics
	opsPerSec := float64(totalOps) / elapsed.Seconds()
	avgLatency := totalLatency / totalOps
	
	// Sort for percentiles
	sortLatencies(latencies)
	p50 := latencies[len(latencies)/2]
	p99 := latencies[len(latencies)*99/100]
	
	// Report results
	t.Logf("=== MCTS Stress Test Results ===")
	t.Logf("Trees: %d", NumTrees)
	t.Logf("Total Operations: %d", totalOps)
	t.Logf("Duration: %v", elapsed)
	t.Logf("Throughput: %.0f ops/sec", opsPerSec)
	t.Logf("Avg Latency: %dμs", avgLatency)
	t.Logf("P50 Latency: %dμs", p50)
	t.Logf("P99 Latency: %dμs", p99)
	
	// Check performance targets
	if opsPerSec < TargetOpsPerSec {
		t.Errorf("Throughput %.0f ops/sec below target %d ops/sec", 
			opsPerSec, TargetOpsPerSec)
	}
	
	if p50 > 100 {
		t.Errorf("P50 latency %dμs exceeds target 100μs", p50)
	}
	
	// Check cache hit rate
	stats := l1.Stats()
	hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	t.Logf("L1 Cache Hit Rate: %.2f%%", hitRate*100)
	
	if hitRate < 0.90 {
		t.Errorf("Cache hit rate %.2f%% below target 90%%", hitRate*100)
	}
}

// TestTimeTravelChess demonstrates instant state manipulation
// Target: <1μs to jump to any position in game history
func TestTimeTravelChess(t *testing.T) {
	eng := vst.New()
	l1, _ := l1cache.New(l1cache.Config{
		CapacityBytes:        64 << 20, // 64MB
		CompressionThreshold: 256,
	})
	eng.AttachStores(l1, nil)
	
	const (
		NumGames     = 100
		MovesPerGame = 50
		Variations   = 10
	)
	
	// Record all snapshots
	snapshots := make([]types.SnapshotID, 0, NumGames*MovesPerGame)
	
	// Play all games
	for game := 0; game < NumGames; game++ {
		for move := 0; move < MovesPerGame; move++ {
			// Chess position (simplified)
			position := fmt.Sprintf("game_%d_move_%d", game, move)
			eng.WriteFile("position.fen", []byte(position))
			
			id, _, _ := eng.Commit(position)
			snapshots = append(snapshots, id)
			
			// Create variations
			for v := 0; v < Variations; v++ {
				variation := fmt.Sprintf("%s_var_%d", position, v)
				eng.WriteFile("variation.fen", []byte(variation))
				eng.Commit(variation)
				
				// Jump back to main line
				eng.Restore(id)
			}
		}
	}
	
	// Test time travel performance
	jumps := 1000
	start := time.Now()
	
	for i := 0; i < jumps; i++ {
		// Random jump to any position
		targetSnap := snapshots[rand.Intn(len(snapshots))]
		err := eng.Restore(targetSnap)
		if err != nil {
			t.Fatalf("Failed to restore snapshot: %v", err)
		}
	}
	
	elapsed := time.Since(start)
	avgJumpTime := elapsed / time.Duration(jumps)
	
	t.Logf("=== Time Travel Test Results ===")
	t.Logf("Total Positions: %d", len(snapshots))
	t.Logf("Random Jumps: %d", jumps)
	t.Logf("Avg Jump Time: %v", avgJumpTime)
	
	if avgJumpTime > 1*time.Microsecond {
		t.Errorf("Jump time %v exceeds target 1μs", avgJumpTime)
	}
}

// TestChaosResilience tests system under random failures
func TestChaosResilience(t *testing.T) {
	// This test randomly kills goroutines, corrupts data,
	// and verifies Merkle tree consistency
	t.Skip("Implement chaos testing framework")
}

// Helper function to sort latencies
func sortLatencies(latencies []int64) {
	// Simple insertion sort for demo
	for i := 1; i < len(latencies); i++ {
		key := latencies[i]
		j := i - 1
		for j >= 0 && latencies[j] > key {
			latencies[j+1] = latencies[j]
			j--
		}
		latencies[j+1] = key
	}
}
