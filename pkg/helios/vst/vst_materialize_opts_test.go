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
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

func TestVST_Materialize_WithSelectors(t *testing.T) {
	v := New()
	_ = v.WriteFile("src/a.go", []byte("a"))
	_ = v.WriteFile("src/b.go", []byte("b"))
	_ = v.WriteFile("docs/readme.md", []byte("# hi"))
	id, _, err := v.Commit("with-selectors")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	tmp, _ := os.MkdirTemp("", "helios-mat-*")
	defer os.RemoveAll(tmp)

	// Include only src/**
	opts := types.MatOpts{Include: []string{"src/**"}}
	if _, err := v.Materialize(id, tmp, opts); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "src/a.go")); err != nil {
		t.Fatalf("want src/a.go materialized")
	}
	if _, err := os.Stat(filepath.Join(tmp, "docs/readme.md")); !os.IsNotExist(err) {
		t.Fatalf("docs/readme.md should be excluded")
	}

	// Exclude docs/**
	opts = types.MatOpts{Exclude: []string{"docs/**"}}
	os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := v.Materialize(id, tmp, opts); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "docs/readme.md")); !os.IsNotExist(err) {
		t.Fatalf("docs/readme.md should be excluded")
	}
}
