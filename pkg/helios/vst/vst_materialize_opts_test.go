package vst

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
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
