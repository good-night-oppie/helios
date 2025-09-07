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
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestHandleCommit_Golden(t *testing.T) {
	fake := &FakeEngine{
		commitResult: "abc123def456",
		commitMetrics: types.CommitMetrics{
			NewObjects: 3,
			NewBytes:   1024,
		},
	}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}
	buf := &bytes.Buffer{}
	if err := HandleCommit(buf, cfg, ""); err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, "commit_basic", buf.Bytes(), *updateGolden)
}

func TestHandleRestore_Golden(t *testing.T) {
	fake := &FakeEngine{}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}
	buf := &bytes.Buffer{}
	if err := HandleRestore(buf, cfg, "abc123def456"); err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, "restore_basic", buf.Bytes(), *updateGolden)
}

func TestHandleDiff_Golden(t *testing.T) {
	fake := &FakeEngine{
		diffResult: types.DiffStats{
			Added:   2,
			Changed: 1,
			Deleted: 3,
		},
	}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}
	buf := &bytes.Buffer{}
	if err := HandleDiff(buf, cfg, "abc123", "def456"); err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, "diff_basic", buf.Bytes(), *updateGolden)
}

func TestHandleMaterialize_Golden(t *testing.T) {
	fake := &FakeEngine{}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}
	buf := &bytes.Buffer{}
	opts := MatOpts{
		Include: []string{"*.go"},
		Exclude: []string{"*_test.go"},
	}
	if err := HandleMaterialize(buf, cfg, "abc123def456", "/tmp/output", opts); err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, "materialize_with_opts", buf.Bytes(), *updateGolden)
}

func TestHandleStats_Golden(t *testing.T) {
	fake := &FakeEngine{
		l1Stats: l1cache.CacheStats{
			Hits:      42,
			Misses:    13,
			Evictions: 5,
			SizeBytes: 2048,
			Items:     37,
		},
	}
	cfg := Config{
		EngineFactory: func() (Engine, error) { return fake, nil },
	}
	buf := &bytes.Buffer{}
	if err := HandleStats(buf, cfg); err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, "stats_basic", buf.Bytes(), *updateGolden)
}

func assertJSONGolden(t *testing.T, name string, got []byte, update bool) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "spec", "golden", "cli", name+".golden.json")
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
