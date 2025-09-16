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

// SPDX-License-Identifier: MIT

package stress

import "time"

// MCTSMetrics represents metrics collected during MCTS stress testing
type MCTSMetrics struct {
	NodesExplored   int64     `json:"nodes_explored"`
	TreeDepth       int       `json:"tree_depth"`
	SelectionTime   time.Duration `json:"selection_time"`
	ExpansionTime   time.Duration `json:"expansion_time"`
	SimulationTime  time.Duration `json:"simulation_time"`
	BackpropTime    time.Duration `json:"backprop_time"`
	TotalIterations int64     `json:"total_iterations"`
	WinRate         float64   `json:"win_rate"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewMCTSMetrics creates a new MCTSMetrics instance
func NewMCTSMetrics() *MCTSMetrics {
	return &MCTSMetrics{
		Timestamp: time.Now(),
	}
}

// UpdateMetrics updates the metrics with new values
func (m *MCTSMetrics) UpdateMetrics(nodes int64, depth int, iterations int64, winRate float64) {
	m.NodesExplored = nodes
	m.TreeDepth = depth
	m.TotalIterations = iterations
	m.WinRate = winRate
	m.Timestamp = time.Now()
}

// GetMetrics returns the current metrics
func (m *MCTSMetrics) GetMetrics() MCTSMetrics {
	return *m
}

// GenerateDemoMetrics creates demo metrics for visualization
func GenerateDemoMetrics() *MCTSMetrics {
	metrics := NewMCTSMetrics()
	metrics.UpdateMetrics(1500, 8, 2500, 0.72)
	return metrics
}