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
