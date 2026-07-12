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
)

// saveStoreEnv unsets HELIOS_STORE_DIR (or sets it to want) for the duration of
// a test and restores the prior value on cleanup. Passing want == "" unsets it.
func saveStoreEnv(t *testing.T, want string) {
	t.Helper()
	prev, had := os.LookupEnv("HELIOS_STORE_DIR")
	if want == "" {
		_ = os.Unsetenv("HELIOS_STORE_DIR")
	} else {
		_ = os.Setenv("HELIOS_STORE_DIR", want)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("HELIOS_STORE_DIR", prev)
		} else {
			_ = os.Unsetenv("HELIOS_STORE_DIR")
		}
	})
}

// saveCWD saves the working directory and restores it on cleanup. Manual
// save/restore is used deliberately (no t.Chdir — go.mod pins go 1.22, and
// t.Chdir was added in go 1.24).
func saveCWD(t *testing.T) string {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return old
}

// snapshotIDFromJSON json-decodes commit output and returns snapshot_id. The
// README's `.decode().strip()` idiom is buggy — parse the JSON.
func snapshotIDFromJSON(t *testing.T, b []byte) string {
	t.Helper()
	var res map[string]any
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("commit output is not JSON (%q): %v", string(b), err)
	}
	id, _ := res["snapshot_id"].(string)
	if id == "" {
		t.Fatalf("no snapshot_id in commit output: %q", string(b))
	}
	return id
}

// TestStoreResolution_CommitReadableByMaterialize proves that a snapshot
// committed with `--work W` from an invocation directory C is readable by a
// later `materialize` invoked from the same C, with HELIOS_STORE_DIR unset.
//
// Root cause (pre-fix): HandleCommit chdir'd into workDir before the store was
// resolved, so `commit --work W` wrote the store under W/.helios/objects, while
// materialize (which never chdir's) resolved it under C/.helios/objects — the
// two disagreed and materialize failed with "unknown snapshot in L2".
//
// The test re-establishes CWD=C before materialize to faithfully simulate the
// separate CLI processes (each starts from the shell's invocation dir C).
func TestStoreResolution_CommitReadableByMaterialize(t *testing.T) {
	saveStoreEnv(t, "") // HELIOS_STORE_DIR must be UNSET or both paths agree and the bug hides
	saveCWD(t)

	c := t.TempDir() // invocation dir (shell CWD)
	w := t.TempDir() // --work dir with content
	if err := os.WriteFile(filepath.Join(w, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("seed work file: %v", err)
	}

	cfg := Config{EngineFactory: DefaultEngineFactory} // REAL factory — the fake never touches the store

	if err := os.Chdir(c); err != nil {
		t.Fatalf("chdir invocation dir: %v", err)
	}
	var commitBuf bytes.Buffer
	if err := HandleCommit(&commitBuf, cfg, w); err != nil {
		t.Fatalf("commit: %v", err)
	}
	id := snapshotIDFromJSON(t, commitBuf.Bytes())

	// A fresh CLI process would start again from C, so reset CWD to C before
	// materialize. On pre-fix code HandleCommit left CWD in W; on fixed code it
	// never moved — either way materialize must resolve the store from C.
	if err := os.Chdir(c); err != nil {
		t.Fatalf("re-chdir invocation dir: %v", err)
	}
	out := filepath.Join(t.TempDir(), "out")
	var matBuf bytes.Buffer
	if err := HandleMaterialize(&matBuf, cfg, id, out, MatOpts{}); err != nil {
		// Pre-fix this fails with "unknown snapshot in L2".
		t.Fatalf("materialize could not read commit (store-resolution bug): %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "hello.txt")); err != nil {
		t.Fatalf("hello.txt not materialized into out dir: %v", err)
	}
}

// TestStoreResolution_HonorsEnvVar is the regression guard the integration pins
// on: with HELIOS_STORE_DIR set, commit and materialize resolve to that store
// regardless of CWD, so a commit from one dir is materializable from another.
// This must keep passing on the fixed code (THINKING constraint 6).
func TestStoreResolution_HonorsEnvVar(t *testing.T) {
	store := t.TempDir()
	saveStoreEnv(t, store)
	saveCWD(t)

	c := t.TempDir()
	w := t.TempDir()
	if err := os.WriteFile(filepath.Join(w, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("seed work file: %v", err)
	}

	cfg := Config{EngineFactory: DefaultEngineFactory}

	if err := os.Chdir(c); err != nil {
		t.Fatalf("chdir commit dir: %v", err)
	}
	var commitBuf bytes.Buffer
	if err := HandleCommit(&commitBuf, cfg, w); err != nil {
		t.Fatalf("commit: %v", err)
	}
	id := snapshotIDFromJSON(t, commitBuf.Bytes())

	// Materialize from a DIFFERENT cwd; the env-pinned store must still resolve.
	c2 := t.TempDir()
	if err := os.Chdir(c2); err != nil {
		t.Fatalf("chdir materialize dir: %v", err)
	}
	out := filepath.Join(t.TempDir(), "out")
	var matBuf bytes.Buffer
	if err := HandleMaterialize(&matBuf, cfg, id, out, MatOpts{}); err != nil {
		t.Fatalf("materialize with HELIOS_STORE_DIR set failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "hello.txt")); err != nil {
		t.Fatalf("hello.txt not materialized with pinned store: %v", err)
	}
}
