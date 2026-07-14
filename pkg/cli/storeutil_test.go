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
	"os"
	"path/filepath"
	"testing"
)

// withStoreEnv sets or unsets HELIOS_STORE_DIR for the test and restores it.
func withStoreEnv(t *testing.T, val string, set bool) {
	t.Helper()
	prev, had := os.LookupEnv("HELIOS_STORE_DIR")
	if set {
		_ = os.Setenv("HELIOS_STORE_DIR", val)
	} else {
		_ = os.Unsetenv("HELIOS_STORE_DIR")
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("HELIOS_STORE_DIR", prev)
		} else {
			_ = os.Unsetenv("HELIOS_STORE_DIR")
		}
	})
}

// TestResolveStore_HonorsEnvVar: when HELIOS_STORE_DIR is set, ResolveStore
// returns exactly that path (created if missing) and ignores cwd. This is the
// pinned-store contract the AI-Scientist integration depends on.
func TestResolveStore_HonorsEnvVar(t *testing.T) {
	want := filepath.Join(t.TempDir(), "pinned", "store")
	withStoreEnv(t, want, true)

	got, err := ResolveStore(t.TempDir()) // cwd arg must be ignored when env is set
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}
	if got != want {
		t.Fatalf("ResolveStore = %q, want env value %q", got, want)
	}
	if fi, err := os.Stat(got); err != nil || !fi.IsDir() {
		t.Fatalf("ResolveStore did not create the store dir %q: err=%v", got, err)
	}
}

// TestResolveStore_DefaultUnderCWD: with HELIOS_STORE_DIR unset, ResolveStore
// falls back to <cwd>/.helios/objects and creates it.
func TestResolveStore_DefaultUnderCWD(t *testing.T) {
	withStoreEnv(t, "", false)

	cwd := t.TempDir()
	got, err := ResolveStore(cwd)
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}
	want := filepath.Join(cwd, ".helios", "objects")
	if got != want {
		t.Fatalf("ResolveStore = %q, want default %q", got, want)
	}
	if fi, err := os.Stat(got); err != nil || !fi.IsDir() {
		t.Fatalf("ResolveStore did not create the default store dir %q: err=%v", got, err)
	}
}
