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


package metrics

import (
	"testing"
	"time"
)

func TestEngineMetrics_BasicFlow(t *testing.T) {
	m := NewEngineMetrics()

	// Should start with zeros
	snap := m.Snapshot()
	if snap.P50 != 0 || snap.P95 != 0 || snap.P99 != 0 {
		t.Errorf("expected zeros for empty metrics, got %+v", snap)
	}
	if snap.NewObjects != 0 || snap.NewBytes != 0 {
		t.Errorf("expected zero counters, got %+v", snap)
	}

	// Add some commit latencies
	m.ObserveCommitLatency(100 * time.Microsecond)
	m.ObserveCommitLatency(200 * time.Microsecond)
	m.ObserveCommitLatency(300 * time.Microsecond)
	m.ObserveCommitLatency(400 * time.Microsecond)
	m.ObserveCommitLatency(500 * time.Microsecond)

	// Add objects and bytes
	m.AddNewObjects(10)
	m.AddNewBytes(1024)
	m.AddNewObjects(5)
	m.AddNewBytes(512)

	// Check snapshot
	snap = m.Snapshot()

	// P50 should be around 300 (median of 100,200,300,400,500)
	if snap.P50 != 300 {
		t.Errorf("expected P50=300, got %d", snap.P50)
	}

	// P95 should be 400 or 500 (95th percentile of 5 samples)
	if snap.P95 != 400 && snap.P95 != 500 {
		t.Errorf("expected P95=400 or 500, got %d", snap.P95)
	}

	// P99 should be 400 or 500 (99th percentile for 5 samples)
	if snap.P99 != 400 && snap.P99 != 500 {
		t.Errorf("expected P99=400 or 500, got %d", snap.P99)
	}

	// Check counters
	if snap.NewObjects != 15 {
		t.Errorf("expected NewObjects=15, got %d", snap.NewObjects)
	}
	if snap.NewBytes != 1536 {
		t.Errorf("expected NewBytes=1536, got %d", snap.NewBytes)
	}
}

func TestEngineMetrics_EdgeCases(t *testing.T) {
	m := NewEngineMetrics()

	// Test adding zero values (should be no-op)
	m.AddNewObjects(0)
	m.AddNewBytes(0)

	snap := m.Snapshot()
	if snap.NewObjects != 0 || snap.NewBytes != 0 {
		t.Errorf("adding zero should be no-op, got %+v", snap)
	}

	// Test single latency observation
	m.ObserveCommitLatency(42 * time.Microsecond)
	snap = m.Snapshot()

	// All percentiles should be the same for single value
	if snap.P50 != 42 || snap.P95 != 42 || snap.P99 != 42 {
		t.Errorf("single value should give same percentiles, got P50=%d, P95=%d, P99=%d",
			snap.P50, snap.P95, snap.P99)
	}
}

func TestPercentile_VariousSizes(t *testing.T) {
	tests := []struct {
		name   string
		series []int64
		p      float64
		want   int64
	}{
		{
			name:   "empty",
			series: []int64{},
			p:      0.5,
			want:   0,
		},
		{
			name:   "single",
			series: []int64{100},
			p:      0.5,
			want:   100,
		},
		{
			name:   "two_p50",
			series: []int64{100, 200},
			p:      0.5,
			want:   100,
		},
		{
			name:   "odd_count_p50",
			series: []int64{1, 2, 3, 4, 5},
			p:      0.5,
			want:   3,
		},
		{
			name:   "even_count_p50",
			series: []int64{1, 2, 3, 4, 5, 6},
			p:      0.5,
			want:   3,
		},
		{
			name:   "p99_small",
			series: []int64{1, 2, 3, 4, 5},
			p:      0.99,
			want:   4, // For 5 samples, index is int(4*0.99) = 3, which is 4
		},
		{
			name:   "unsorted",
			series: []int64{5, 1, 4, 2, 3},
			p:      0.5,
			want:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.series, tt.p)
			if got != tt.want {
				t.Errorf("percentile(%v, %.2f) = %d, want %d",
					tt.series, tt.p, got, tt.want)
			}
		})
	}
}
