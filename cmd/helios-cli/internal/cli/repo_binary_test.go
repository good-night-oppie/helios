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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// elfMagic is the 4-byte magic number at the start of every ELF executable.
const elfMagic = "\x7fELF"

// gitTopLevel returns the absolute path of the repository root by shelling out
// to `git rev-parse --show-toplevel`.
func gitTopLevel(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("not a git checkout (git rev-parse failed: %v)", err)
	}
	return strings.TrimSpace(string(out))
}

// gitLsFilesZ returns the NUL-separated list of tracked files from the repo
// root using `git ls-files -z`.
func gitLsFilesZ(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files failed: %v", err)
	}
	return string(out)
}

// TestNoTrackedBinaries is a repo-hygiene guard: no ELF executable may be
// committed to the repository. Prebuilt CLI binaries (helios-cli, bin/helios,
// dist/helios) are build artifacts and must be produced from HEAD
// (`go build ./cmd/helios-cli`), never tracked — a stale tracked binary is a
// silent-data-loss trap for anyone who runs it. This test is general: it
// catches any future committed binary, not just the three known ones.
func TestNoTrackedBinaries(t *testing.T) {
	root := gitTopLevel(t)
	out := gitLsFilesZ(t, root)

	for _, rel := range strings.Split(strings.TrimRight(out, "\x00"), "\x00") {
		if rel == "" {
			continue
		}
		f, err := os.Open(filepath.Join(root, rel))
		if err != nil {
			// Tracked-but-absent on disk: nothing to inspect.
			continue
		}
		var magic [4]byte
		n, _ := io.ReadFull(f, magic[:])
		f.Close()
		if n == 4 && string(magic[:]) == elfMagic {
			t.Errorf("tracked ELF binary must not be committed: %s", rel)
		}
	}
}
