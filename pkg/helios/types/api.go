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


package types

import "time"

type SnapshotID string

type CommitMetrics struct {
	CommitLatency time.Duration
	NewObjects    int64
	NewBytes      int64
}

type DiffStats struct {
	Added   int
	Changed int
	Deleted int
}

type MatOpts struct {
	Include []string
	Exclude []string
}

type StateManager interface {
	Commit(msg string) (SnapshotID, CommitMetrics, error)
	Restore(id SnapshotID) error
	Diff(from, to SnapshotID) (DiffStats, error)
	Materialize(id SnapshotID, outDir string, opts MatOpts) (CommitMetrics, error)
}
