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

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// We only assert "shape", not long strings, to keep golden stable.
func TestCLI_CommitAndStats_Smoke(t *testing.T) {
	// Build a test binary next to sources
	bin := filepath.Join(t.TempDir(), "helios-cli.testbin")
	cmdBuild := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmdBuild.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, string(out))
	}

	// Run `commit` on an empty work dir; CLI should not panic and must print JSON
	work := t.TempDir()
	outC, err := exec.Command(bin, "commit", "--work", work).CombinedOutput()
	if err != nil {
		t.Fatalf("commit failed: %v\n%s", err, string(outC))
	}
	var jc map[string]any
	if err := json.Unmarshal(outC, &jc); err != nil {
		t.Fatalf("commit output is not JSON: %v\n%s", err, string(outC))
	}
	if _, ok := jc["snapshot_id"]; !ok {
		t.Fatalf("missing snapshot_id in commit output: %v", jc)
	}

	// `stats` should be valid JSON and contain a top-level "l1"
	outS, err := exec.Command(bin, "stats").CombinedOutput()
	if err != nil {
		t.Fatalf("stats failed: %v\n%s", err, string(outS))
	}
	var js map[string]any
	if err := json.Unmarshal(outS, &js); err != nil {
		t.Fatalf("stats output is not JSON: %v\n%s", err, string(outS))
	}
	if _, ok := js["l1"]; !ok {
		t.Fatalf("missing l1 in stats output: %v", js)
	}
}

func TestCLI_Help(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "helios-cli.testbin")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, string(out))
	}
	out, _ := exec.Command(bin, "-h").CombinedOutput()
	if len(out) == 0 {
		t.Fatalf("help should print usage")
	}
	_ = os.Stdout
}

func TestCLI_Version(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "helios-cli.testbin")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, string(out))
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\n%s", err, string(out))
	}
	if len(out) == 0 {
		t.Fatalf("version should print output")
	}
}
