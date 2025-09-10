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


package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios/internal/metrics"
	"github.com/good-night-oppie/helios/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios/pkg/helios/objstore"
	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// FakeEngine implements Engine interface for testing
type FakeEngine struct {
	commitResult     types.SnapshotID
	commitMetrics    types.CommitMetrics
	commitError      error
	restoreError     error
	diffResult       types.DiffStats
	diffError        error
	materializeError error
	l1Stats          l1cache.CacheStats
}

func (f *FakeEngine) AttachStores(l1cache.Cache, objstore.Store) {}

func (f *FakeEngine) WriteFile(path string, content []byte) error {
	return nil
}

func (f *FakeEngine) Commit(msg string) (types.SnapshotID, types.CommitMetrics, error) {
	return f.commitResult, f.commitMetrics, f.commitError
}

func (f *FakeEngine) Restore(id types.SnapshotID) error {
	return f.restoreError
}

func (f *FakeEngine) Diff(from, to types.SnapshotID) (types.DiffStats, error) {
	return f.diffResult, f.diffError
}

func (f *FakeEngine) Materialize(id types.SnapshotID, outDir string, opts types.MatOpts) (types.CommitMetrics, error) {
	return types.CommitMetrics{}, f.materializeError
}

func (f *FakeEngine) L1Stats() l1cache.CacheStats {
	return f.l1Stats
}

func (f *FakeEngine) EngineMetricsSnapshot() metrics.Snapshot {
	return metrics.Snapshot{}
}

func TestHandleCommit(t *testing.T) {
	tests := []struct {
		name     string
		workDir  string
		fake     *FakeEngine
		wantErr  bool
		contains []string
	}{
		{
			name:    "success",
			workDir: "",
			fake: &FakeEngine{
				commitResult: "abc123",
			},
			contains: []string{"abc123", "snapshot_id"},
		},
		{
			name:    "commit error",
			workDir: "",
			fake: &FakeEngine{
				commitError: testError("commit failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				EngineFactory: func() (Engine, error) { return tt.fake, nil },
			}

			buf := &bytes.Buffer{}
			err := HandleCommit(buf, cfg, tt.workDir)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			for _, want := range tt.contains {
				found := false
				for k, v := range result {
					if k == want || (v != nil && v.(string) == want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("output %v should contain %q", result, want)
				}
			}
		})
	}
}

func TestHandleRestore(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		fake    *FakeEngine
		wantErr bool
	}{
		{
			name: "success",
			id:   "abc123",
			fake: &FakeEngine{},
		},
		{
			name:    "missing id",
			id:      "",
			fake:    &FakeEngine{},
			wantErr: true,
		},
		{
			name: "restore error",
			id:   "abc123",
			fake: &FakeEngine{
				restoreError: testError("restore failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				EngineFactory: func() (Engine, error) { return tt.fake, nil },
			}

			buf := &bytes.Buffer{}
			err := HandleRestore(buf, cfg, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if result["restored"] != tt.id {
				t.Errorf("got restored=%v, want %v", result["restored"], tt.id)
			}
		})
	}
}

func TestHandleDiff(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		fake    *FakeEngine
		wantErr bool
	}{
		{
			name: "success",
			from: "abc123",
			to:   "def456",
			fake: &FakeEngine{
				diffResult: types.DiffStats{Added: 1, Changed: 2, Deleted: 3},
			},
		},
		{
			name:    "missing from",
			from:    "",
			to:      "def456",
			fake:    &FakeEngine{},
			wantErr: true,
		},
		{
			name:    "missing to",
			from:    "abc123",
			to:      "",
			fake:    &FakeEngine{},
			wantErr: true,
		},
		{
			name: "diff error",
			from: "abc123",
			to:   "def456",
			fake: &FakeEngine{
				diffError: testError("diff failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				EngineFactory: func() (Engine, error) { return tt.fake, nil },
			}

			buf := &bytes.Buffer{}
			err := HandleDiff(buf, cfg, tt.from, tt.to)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result types.DiffStats
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if result != tt.fake.diffResult {
				t.Errorf("got %+v, want %+v", result, tt.fake.diffResult)
			}
		})
	}
}

func TestHandleMaterialize(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		outDir  string
		opts    MatOpts
		fake    *FakeEngine
		wantErr bool
	}{
		{
			name:   "success",
			id:     "abc123",
			outDir: "/tmp/out",
			fake:   &FakeEngine{},
		},
		{
			name:    "missing id",
			id:      "",
			outDir:  "/tmp/out",
			fake:    &FakeEngine{},
			wantErr: true,
		},
		{
			name:    "missing outDir",
			id:      "abc123",
			outDir:  "",
			fake:    &FakeEngine{},
			wantErr: true,
		},
		{
			name:   "materialize error",
			id:     "abc123",
			outDir: "/tmp/out",
			fake: &FakeEngine{
				materializeError: testError("materialize failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				EngineFactory: func() (Engine, error) { return tt.fake, nil },
			}

			buf := &bytes.Buffer{}
			err := HandleMaterialize(buf, cfg, tt.id, tt.outDir, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if result["materialized"] != tt.id {
				t.Errorf("got materialized=%v, want %v", result["materialized"], tt.id)
			}
			if result["out"] != tt.outDir {
				t.Errorf("got out=%v, want %v", result["out"], tt.outDir)
			}
		})
	}
}

func TestHandleStats(t *testing.T) {
	tests := []struct {
		name string
		fake *FakeEngine
	}{
		{
			name: "success",
			fake: &FakeEngine{
				l1Stats: l1cache.CacheStats{
					Hits:      10,
					Misses:    5,
					Evictions: 2,
					SizeBytes: 1024,
					Items:     8,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				EngineFactory: func() (Engine, error) { return tt.fake, nil },
			}

			buf := &bytes.Buffer{}
			err := HandleStats(buf, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			l1, ok := result["l1"].(map[string]any)
			if !ok {
				t.Fatal("missing or invalid l1 field")
			}

			expected := tt.fake.l1Stats
			if uint64(l1["hits"].(float64)) != expected.Hits {
				t.Errorf("got hits=%v, want %v", l1["hits"], expected.Hits)
			}
			if uint64(l1["misses"].(float64)) != expected.Misses {
				t.Errorf("got misses=%v, want %v", l1["misses"], expected.Misses)
			}
		})
	}
}

func TestDefaultEngineFactory(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	eng, err := DefaultEngineFactory()
	if err != nil {
		t.Fatalf("DefaultEngineFactory failed: %v", err)
	}
	if eng == nil {
		t.Fatal("expected non-nil engine")
	}

	// Verify .helios directory was created
	heliosDir := filepath.Join(tmpDir, ".helios", "objects")
	if _, err := os.Stat(heliosDir); os.IsNotExist(err) {
		t.Fatal("expected .helios/objects directory to be created")
	}

	// Test that L1 stats work
	stats := eng.L1Stats()
	// Items is uint64, so this check is always true, but kept for documentation
	_ = stats.Items
}

func TestEngineFactoryError(t *testing.T) {
	cfg := Config{
		EngineFactory: func() (Engine, error) {
			return nil, testError("engine factory failed")
		},
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{"commit", func() error { return HandleCommit(&bytes.Buffer{}, cfg, "") }},
		{"restore", func() error { return HandleRestore(&bytes.Buffer{}, cfg, "test") }},
		{"diff", func() error { return HandleDiff(&bytes.Buffer{}, cfg, "a", "b") }},
		{"materialize", func() error { return HandleMaterialize(&bytes.Buffer{}, cfg, "test", "/tmp", MatOpts{}) }},
		{"stats", func() error { return HandleStats(&bytes.Buffer{}, cfg) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected engine factory error")
			}
			if err.Error() != "engine factory failed" {
				t.Errorf("got error %v, want 'engine factory failed'", err)
			}
		})
	}
}

func TestMaterializeWithIncludeExclude(t *testing.T) {
	fake := &FakeEngine{}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}

	opts := MatOpts{
		Include: []string{"*.go", "*.md"},
		Exclude: []string{"*_test.go"},
	}

	buf := &bytes.Buffer{}
	err := HandleMaterialize(buf, cfg, "test-id", "/tmp/out", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["materialized"] != "test-id" {
		t.Errorf("got materialized=%v, want test-id", result["materialized"])
	}
}

func TestHandleCommitWithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	fake := &FakeEngine{
		commitResult: "workdir-test",
	}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}

	buf := &bytes.Buffer{}
	err := HandleCommit(buf, cfg, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["snapshot_id"] != "workdir-test" {
		t.Errorf("got snapshot_id=%v, want workdir-test", result["snapshot_id"])
	}
}

type testError string

func (e testError) Error() string {
	return string(e)
}
