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
	"sync"
	"time"
)

// EngineMetrics collects minimal metrics for Helios Day 8.
// Keep it tiny and lock-based to avoid allocation-heavy deps.
type EngineMetrics struct {
	mu sync.Mutex

	commitUS   []int64 // microseconds for commits (append-only)
	newObjects uint64
	newBytes   uint64
}

func NewEngineMetrics() *EngineMetrics {
	return &EngineMetrics{
		commitUS: make([]int64, 0, 1024),
	}
}

func (m *EngineMetrics) ObserveCommitLatency(d time.Duration) {
	m.mu.Lock()
	m.commitUS = append(m.commitUS, d.Microseconds())
	m.mu.Unlock()
}

func (m *EngineMetrics) AddNewObjects(n uint64) {
	if n == 0 {
		return
	}
	m.mu.Lock()
	m.newObjects += n
	m.mu.Unlock()
}

func (m *EngineMetrics) AddNewBytes(n uint64) {
	if n == 0 {
		return
	}
	m.mu.Lock()
	m.newBytes += n
	m.mu.Unlock()
}

type Snapshot struct {
	P50        int64  `json:"commit_latency_us_p50"`
	P95        int64  `json:"commit_latency_us_p95"`
	P99        int64  `json:"commit_latency_us_p99"`
	NewObjects uint64 `json:"new_objects"`
	NewBytes   uint64 `json:"new_bytes"`
}

// Snapshot returns a percentile summary + counters.
// Percentiles are computed via quickselect on a copy to avoid mutating the series.
func (m *EngineMetrics) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	p50 := percentile(m.commitUS, 0.50)
	p95 := percentile(m.commitUS, 0.95)
	p99 := percentile(m.commitUS, 0.99)

	return Snapshot{
		P50:        p50,
		P95:        p95,
		P99:        p99,
		NewObjects: m.newObjects,
		NewBytes:   m.newBytes,
	}
}

func percentile(series []int64, p float64) int64 {
	if len(series) == 0 {
		return 0
	}
	cp := make([]int64, len(series))
	copy(cp, series)
	k := int(float64(len(cp)-1) * p)
	quickselect(cp, 0, len(cp)-1, k)
	return cp[k]
}

func quickselect(a []int64, l, r, k int) {
	for l < r {
		p := partition(a, l, r)
		if k == p {
			return
		} else if k < p {
			r = p - 1
		} else {
			l = p + 1
		}
	}
}

func partition(a []int64, l, r int) int {
	p := a[r]
	i := l
	for j := l; j < r; j++ {
		if a[j] < p {
			a[i], a[j] = a[j], a[i]
			i++
		}
	}
	a[i], a[r] = a[r], a[i]
	return i
}
