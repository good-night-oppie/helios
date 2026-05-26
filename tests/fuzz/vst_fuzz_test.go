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


package fuzz

import (
	"os"
	"strings"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/vst"
	"unicode/utf8"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// sandboxCWD chdirs into t.TempDir() for the duration of the test to contain
// any disk writes from vst.WriteFile (see helios issue #42). Without this,
// fuzz inputs like "../y" and "a/b.go" would escape into the test's package
// directory and break subsequent `go list ./...` in CI.
func sandboxCWD(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func FuzzPathRoundTrip(f *testing.F) {
	seed := []string{"a.txt", "dir/b.txt", "weird_字符/空 白.md", "./x", "../y"}
	for _, s := range seed {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, path string) {
		sandboxCWD(t)
		// Skip obviously insane inputs to keep fuzz time short
		if path == "" || !utf8.ValidString(path) || len(path) > 2048 {
			t.Skip()
		}
		// normalize your own way if needed
		eng := vst.New()
		data := []byte("fuzz")
		_ = eng.WriteFile(path, data)
		id, _, err := eng.Commit("fuzz commit")
		if err != nil {
			t.Fatal(err)
		}

		// Use same engine instance to restore (snapshots are in memory)
		if err := eng.Restore(id); err != nil {
			t.Fatal(err)
		}

		got, err := eng.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "fuzz" {
			t.Fatalf("round-trip failed: %q", path)
		}
	})
}

func FuzzMaterializeSelectors(f *testing.F) {
	for _, s := range []string{"*.md", "**/*.go", "dir/**", "?.txt", "[ab]*"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, glob string) {
		sandboxCWD(t)
		if strings.Contains(glob, "\x00") || len(glob) > 256 {
			t.Skip()
		}
		eng := vst.New()
		_ = eng.WriteFile("a/a.md", []byte("m"))
		_ = eng.WriteFile("a/b.go", []byte("g"))
		_ = eng.WriteFile("root.txt", []byte("t"))
		id, _, _ := eng.Commit("fuzz materialize")

		// Use same engine instance (snapshots are in memory)
		if err := eng.Restore(id); err != nil {
			t.Fatal(err)
		}
		tmp := t.TempDir()
		opts := types.MatOpts{Include: []string{glob}}
		// Materialize should never panic or corrupt output
		_, _ = eng.Materialize(id, tmp, opts)
	})
}
