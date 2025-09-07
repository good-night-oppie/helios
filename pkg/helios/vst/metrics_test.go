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
	"testing"
)

// This test asserts that after a real Commit(), engine metrics snapshot reflects activity.
func TestEngineMetrics_AfterCommit(t *testing.T) {
	e := New()

	// Write a tiny file and commit once
	if err := e.WriteFile("a.txt", []byte("x")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _, err := e.Commit("test commit")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	snap := e.EngineMetricsSnapshot()
	if snap.P50 == 0 && snap.P95 == 0 && snap.P99 == 0 {
		t.Fatalf("expect commit latency percentiles to reflect activity, got: %+v", snap)
	}
	// new_objects/new_bytes should be >0 for a first commit with at least one blob
	if snap.NewObjects == 0 || snap.NewBytes == 0 {
		t.Fatalf("expect non-zero new objects/bytes, got: %+v", snap)
	}
}
